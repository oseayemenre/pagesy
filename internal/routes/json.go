package routes

import (
	"encoding/json"
	"net/http"
)

func respondWithSuccess(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func respondWithError(w http.ResponseWriter, code int, error error) {
	respondWithSuccess(w, code, map[string]string{"error": error.Error()})
}
