package shared

import (
	"fmt"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/oseayemenre/pagesy/internal/logger"
	"github.com/oseayemenre/pagesy/internal/store"
	"golang.org/x/crypto/bcrypt"
)

type Server struct {
	Router      *chi.Mux
	Logger      logger.Logger
	ObjectStore store.ObjectStore
	Store       store.Store
}

var Validate = validator.New()

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return "", fmt.Errorf("error hashing password: %v", err)
	}

	return string(hash), nil
}
