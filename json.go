package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

type errorResponse struct {
	Error string `json:"error"`
}

var (
	errValidation = errors.New("validation")
)

func encode(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func decode(r *http.Request, data any) error {
	if err := json.NewDecoder(r.Body).Decode(data); err != nil {
		return fmt.Errorf("error decoding json, %v", err)
	}

	if err := validate.Struct(data); err != nil {
		return fmt.Errorf("%w, %w", errValidation, err)
	}

	return nil
}
