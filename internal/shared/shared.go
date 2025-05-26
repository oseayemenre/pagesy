package shared

import (
	"github.com/go-chi/chi/v5"
	"github.com/oseayemenre/pagesy/internal/logger"
)

type Server struct {
	Router *chi.Mux
	Logger logger.Logger
}
