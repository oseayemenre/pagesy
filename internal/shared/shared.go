package shared

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/oseayemenre/pagesy/internal/logger"
)

type Server struct {
	Router *chi.Mux
	Logger logger.Logger
}

var Validate = validator.New()
