package routes

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/markbates/goth/gothic"
	"github.com/oseayemenre/pagesy/internal/bcrypt"
	"github.com/oseayemenre/pagesy/internal/cookies"
	"github.com/oseayemenre/pagesy/internal/jwt"
	"github.com/oseayemenre/pagesy/internal/models"
	"github.com/oseayemenre/pagesy/internal/shared"
)

// HandleGoogleSignIn godoc
//
//	@Summary		Sign in with google
//	@Description	Sign in with google
//	@Tags			auth
//	@Success		302
//	@Success		307
//	@Router			/auth/google [get]
func (s *Server) HandleGoogleSignIn(w http.ResponseWriter, r *http.Request) {
	r = r.WithContext(context.WithValue(r.Context(), "provider", "google"))
	gothic.BeginAuthHandler(w, r)
}

// HandleGoogleSignInCallback godoc
//
//	@Summary		Google auth callback url
//	@Description	Google auth callback url
//	@Tags			auth
//	@Failure		404	{object}	models.ErrorResponse
//	@Failure		500	{object}	models.ErrorResponse
//	@Success		302
//	@Router			/auth/google/callback [get]
func (s *Server) HandleGoogleSignInCallback(w http.ResponseWriter, r *http.Request) {
	r = r.WithContext(context.WithValue(r.Context(), "provider", "google"))

	user, err := gothic.CompleteUserAuth(w, r)

	if err != nil {
		s.Logger.Warn(fmt.Sprintf("error retrieving user details: %v", err), "service", "HandleGoogleSignInCallback")
		respondWithError(w, http.StatusNotFound, fmt.Errorf("error retrieving user details: %v", err))
		return
	}

	id, err := s.Store.CheckIfUserExists(r.Context(), user.Email, "")

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.Logger.Warn(err.Error(), "service", "HandleGoogleSignInCallback")
		respondWithError(w, http.StatusNotFound, err)
		return
	}

	if id != nil {
		if err := cookies.CreateAccessAndRefreshTokens(w, id.String(), s.Config.Jwt_secret); err != nil {
			s.Server.Logger.Error(err.Error(), "service", "HandleOnboarding")
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}
	}

	session, _ := gothic.Store.Get(r, "app_session")
	session.Values["user_email"] = user.Email
	session.Save(r, w)
	http.Redirect(w, r, "/healthz", http.StatusFound) //TODO: put a proper redirect link here when there's a frontend
}

// HandleOnboarding godoc
//
//	@Summary		Onboard users
//	@Description	Onboard users with display_name, name, about and image
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
func (s *Server) HandleOnboarding(w http.ResponseWriter, r *http.Request) {
	session, _ := gothic.Store.Get(r, "app_session")

	email, ok := session.Values["user_email"].(string)

	if !ok || email == "" {
		s.Logger.Warn("no user in session", "status", "permission denied")
		respondWithError(w, http.StatusNotFound, fmt.Errorf("no user in session"))
		return
	}

	password, _ := session.Values["user_password"].(string)

	r.Body = http.MaxBytesReader(w, r.Body, 500<<10)

	if err := r.ParseMultipartForm(500 << 10); err != nil {
		s.Logger.Warn(fmt.Sprintf("error parsing data: %v", err), "service", "HandleOnboarding")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	params := models.HandleOnboardingParams{
		Username:     r.FormValue("username"),
		Display_name: r.FormValue("display_name"),
		About:        r.FormValue("about"),
	}

	if err := shared.Validate.Struct(&params); err != nil {
		s.Server.Logger.Warn(fmt.Sprintf("validation error: %v", err), "service", "HandleOnboarding")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("validation error: %v", err))
		return
	}

	file, header, err := r.FormFile("image")

	if err != nil && err != http.ErrMissingFile {
		s.Server.Logger.Error(fmt.Sprintf("error reading bytes: %v", err), "service", "HandleOnboarding")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error reading bytes: %v", err))
		return
	}

	if err == nil {
		defer file.Close()
		image, err := io.ReadAll(file)

		if err != nil {
			s.Server.Logger.Error(fmt.Sprintf("error reading bytes: %v", err), "service", "HandleOnboarding")
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error reading bytes: %v", err))
			return
		}

		if len(image) > 400<<10 {
			s.Server.Logger.Error("image too large", "service", "HandleOnboarding")
			respondWithError(w, http.StatusRequestEntityTooLarge, fmt.Errorf("image too large"))
			return
		}

		if contentType := http.DetectContentType(image); !strings.HasPrefix(contentType, "image/") {
			s.Server.Logger.Warn("invalid file type", "service", "HandleOnboarding")
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("invalid file type"))
			return
		}

		params.Image, err = s.Server.ObjectStore.UploadFile(r.Context(), bytes.NewReader(image), fmt.Sprintf("%s_%s", email, header.Filename))

		if err != nil {
			s.Server.Logger.Error(err.Error(), "service", "HandleOnboarding")
			respondWithError(w, http.StatusBadRequest, err)
			return
		}
	}

	id, err := s.Store.CreateUser(r.Context(), &models.User{
		Username:     params.Username,
		Display_name: params.Display_name,
		Email:        email,
		Password:     password,
		About:        params.About,
		Image:        params.Image,
	})

	if err != nil {
		s.Server.Logger.Error(err.Error(), "service", "HandleOnboarding")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	delete(session.Values, "user_email")
	delete(session.Values, "user_password")

	session.Options.MaxAge = -1

	if err := session.Save(r, w); err != nil {
		s.Server.Logger.Error(fmt.Sprintf("error deleting session: %v", err), "service", "HandleOnboarding")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error deleting session: %v", err))
		return
	}

	if err := cookies.CreateAccessAndRefreshTokens(w, id.String(), s.Config.Jwt_secret); err != nil {
		s.Server.Logger.Error(err.Error(), "service", "HandleOnboarding")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithSuccess(w, http.StatusCreated, &models.HandleRegisterResponse{Id: id.String()})
}

// HandleRegister godoc
//
//	@Summary		Register user
//	@Description	Register user using email, username and password
//	@Tags			auth
//	@Accept			application/json
//	@Produce		json
//	@Param			user	body		models.HandleRegisterParams	true	"user"
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		409		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Success		302
//	@Header			302	{string}	Set-Cookie	"app_session"
//	@Router			/auth/register [post]
func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var params models.HandleRegisterParams

	if err := decodeJson(r, &params); err != nil {
		s.Logger.Warn(err.Error(), "service", "HandleRegister")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if err := shared.Validate.Struct(&params); err != nil {
		s.Logger.Warn(fmt.Sprintf("error validating fields: %v", err), "service", "HandleRegister")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("error validating fields: %v", err))
		return
	}

	id, err := s.Store.CheckIfUserExists(r.Context(), params.Email, "")

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.Logger.Error(err.Error(), "service", "HandleRegister")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	if id != nil {
		s.Logger.Warn("user already exists", "service", "HandleRegister")
		respondWithError(w, http.StatusConflict, fmt.Errorf("user already exists"))
		return
	}

	hashedPaswword, err := bcrypt.HashPassword(params.Password)

	if err != nil {
		s.Logger.Error(err.Error(), "service", "HandleRegister")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	session, _ := gothic.Store.Get(r, "app_session")
	session.Values["user_email"] = params.Email
	session.Values["user_password"] = hashedPaswword
	session.Save(r, w)
	http.Redirect(w, r, "/healthz", http.StatusFound) //TODO: put a proper redirect link here when there's a frontend
}

// HandleLogin godoc
//
//	@Summary		Login
//	@Description	Login using either email, username or both and password
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			user_credentials	body		models.HandleLoginParams	true	"user credentials"
//	@Failure		400					{object}	models.ErrorResponse
//	@Failure		401					{object}	models.ErrorResponse
//	@Failure		404					{object}	models.ErrorResponse
//	@Failure		500					{object}	models.ErrorResponse
//	@Success		204
//	@Header			204	{string}	Set-Cookie	"access_token=12345 refresh_token=12345"
//	@Router			/auth/login [post]
func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {

	var params models.HandleLoginParams

	if err := decodeJson(r, &params); err != nil {
		s.Logger.Warn(err.Error(), "service", "HandleLogin")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if params.Email == "" && params.Username == "" {
		s.Logger.Warn("email and username cannot be empty", "service", "HandleLogin")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("email and username cannot be empty"))
		return
	}

	if err := shared.Validate.Struct(&params); err != nil {
		s.Logger.Warn(fmt.Sprintf("error validating fields: %v", err), "service", "HandleLogin")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("error validating fields: %v", err))
		return
	}

	id, err := s.Store.CheckIfUserExists(r.Context(), params.Email, params.Username)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.Logger.Warn("account does not exist", "service", "HandleLogin")
			respondWithError(w, http.StatusNotFound, fmt.Errorf("account does not exist"))
			return
		}
		s.Logger.Error(err.Error(), "service", "HandleLogin")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	password, err := s.Store.GetUserPassword(r.Context(), id.String())

	if err != nil {
		s.Logger.Error(err.Error(), "service", "HandleLogin")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	if err := bcrypt.ComparePassword(params.Password, password); err != nil {
		s.Logger.Warn(err.Error(), "service", "HandleLogin")
		respondWithError(w, http.StatusUnauthorized, err)
		return
	}

	if err := cookies.CreateAccessAndRefreshTokens(w, id.String(), s.Config.Jwt_secret); err != nil {
		s.Server.Logger.Error(err.Error(), "service", "HandleLogin")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithSuccess(w, http.StatusNoContent, nil)
}

// HandleLogout godoc
//
//	@Summary		Logout user
//	@Description	Logout user
//	@Tags			auth
//	@Success		204
//	@Router			/auth/logout [post]
func (s *Server) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "access_token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	http.SetCookie(w, &http.Cookie{
		Name:   "refresh_token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	respondWithSuccess(w, http.StatusNoContent, nil)
}

// HandleRefreshToken godoc
//
//	@Summary		Refresh token
//	@Description	Get new access token
//	@Tags			auth
//	@Failure		404	{object}	models.ErrorResponse
//	@Failure		500	{object}	models.ErrorResponse
//	@Success		204
//	@Header			204	{string}	Set-Cookie	"access_token=12345 refresh_token=12345"
//	@Router			/auth/refresh-token [post]
func (s *Server) HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	token, err := r.Cookie("refresh_token")

	if err != nil {
		s.Server.Logger.Warn("refresh token cookie not found", "service", "HandleRefreshToken")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("refresh token cookie not found"))
		return
	}

	id, err := jwt.DecodeJWTToken(token.Value, s.Config.Jwt_secret)

	if err != nil {
		s.Logger.Warn(err.Error(), "service", "HandleRefreshToken")
		respondWithError(w, http.StatusUnauthorized, err)
		return
	}

	access_token, err := jwt.CreateJWTToken(id, s.Config.Jwt_secret)

	if err != nil {
		s.Logger.Warn(err.Error(), "service", "HandleRefreshToken")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    access_token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   24 * 60 * 60,
	})

	respondWithSuccess(w, http.StatusNoContent, nil)
}
