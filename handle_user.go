package main

import (
	"database/sql"
	"fmt"
	"net/http"
)

// handleGetProfile godoc
// @Summary Get current user profile
// @Description Get current user profile
// @Tags users
// @Produce json
// @Failure 404 {object} errorResponse
// @Success 200 {object} main.handleGetProfile.response
// @Router /api/v1/users/me [get]
func (s *server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Email        string         `json:"email"`
		Display_name string         `json:"display_name"`
		Image        sql.NullString `json:"image"`
		About        sql.NullString `json:"about"`
		Roles        []string       `json:"roles"`
	}

	cookie, err := r.Cookie("access_token")
	if err != nil {
		encode(w, http.StatusNotFound, &errorResponse{Error: fmt.Sprintf("error retrieving access token, %v", err)})
		return
	}

	id, err := decodeJWTToken(cookie.Value)
	if err != nil {
		encode(w, http.StatusBadRequest, &errorResponse{Error: err.Error()})
		return
	}

	user, err := s.getUser(r.Context(), id)
	if err != nil {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	encode(w, http.StatusOK, &response{Email: user.email, Display_name: user.display_name, Image: user.image, About: user.about, Roles: user.roles})
}
