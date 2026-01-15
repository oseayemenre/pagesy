package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

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

	id, err := s.checkIfUserExists(r.Context(), user.Email, "")

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.logger.Error(err.Error())
		responseFailure(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if id != "" {
		if err := createAccessAndRefreshTokens(w, id, os.Getenv("JWT_SECRET")); err != nil {
			s.logger.Error(err.Error())
			responseFailure(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

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
//	@Failure		400				{object}	errorResponse
//	@Failure		404				{object}	errorResponse
//	@Failure		413				{object}	errorResponse
//	@Failure		500				{object}	errorResponse
//	@Success		201				{object}	main.handleAuthOnboarding.response
//	@Header			201				{string}	Set-Cookie	"access_token=12345 refresh_token=12345"
//	@Router			/auth/onboarding [post]

func (s *server) handleAuthOnboarding(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Id string `json:"id"`
	}

	type request struct {
		username     string
		display_name string
		about        string
		image        string
	}

	session, _ := gothic.Store.Get(r, "app_session")

	email, ok := session.Values["user_email"].(string)

	if !ok || email == "" {
		responseFailure(w, http.StatusNotFound, "no user in session")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 500<<10)

	if err := r.ParseMultipartForm(500 << 10); err != nil {
		responseFailure(w, http.StatusBadRequest, err)
		return
	}

	params := request{
		username:     r.FormValue("username"),
		display_name: r.FormValue("display_name"),
		about:        r.FormValue("about"),
	}

	if err := validate.Struct(&params); err != nil {
		responseFailure(w, http.StatusBadRequest, fmt.Sprintf("validation error: %v", err))
		return
	}

	file, header, err := r.FormFile("image")

	if err != nil && err != http.ErrMissingFile {
		responseFailure(w, http.StatusBadRequest, fmt.Sprintf("error retrieving file: %v", err))
		return
	}

	if err == nil {
		defer file.Close()
		image, err := io.ReadAll(file)

		if err != nil {
			s.logger.Error(fmt.Sprintf("error reading bytes: %v", err))
			responseFailure(w, http.StatusInternalServerError, "internal server error")
			return
		}

		if len(image) > 400<<10 {
			responseFailure(w, http.StatusRequestEntityTooLarge, "image too large")
			return
		}

		if contentType := http.DetectContentType(image); !strings.HasPrefix(contentType, "image/") {
			responseFailure(w, http.StatusBadRequest, "invalid file type")
			return
		}

		params.image, err = a.objectStore.UploadFile(r.Context(), bytes.NewReader(image), fmt.Sprintf("%s_%s", email, header.Filename))

		if err != nil {
			s.logger.Error(err.Error())
			responseFailure(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	id, err := s.createUser(r.Context(), &user{
		username:     params.username,
		display_name: params.display_name,
		email:        email,
		password:     session.Values["user_password"].(string),
		about:        params.about,
		image:        params.image,
	})

	if err != nil {
		s.logger.Error(err.Error())
		responseFailure(w, http.StatusInternalServerError, "internal server error")
		return
	}

	delete(session.Values, "user_email")
	delete(session.Values, "user_password")

	session.Options.MaxAge = -1

	if err := session.Save(r, w); err != nil {
		s.logger.Error(fmt.Sprintf("error deleting session: %v", err))
		responseFailure(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := createAccessAndRefreshTokens(w, id, os.Getenv("JWT_SECRET")); err != nil {
		s.logger.Error(err.Error())
		responseFailure(w, http.StatusInternalServerError, err.Error())
		return
	}

	responseSuccess(w, http.StatusCreated, response{Id: id})
}

func (s *server) handleAuthRegister(w http.ResponseWriter, r *http.Request) {}

func (s *server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {}

func (s *server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {}

func (s *server) handleAuthRefreshToken(w http.ResponseWriter, r *http.Request) {}
