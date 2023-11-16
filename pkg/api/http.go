package api

import (
	"github.com/go-chi/chi/v5"
)

type Api interface {
	Register(r chi.Router)
}
