package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/oseayemenre/pagesy/internal/config"
	"github.com/oseayemenre/pagesy/internal/models"
)

func TestHandleRegister(t *testing.T) {
	tests := []struct {
		name           string
		body           any
		userExistsFunc func(ctx context.Context, email string, username string) (*uuid.UUID, error)
		expectedCode   int
	}{
		{
			name:         "should return 400 if json could not be decoded",
			body:         &struct{ Email int }{Email: 1},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "should return 400 if fields could not be validated",
			body: &models.HandleRegisterParams{
				Email:    "fail_email",
				Password: "12345678",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "should return 500 if something went wrong while checking if user exists",
			body: &models.HandleRegisterParams{
				Email:    "test@test.com",
				Password: "12345678",
			},
			userExistsFunc: func(ctx context.Context, email string, username string) (*uuid.UUID, error) {
				return nil, errors.New("something went wrong")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "should return 409 if user already exists",
			body: &models.HandleRegisterParams{
				Email:    "test@test.com",
				Password: "12345678",
			},
			userExistsFunc: func(ctx context.Context, email string, username string) (*uuid.UUID, error) {
				id := uuid.New()
				return &id, nil
			},
			expectedCode: http.StatusConflict,
		},
		{
			name: "should return 302 if session has been created",
			body: &models.HandleRegisterParams{
				Email:    "test@test.com",
				Password: "12345678",
			},
			userExistsFunc: func(ctx context.Context, email string, username string) (*uuid.UUID, error) {
				return nil, nil
			},
			expectedCode: http.StatusFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Api{
				logger:      &testLogger{},
				objectStore: &testObjectStore{},
				store: &testStore{
					userExistsFunc: tt.userExistsFunc,
				},
			}

			data, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/register", bytes.NewBuffer(data))
			rr := httptest.NewRecorder()

			a.HandleRegister(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleLogin(t *testing.T) {
	tests := []struct {
		name                string
		body                any
		userExistsFunc      func(ctx context.Context, email string, username string) (*uuid.UUID, error)
		getUserPasswordFunc func(ctx context.Context, id string) (string, error)
		expectedCode        int
	}{
		{
			name:         "should return 400 if json could not be decoded",
			body:         &struct{ Password int }{Password: 1},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "should return 400 if email and username is empty",
			body: &models.HandleLoginParams{
				Password: "12345678",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "should return 400 if fields could not be validated",
			body: &models.HandleLoginParams{
				Email: "fail_email",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "should return 500 if something went wrong while checking if user exists",
			body: &models.HandleLoginParams{
				Email:    "test@test.com",
				Password: "_",
			},
			userExistsFunc: func(ctx context.Context, email, username string) (*uuid.UUID, error) {
				return nil, errors.New("something went wrong")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "should return 404 if user is not found",
			body: &models.HandleLoginParams{
				Email:    "test@test.com",
				Password: "_",
			},
			userExistsFunc: func(ctx context.Context, email, username string) (*uuid.UUID, error) {
				return nil, sql.ErrNoRows
			},
			expectedCode: http.StatusNotFound,
		},
		{
			name: "should return 500 if something went wrong while getting user password",
			body: &models.HandleLoginParams{
				Email:    "test@test.com",
				Password: "_",
			},
			userExistsFunc: func(ctx context.Context, email string, username string) (*uuid.UUID, error) {
				id := uuid.New()
				return &id, nil
			},
			getUserPasswordFunc: func(ctx context.Context, id string) (string, error) {
				return "", errors.New("something went wrong")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "should return 401 if password doesn't match",
			body: &models.HandleLoginParams{
				Email:    "test@test.com",
				Password: "_",
			},
			userExistsFunc: func(ctx context.Context, email string, username string) (*uuid.UUID, error) {
				id := uuid.New()
				return &id, nil
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "should return 204 and login user",
			body: &models.HandleLoginParams{
				Email:    "test@test.com",
				Password: "password",
			},
			userExistsFunc: func(ctx context.Context, email string, username string) (*uuid.UUID, error) {
				id := uuid.New()
				return &id, nil
			},
			expectedCode: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Api{
				logger:      &testLogger{},
				objectStore: &testObjectStore{},
				config: &config.Config{
					Jwt_secret: "secret",
				},
				store: &testStore{
					userExistsFunc:      tt.userExistsFunc,
					getUserPasswordFunc: tt.getUserPasswordFunc,
				},
			}

			data, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(data))
			rr := httptest.NewRecorder()

			a.HandleLogin(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleLogout(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rr := httptest.NewRecorder()

	http.SetCookie(rr, &http.Cookie{
		Name:   "access_token",
		Value:  "12345678",
		Path:   "/",
		MaxAge: 60 * 60,
	})

	http.SetCookie(rr, &http.Cookie{
		Name:   "refresh_token",
		Value:  "12345678",
		Path:   "/",
		MaxAge: 60 * 60,
	})

	a := &Api{
		logger:      &testLogger{},
		objectStore: &testObjectStore{},
		store:       &testStore{},
	}

	a.HandleLogout(rr, req)

	var hasaccesstoken, hashrefreshtoken bool
	var access_token, refresh_token string

	for _, w := range rr.Result().Cookies() {
		if w.Name == "access_token" {
			access_token = w.Value
			hasaccesstoken = true
		}

		if w.Name == "refresh_token" {
			access_token = w.Value
			hasaccesstoken = true
		}
	}

	if hasaccesstoken && hashrefreshtoken {
		t.Fatal("expected false got true")
	}

	if access_token != "" && refresh_token != "" {
		t.Fatal("value still in cookie")
	}
}

func TestHandleRefreshToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh-token", nil)
	rr := httptest.NewRecorder()

	http.SetCookie(rr, &http.Cookie{
		Name:   "refresh_token",
		Value:  "12345678",
		Path:   "/",
		MaxAge: 60 * 60,
	})

	a := &Api{
		logger:      &testLogger{},
		objectStore: &testObjectStore{},
		store:       &testStore{},
	}

	a.HandleRefreshToken(rr, req)

	var hasaccesstoken, hashrefreshtoken bool
	var access_token, refresh_token string

	for _, w := range rr.Result().Cookies() {
		if w.Name == "access_token" {
			access_token = w.Value
			hasaccesstoken = true
		}

		if w.Name == "refresh_token" {
			access_token = w.Value
			hasaccesstoken = true
		}
	}

	if !hasaccesstoken && !hashrefreshtoken {
		t.Fatal("expected true got false")
	}

	if access_token == "" && refresh_token == "" {
		t.Fatal("token not set in cookies")
	}
}
