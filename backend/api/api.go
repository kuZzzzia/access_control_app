package api

import (
	"context"
	"encoding/json"

	"net/http"

	"github.com/rs/zerolog"

	specs "github.com/kuZzzzia/access_control_app/api"
)

type Controller struct {
}

func NewController() *Controller {
	return &Controller{}
}

var _ specs.ServerInterface = &Controller{}

func (ctrl *Controller) GetImage(w http.ResponseWriter, r *http.Request) {
	WithError(r.Context(), w, http.StatusNotImplemented, http.StatusText(http.StatusNotImplemented))
}

func WithError(ctx context.Context, w http.ResponseWriter, code int, message string) {
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
			zerolog.Ctx(ctx).WithError(err).Error("write answer")
		}
	}
}
