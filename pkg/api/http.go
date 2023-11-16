package api

import (
	"github.com/caas-team/sparrow/pkg/db"
	"github.com/go-chi/chi/v5"
)

type HttpApi struct {
}

type Api interface {
	Register(r chi.Router)
}

func New(db db.DB) HttpApi {
	api := HttpApi{}
	return api
}

func (h *HttpApi) Register(r chi.Router) {
	// TODO register handlers

}
