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

func decodeJson(r *http.Request, params any) error {
	err := json.NewDecoder(r.Body).Decode(&params)

	if err != nil {
		return fmt.Errorf("error decoding json: %v", err)
	}

	return nil
}
