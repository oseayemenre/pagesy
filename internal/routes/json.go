package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/oseayemenre/pagesy/internal/models"
)

func respondWithSuccess(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func respondWithError(w http.ResponseWriter, code int, error error) {
	respondWithSuccess(w, code, models.ErrorResponse{Error: error.Error()})
}

func (s *Server) decodeJson(w http.ResponseWriter, r *http.Request, params any, service string) error {
	err := json.NewDecoder(r.Body).Decode(&params)

	if err != nil {
		s.Logger.Warn(fmt.Sprintf("error decoding json: %v", err), "service", service)
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error decoding json: %v", err))
	}

	return err
}
