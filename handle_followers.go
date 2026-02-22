package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// handleFollowUser godoc
//
//	@Summary		Follow user
//	@Description	Follow user
//	@Tags			followers
//	@Param			userID	path		string	true	"user id"
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Success		204
//	@Router			/users/{userID}/follow [post]
func (s *server) handleFollowUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	followerID := r.Context().Value("user").(string)

	if userID == followerID {
		encode(w, http.StatusBadRequest, &errorResponse{Error: "user cannot follow themselves"})
		return
	}

	displayName, err := s.followUser(r.Context(), r.Context().Value("user").(string), chi.URLParam(r, "userID"))
	if errors.Is(err, errUserNotFound) {
		encode(w, http.StatusNotFound, &errorResponse{Error: err.Error()})
		return
	}
	if err != nil {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	s.hub.regular[userID].send <- fmt.Appendf([]byte{}, "%v followed you", displayName)

	encode(w, http.StatusNoContent, nil)
}
