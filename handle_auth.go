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

// handleAuthGoogle godoc
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

// handleAuthGoogleCallback godoc
//
//	@Summary		Google auth callback url
//	@Description	Google auth callback url
//	@Tags			auth
//	@Failure		404	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
func (s *server) handleAuthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	r = r.WithContext(context.WithValue(r.Context(), "provider", "google"))

	user, err := gothic.CompleteUserAuth(w, r)

	if err != nil {
		encode(w, http.StatusNotFound, &errorResponse{Error: fmt.Sprintf("error retrieving user details, %v", err)})
		return
	}

	id, err := s.checkIfUserExists(r.Context(), user.Email, "")

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if id != "" {
		if err := createAccessAndRefreshTokens(w, id, os.Getenv("JWT_SECRET")); err != nil {
			s.logger.Error(err.Error())
			encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
			return
		}
		return
	}

	session, _ := gothic.Store.Get(r, "app_session")
	session.Values["user_email"] = user.Email
	session.Save(r, w)
	http.Redirect(w, r, "/healthz", http.StatusFound) //TODO: put a proper redirect link here when there's a frontend
}

// handleAuthOnboarding godoc
//
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
	type request struct {
		username     string
		display_name string
		about        string
		image        string
	}

	type response struct {
		Id string `json:"id"`
	}

	session, _ := gothic.Store.Get(r, "app_session")

	email, ok := session.Values["user_email"].(string)

	if !ok || email == "" {
		encode(w, http.StatusNotFound, &errorResponse{Error: "no user in session"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 500<<10)

	if err := r.ParseMultipartForm(500 << 10); err != nil {
		encode(w, http.StatusBadRequest, &errorResponse{Error: err.Error()})
		return
	}

	params := request{
		username:     r.FormValue("username"),
		display_name: r.FormValue("display_name"),
		about:        r.FormValue("about"),
	}

	if err := s.validator.Struct(&params); err != nil {
		encode(w, http.StatusBadRequest, &errorResponse{Error: fmt.Sprintf("validation error: %v", err)})
		return
	}

	file, header, err := r.FormFile("image")

	if err != nil && err != http.ErrMissingFile {
		encode(w, http.StatusBadRequest, &errorResponse{Error: fmt.Sprintf("error retrieving file: %v", err)})
		return
	}

	if err == nil {
		defer file.Close()
		image, err := io.ReadAll(file)

		if err != nil {
			s.logger.Error(fmt.Sprintf("error reading bytes, %v", err))
			encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
			return
		}

		if len(image) > 400<<10 {
			encode(w, http.StatusRequestEntityTooLarge, &errorResponse{Error: "image too large"})
			return
		}

		if contentType := http.DetectContentType(image); !strings.HasPrefix(contentType, "image/") {
			encode(w, http.StatusBadRequest, &errorResponse{Error: "invalid file type"})
			return
		}

		img_url, err := s.objectStore.upload(r.Context(), fmt.Sprintf("books/%s_%s", email, header.Filename), bytes.NewReader(image))

		if err != nil {
			s.logger.Error(err.Error())
			encode(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
			return
		}

		params.image = img_url
	}

	var password string

	if session.Values["user_password"] == nil {
		password = ""
	} else {
		password = session.Values["user_password"].(string)
	}

	id, err := s.createUser(r.Context(), &user{
		username:     params.username,
		display_name: params.display_name,
		email:        email,
		password:     password,
		about:        params.about,
		image:        params.image,
	})

	if err != nil {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	delete(session.Values, "user_email")
	delete(session.Values, "user_password")

	session.Options.MaxAge = -1

	if err := session.Save(r, w); err != nil {
		s.logger.Error(fmt.Sprintf("error deleting session, %v", err))
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	if err := createAccessAndRefreshTokens(w, id, os.Getenv("JWT_SECRET")); err != nil {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: err.Error()})
		return
	}

	encode(w, http.StatusCreated, response{Id: id})
}

// handleRegister godoc
//
//	@Summary		Register user
//	@Description	Register user using email, username and password
//	@Tags			auth
//	@Accept			application/json
//	@Produce		json
//	@Param			user	body		main.handleAuthRegister.request	true	"user"
//	@Failure		400		{object}	errorResponse
//	@Failure		409		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Success		302
//	@Header			302	{string}	Set-Cookie	"app_session"
//	@Router			/auth/register [post]
func (s *server) handleAuthRegister(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Email    string `json:"email" validate:"email,required"`
		Password string `json:"password" validate:"required,min=8"`
	}
}

func (s *server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {}

func (s *server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {}

func (s *server) handleAuthRefreshToken(w http.ResponseWriter, r *http.Request) {}
