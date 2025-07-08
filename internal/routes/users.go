package routes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/markbates/goth/gothic"
	"github.com/oseayemenre/pagesy/internal/models"
)

// HandleGoogleSignIn godoc
// @Summary Sign in with google
// @Description Sign in with google
// @Tags users
// @Success 302
// @Success 307
// @Router /auth/google [get]
func (s *Server) HandleGoogleSignIn(w http.ResponseWriter, r *http.Request) {
	r = r.WithContext(context.WithValue(r.Context(), "provider", "google"))

	if _, err := gothic.CompleteUserAuth(w, r); err == nil {
		http.Redirect(w, r, "/", http.StatusFound) //TODO: put a proper redirect link here when there's a frontend
	} else {
		gothic.BeginAuthHandler(w, r)
	}
}

// HandleGoogleSignInCallback godoc
// @Summary Google auth callback url
// @Description Google auth callback url
// @Tags users
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Success 302
// @Router /auth/google/callback [get]
func (s *Server) HandleGoogleSignInCallback(w http.ResponseWriter, r *http.Request) {
	r = r.WithContext(context.WithValue(r.Context(), "provider", "google"))

	user, err := gothic.CompleteUserAuth(w, r)

	if err != nil {
		s.Logger.Warn(fmt.Sprintf("error retrieving user details: %v", err), "service", "HandleGoogleSignInCallback")
		respondWithError(w, http.StatusNotFound, fmt.Errorf("error retrieving user details: %v", err))
		return
	}

	id, err := s.Store.CheckIfUserExists(r.Context(), user.Email)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.Logger.Error(err.Error(), "service", "HandleGoogleSignInCallback")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	if id != nil {
		s.Logger.Warn("user already exists", "service", "HandleGoogleSignInCallback")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	id, err = s.Store.CreateUserOauth(r.Context(), &models.User{
		Email: user.Email,
		Image: user.AvatarURL,
	})

	if err != nil {
		s.Logger.Warn(err.Error(), "service", "HandleGoogleSignInCallback")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound) //TODO: put a proper redirect link here when there's a frontend
}

func (s *Server) HandleCreateUser(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleBanUser(w http.ResponseWriter, r *http.Request) {}
