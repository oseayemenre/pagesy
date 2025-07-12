package routes

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/markbates/goth/gothic"
	"github.com/oseayemenre/pagesy/internal/models"
	"github.com/oseayemenre/pagesy/internal/shared"
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
	gothic.BeginAuthHandler(w, r)
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
		session, _ := gothic.Store.Get(r, "app_session")
		session.Values["user_id"] = id.String()
		session.Save(r, w)
		http.Redirect(w, r, "/healthz", http.StatusFound) //TODO: put a proper redirect link here when there's a frontend
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

	session, _ := gothic.Store.Get(r, "app_session")
	session.Values["user_id"] = id.String()
	session.Save(r, w)

	http.Redirect(w, r, "/healthz", http.StatusFound) //TODO: put a proper redirect link here when there's a frontend
}

// HandleRegister godoc
// @Summary Register user
// @Description Register user using emal, username and password
// @Tags users
// @Accept application/json
// @Produce json
// @Param user body models.HandleRegisterParams true "user"
// @Failure 400 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Success 201 {object} models.HandleRegisterResponse
// @Router /auth/register [post]
func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var params models.HandleRegisterParams

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		s.Logger.Warn(fmt.Sprintf("error decoding json: %v", err), "service", "HandleRegister")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error decoding json: %v", err))
		return
	}

	if err := shared.Validate.Struct(&params); err != nil {
		s.Logger.Warn(fmt.Sprintf("error validating fields: %v", err), "service", "HandleRegister")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("error validating fields: %v", err))
		return
	}

	id, err := s.Store.CheckIfUserExists(r.Context(), params.Email)

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

	hashedPaswword, err := shared.HashPassword(params.Password)

	if err != nil {
		s.Logger.Error(err.Error(), "service", "HandleRegister")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	id, err = s.Store.CreateUser(r.Context(), &models.User{
		Username: params.Username,
		Email:    params.Email,
		Password: hashedPaswword,
	})

	if err != nil {
		s.Logger.Error(err.Error(), "service", "HandleRegister")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithSuccess(w, http.StatusCreated, &models.HandleRegisterResponse{Id: id.String()})
}

func (s *Server) HandleLogin(w http.ResponseWriter, r http.Request) {}

func (s *Server) HandleBanUser(w http.ResponseWriter, r *http.Request) {}
