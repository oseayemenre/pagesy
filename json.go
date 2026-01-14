package main

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

func responseSuccess(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func responseFailure(w http.ResponseWriter, status int, data any) {
	responseSuccess(w, status, errorResponse{
		Error: data.(string),
	})
}
