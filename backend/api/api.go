package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"

	"net/http"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.opencensus.io/trace"

	"github.com/kuZzzzia/access_control_app/backend/service"
	"github.com/kuZzzzia/access_control_app/backend/specs"
	"github.com/kuZzzzia/access_control_app/backend/storage/postgres"
)

type Controller struct {
	srv  *service.Service
	repo *postgres.Repo

	DenyTypes map[string]string
	SizeLimit int64
}

func NewController(srv *service.Service,
	repo *postgres.Repo,
	DenyTypes map[string]string,
	SizeLimit int64) *Controller {
	return &Controller{
		srv:  srv,
		repo: repo,
	}
}

var _ specs.ServerInterface = &Controller{}

func (ctrl *Controller) CreateImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	entry := zerolog.Ctx(ctx)
	var err error

	if r.Body == nil {
		entry.Warn().Msg("nil body")
		withError(ctx, w, http.StatusBadRequest, "nil body")
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		err = r.MultipartForm.RemoveAll()
		if err != nil {
			entry.Error().Err(err).Msg("unable to remove form")
		}
		withError(ctx, w, http.StatusBadRequest, "form file 'upfile' not found")
		return
	}
	defer func() {
		errClose := file.Close()
		if errClose != nil {
			entry.Error().Err(errClose).Msg("unable to close form file")
		}
		err = r.MultipartForm.RemoveAll()
		if err != nil {
			entry.Error().Err(err).Msg("unable to remove form")
		}
	}()

	contentType := fileHeader.Header.Get("Content-Type")

	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		entry.Warn().Err(err).Msg("parse media type")
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
		entry.Warn().Int64("Content-Length", r.ContentLength).Int64("max size limit", ctrl.SizeLimit).
			Msg("incorrect Content-Length")
		withError(ctx, w, http.StatusBadRequest, "incorrect Content-Length")
		return
	}
	entry.Info().Int64("Content-Length", r.ContentLength).Msg("parse Content-Length successful")

	object := &service.ImageInfo{
		ID:          uuid.New(),
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
		entry.Error().Err(err).Msg("begin tx")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	shouldRollback := true
	defer repo.Rollback(ctx, &shouldRollback)

	objectService := ctrl.srv.WithNewRepo(repo)

	err = objectService.CreateObject(ctx, file, object)
	if err != nil {
		entry.Error().Err(err).Msg("create object")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = repo.Commit()
	if err != nil {
		entry.Error().Err(err).Msg("commit")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	shouldRollback = false

	res, err := ctrl.srv.GetObjectInfo(ctx, object.ID)
	if err != nil {
		entry.Error().Err(err).Msg("return resume info")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	withJSON(ctx, w, http.StatusOK, GetImageForResponse(res))
}

func GetImageForResponse(res *service.ImageInfo) *specs.GetImageInfoResponse {
	return &specs.GetImageInfoResponse{
		Id:           res.ID.String(),
		PeopleNumber: res.PeopleNumber,
	}
}

func (ctrl *Controller) GetImage(w http.ResponseWriter, r *http.Request, imageId string) {
	ctx := r.Context()

	span := trace.FromContext(ctx)
	defer span.End()
	entry := zerolog.Ctx(ctx)

	objectID, err := uuid.Parse(imageId)
	if err != nil {
		entry.Warn().Err(err).Msg("invalid uuid")
		withError(ctx, w, http.StatusBadRequest, "invalid process_code")
		return
	}

	obj, object, err := ctrl.srv.GetObject(ctx, objectID)
	if err != nil {
		entry.Error().Err(err).Msg("get object")
		switch {
		case errors.As(err, &service.ErrorObjectNotFound):
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	defer obj.Close()

	w.Header().Set("Content-Type", object.ContentType)
	w.Header().Set("Content-Disposition", "attachment; filename="+object.Name)

	_, err = io.Copy(w, obj)
	if err != nil {
		entry.Error().Err(err).Msg("return obj")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (ctrl *Controller) GetImageInfo(w http.ResponseWriter, r *http.Request, imageId string) {
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
