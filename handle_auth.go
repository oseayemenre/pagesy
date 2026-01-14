package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"database/sql"

	"github.com/markbates/goth/gothic"
)

// HandleAuthGoogle godoc
//
//	@Summary		Sign in with google
//	@Description	Sign in with google
//	@Tags			auth
//	@Success		302
//	@Success		307
//	@Router			/auth/google [get]
func (s *server) handleAuthGoogle(w http.ResponseWriter, r *http.Request) {
	r = r.WithContext(context.WithValue(r.Context(), "provider", "google"))
	gothic.BeginAuthHandler(w, r)
}

// HandleAuthGoogleCallback godoc
//
//	@Summary		Google auth callback url
//	@Description	Google auth callback url
//	@Tags			auth
//	@Failure		404	{object}	responseFailure
//	@Failure		404	{object}	responseFailure
func (s *server) handleAuthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	r = r.WithContext(context.WithValue(r.Context(), "provider", "google"))

	user, err := gothic.CompleteUserAuth(w, r)

	if err != nil {
		responseFailure(w, http.StatusNotFound, fmt.Errorf("error retrieving user details: %v", err))
		return
	}

	s.logger.Debug("checking if user exists...")

	id, err := s.CheckIfUserExists(r.Context(), user.Email, "")

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.logger.Error(err.Error())
		responseFailure(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	s.logger.Debug("creating access and refresh tokens...")
	if id != "" {
		if err := createAccessAndRefreshTokens(w, id, os.Getenv("JWT_SECRET")); err != nil {
			s.logger.Error(err.Error())
			responseFailure(w, http.StatusInternalServerError, "something went wrong")
			return
		}
	}

	s.logger.Debug("setting cookies...")
	session, _ := gothic.Store.Get(r, "app_session")
	session.Values["user_email"] = user.Email
	session.Save(r, w)
	http.Redirect(w, r, "/healthz", http.StatusFound) //TODO: put a proper redirect link here when there's a frontend
}

// HandleAuthOnboarding godoc
//	@Summary		Onboard users
//	@Description	Onboard users with display name, name, about and image
//	@Tags			auth
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			username		formData	string	true	"username"
//	@Param			display_name	formData	string	true	"display name"
//	@Param			about			formData	string	false	"about"
//	@Param			image			formData	file	false	"profile_picture"
//	@Param			Cookie			header		string	true	"app_session=12345"
//	@Failure		400				{object}	models.ErrorResponse
//	@Failure		404				{object}	models.ErrorResponse
//	@Failure		413				{object}	models.ErrorResponse
//	@Failure		500				{object}	models.ErrorResponse
//	@Success		201				{object}	models.HandleRegisterResponse
//	@Header			201				{string}	Set-Cookie	"access_token=12345 refresh_token=12345"
//	@Router			/auth/onboarding [post]

func (s *server) handleAuthOnboarding(w http.ResponseWriter, r *http.Request) {}

func (s *server) handleAuthRegister(w http.ResponseWriter, r *http.Request) {}

func (s *server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {}

func (s *server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {}

func (s *server) handleAuthRefreshToken(w http.ResponseWriter, r *http.Request) {}
