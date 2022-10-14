package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"strconv"
	"time"

	"net/http"
	"net/url"

	firebase "firebase.google.com/go"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/minio/minio-go/v7"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opencensus.io/trace"

	"github.com/kuZzzzia/access_control_app/backend/pagination"
	"github.com/kuZzzzia/access_control_app/backend/service"
	"github.com/kuZzzzia/access_control_app/backend/specs"
	"github.com/kuZzzzia/access_control_app/backend/storage/postgres"
)

type Controller struct {
	srv  *service.Service
	repo *postgres.Repo

	DenyTypes map[string]string
	SizeLimit int64

	clients map[*websocket.Conn]bool

	lastPeopleNumber int
	// peopleNumberNotification chan int

	HTTPClient http.Client

	FireBaseUrl   *url.URL
	FirebaseToken string
	App           *firebase.App
}

func NewController(srv *service.Service,
	repo *postgres.Repo,
	denyTypes map[string]string,
	sizeLimit int64, app *firebase.App) *Controller {

	return &Controller{
		srv:       srv,
		repo:      repo,
		DenyTypes: denyTypes,
		SizeLimit: sizeLimit,

		HTTPClient: http.Client{},

		clients: make(map[*websocket.Conn]bool),
		// peopleNumberNotification: make(chan int, 2),
		App: app,
	}
}

var _ specs.ServerInterface = &Controller{}

const layout = "02-01-2006_15:04:05"

func (ctrl *Controller) CreateImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var err error

	if r.Body == nil {
		log.Warn().Msg("nil body")
		withError(ctx, w, http.StatusBadRequest, "nil body")
		return
	}

	file, fileHeader, err := r.FormFile("img")
	if err != nil {
		log.Error().Err(err).Msg("err FormFile")
		err = r.MultipartForm.RemoveAll()
		if err != nil {
			log.Error().Err(err).Msg("unable to remove form")
		}
		withError(ctx, w, http.StatusBadRequest, "form file 'upfile' not found")
		return
	}
	defer func() {
		errClose := file.Close()
		if errClose != nil {
			log.Error().Err(errClose).Msg("unable to close form file")
		}
		err = r.MultipartForm.RemoveAll()
		if err != nil {
			log.Error().Err(err).Msg("unable to remove form")
		}
	}()

	people_number := r.FormValue("people_number")
	pn, err := strconv.Atoi(people_number)
	if err != nil {
		log.Warn().Err(err).Msg("parse people_number")
		withError(ctx, w, http.StatusBadRequest, "can't parse people_number")
		return
	}

	createdAt := r.FormValue("created_at")
	createdAtTime, err := time.Parse(layout, createdAt)
	if err != nil {
		log.Warn().Err(err).Msg("parse created_at")
		withError(ctx, w, http.StatusBadRequest, "can't parse created_at")
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")

	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		log.Warn().Err(err).Msg("parse media type")
		withError(ctx, w, http.StatusBadRequest, "can't parse media type")
		return
	}

	extension, ok := ctrl.DenyTypes[mt]
	if ok {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	// bytes
	if r.ContentLength <= 0 || r.ContentLength > ctrl.SizeLimit {
		log.Warn().Int64("Content-Length", r.ContentLength).Int64("max size limit", ctrl.SizeLimit).
			Msg("incorrect Content-Length")
		withError(ctx, w, http.StatusBadRequest, "incorrect Content-Length")
		return
	}
	log.Info().Int64("Content-Length", r.ContentLength).Msg("parse Content-Length successful")

	object := &service.ImageInfo{
		ID:        uuid.New(),
		CreatedAt: createdAtTime,

		PeopleNumber: pn,

		ContentType: contentType,
		Extension:   extension,
		Size:        fileHeader.Size,
		Name:        fileHeader.Filename,
		// TODO:
		UserID: uuid.MustParse("b07fd4a1-6f73-4541-949d-b8a97b3d2c04"),
	}

	if object.Name == "" {
		object.Name = object.ID.String()
	}

	repo, err := ctrl.repo.BeginTx(ctx)
	if err != nil {
		log.Error().Err(err).Msg("begin tx")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	shouldRollback := true
	defer repo.Rollback(ctx, &shouldRollback)

	objectService := ctrl.srv.WithNewRepo(repo)

	err = objectService.CreateObject(ctx, file, object)
	if err != nil {
		log.Error().Err(err).Msg("create object")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = repo.Commit()
	if err != nil {
		log.Error().Err(err).Msg("commit")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	shouldRollback = false

	go func() {
		if ctrl.lastPeopleNumber != object.PeopleNumber && object.PeopleNumber != 0 {
			ctrl.lastPeopleNumber = object.PeopleNumber
			// ctrl.peopleNumberNotification <- object.PeopleNumber
			ctrl.PushPeopleNumber(context.Background(), object.PeopleNumber)
		}
	}()

	res, err := ctrl.srv.GetObjectInfo(ctx, object.ID)
	if err != nil {
		log.Error().Err(err).Msg("return resume info")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	withJSON(ctx, w, http.StatusOK, GetImageForResponse(res))
}

func GetImageForResponse(res *service.ImageInfo) *specs.GetImageInfoResponse {
	return &specs.GetImageInfoResponse{
		Id:           res.ID.String(),
		CreatedAt:    res.CreatedAt,
		PeopleNumber: res.PeopleNumber,
	}
}

func (ctrl *Controller) GetImage(w http.ResponseWriter, r *http.Request, imageId string) {
	ctx := r.Context()

	span := trace.FromContext(ctx)
	defer span.End()

	objectID, err := uuid.Parse(imageId)
	if err != nil {
		log.Warn().Err(err).Msg("invalid uuid")
		withError(ctx, w, http.StatusBadRequest, "invalid image id")
		return
	}

	file, object, err := ctrl.srv.GetObject(ctx, objectID)
	if err != nil {
		log.Error().Err(err).Msg("get object")
		switch {
		case errors.As(err, &service.ErrorObjectNotFound):
			withError(ctx, w, http.StatusNotFound, "object not found")
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer, err = createMiltuPartImage(writer, file, object)
	if err != nil {
		writer.Close()
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Close()

	w.Header().Set("Content-Type", writer.FormDataContentType())
	_, err = io.Copy(w, body)
	if err != nil {
		log.Error().Err(err).Msg("return obj")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (ctrl *Controller) GetImageInfo(w http.ResponseWriter, r *http.Request, imageId string) {
	ctx := r.Context()

	span := trace.FromContext(ctx)
	defer span.End()
	entry := zerolog.Ctx(ctx)

	objectID, err := uuid.Parse(imageId)
	if err != nil {
		entry.Warn().Err(err).Msg("invalid uuid")
		withError(ctx, w, http.StatusBadRequest, "invalid image id")
		return
	}

	object, err := ctrl.srv.GetObjectInfo(ctx, objectID)
	if err != nil {
		entry.Error().Err(err).Msg("get object")
		switch {
		case errors.As(err, &service.ErrorObjectNotFound):
			withError(ctx, w, http.StatusNotFound, "object not found")
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	withJSON(ctx, w, http.StatusOK, GetImageForResponse(object))
}

func (ctrl *Controller) GetLastImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	span := trace.FromContext(ctx)
	defer span.End()

	file, object, err := ctrl.srv.GetLastObject(ctx)
	if err != nil {
		log.Error().Err(err).Msg("get object")
		switch {
		case errors.As(err, &service.ErrorObjectNotFound):
			withError(ctx, w, http.StatusNotFound, "object not found")
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer, err = createMiltuPartImage(writer, file, object)
	if err != nil {
		writer.Close()
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Close()

	w.Header().Set("Content-Type", writer.FormDataContentType())
	_, err = io.Copy(w, body)
	if err != nil {
		log.Error().Err(err).Msg("return obj")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func createMiltuPartImage(writer *multipart.Writer, file *minio.Object, object *service.ImageInfo) (*multipart.Writer, error) {
	fw, err := writer.CreateFormField("info")
	if err != nil {
		log.Error().Err(err).Msg("CreateFormFile info")

	}

	err = json.NewEncoder(fw).Encode(GetImageForResponse(object))
	if err != nil {
		log.Error().Err(err).Msg("json encode object")
		return writer, err
	}

	fw, err = writer.CreateFormFile("img", object.Name)
	if err != nil {
		log.Error().Err(err).Msg("createFormFile img")
		return writer, err
	}

	_, err = io.Copy(fw, file)
	if err != nil {
		log.Error().Err(err).Msg("copy obj")
		return writer, err
	}

	return writer, nil
}

func (ctrl *Controller) DeleteImage(w http.ResponseWriter, r *http.Request, imageId string) {
	ctx := r.Context()

	span := trace.FromContext(ctx)
	defer span.End()

	objectID, err := uuid.Parse(imageId)
	if err != nil {
		log.Warn().Err(err).Msg("invalid uuid")
		withError(ctx, w, http.StatusBadRequest, "invalid image id")
		return
	}

	repo, err := ctrl.repo.BeginTx(ctx)
	if err != nil {
		log.Error().Err(err).Msg("begin tx")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	shouldRollback := true
	defer repo.Rollback(ctx, &shouldRollback)

	objectService := ctrl.srv.WithNewRepo(repo)

	err = objectService.DeleteObject(ctx, objectID)
	if err != nil {
		log.Error().Err(err).Msg("get object")
		switch {
		case errors.As(err, &service.ErrorObjectNotFound):
			withError(ctx, w, http.StatusNotFound, "object not found")
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	err = repo.Commit()
	if err != nil {
		log.Error().Err(err).Msg("commit")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	shouldRollback = false

	w.WriteHeader(http.StatusOK)
	return
}

func (ctrl *Controller) DeleteOldImages(w http.ResponseWriter, r *http.Request, params specs.DeleteOldImagesParams) {
	ctx := r.Context()

	span := trace.FromContext(ctx)
	defer span.End()
	entry := zerolog.Ctx(ctx)

	repo, err := ctrl.repo.BeginTx(ctx)
	if err != nil {
		log.Error().Err(err).Msg("begin tx")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	shouldRollback := true
	defer repo.Rollback(ctx, &shouldRollback)

	objectService := ctrl.srv.WithNewRepo(repo)

	err = objectService.DeleteObjects(ctx, params.CreatedAt)
	if err != nil {
		entry.Error().Err(err).Msg("get object")
		switch {
		case errors.As(err, &service.ErrorObjectNotFound):
			withError(ctx, w, http.StatusNotFound, "object not found")
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	err = repo.Commit()
	if err != nil {
		log.Error().Err(err).Msg("commit")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	shouldRollback = false

	w.WriteHeader(http.StatusOK)
	return
}

func GetApplicationPaginationPolitics() pagination.PaginationPolitics {
	return pagination.PaginationPolitics{
		MaxLimit:     50,
		DefaultLimit: 25,
		OrderByMappgin: map[string]string{
			"date_created": "date_created",
			"status":       "status",
		},
	}
}

func (ctrl *Controller) ListObjectInfo(w http.ResponseWriter, r *http.Request, params specs.ListObjectInfoParams) {
	ctx := r.Context()

	pgnPolitics, err := GetApplicationPaginationPolitics().MakePagination(params.Pagination, params.Sort)
	if err != nil {
		withError(ctx, w, http.StatusBadRequest, err.Error())
		return
	}

	filter := service.ObjectFilter{
		Pagination: pgnPolitics,
	}

	list, total, err := ctrl.srv.ListObjectInfo(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("ListObjectInfo")
		withError(ctx, w, http.StatusInternalServerError, "")
		return
	}

	response := specs.ListObjectsInfoResponse{
		Data: arrayInArray(list, func(in *service.ImageInfo) specs.GetImageInfoResponse {
			out := GetImageForResponse(in)
			return *out
		}),
		Meta: &specs.ResponseMetaTotal{
			Total: total,
		},
	}

	withJSON(ctx, w, http.StatusOK, response)
	return
}

func withError(ctx context.Context, w http.ResponseWriter, code int, message string) {
	resp := specs.Error{
		Code:    code,
		Message: message,
	}

	withJSON(ctx, w, code, resp)
}

func withJSON(ctx context.Context, w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if payload != nil {
		err := json.NewEncoder(w).Encode(payload)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("write answer")
		}
	}
}

func arrayInArray[T any, R any](in []T, f func(T) R) []R {
	if len(in) == 0 {
		return []R{}
	}

	out := make([]R, len(in))

	for i := range in {
		out[i] = f(in[i])
	}

	return out
}
