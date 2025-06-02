package shared

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/oseayemenre/pagesy/internal/logger"
	"github.com/oseayemenre/pagesy/internal/store"
)

type Server struct {
	Router      *chi.Mux
	Logger      logger.Logger
	ObjectStore store.ObjectStore
	Store       store.Store
}

var Validate = validator.New()
