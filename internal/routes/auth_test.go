package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oseayemenre/pagesy/internal/config"
	"github.com/oseayemenre/pagesy/internal/models"
	"github.com/oseayemenre/pagesy/internal/shared"
)

func TestHandleRegister(t *testing.T) {
	tests := []struct {
		name         string
		body         *models.HandleRegisterParams
		expectedCode int
		userExists   bool
	}{
		{
			name: "should fail if fields could not be validated",
			body: &models.HandleRegisterParams{
				Email:    "fail_email",
				Password: "12345678",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "should fail if user already exists",
			body: &models.HandleRegisterParams{
				Email:    "test@test.com",
				Password: "12345678",
			},
			expectedCode: http.StatusConflict,
			userExists:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				Server: &shared.Server{
					Logger:      &testLogger{},
					ObjectStore: &testObjectStore{},
					Store: &testStore{
						userExists: tt.userExists,
					},
				},
			}

			data, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/register", bytes.NewBuffer(data))
			rr := httptest.NewRecorder()

			s.HandleRegister(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleLogin(t *testing.T) {
	tests := []struct {
		name         string
		body         *models.HandleLoginParams
		expectedCode int
		userExists   bool
	}{
		{
			name: "should fail if email and username is empty",
			body: &models.HandleLoginParams{
				Password: "12345678",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "should fail if fields could not be validated",
			body: &models.HandleLoginParams{
				Email: "fail_email",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "should fail if password doesn't match",
			body: &models.HandleLoginParams{
				Email:    "test@test.com",
				Password: "_",
			},
			expectedCode: http.StatusUnauthorized,
			userExists:   true,
		},
		{
			name: "should login user",
			body: &models.HandleLoginParams{
				Email:    "test@test.com",
				Password: "password",
			},
			expectedCode: http.StatusNoContent,
			userExists:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				Server: &shared.Server{
					Logger:      &testLogger{},
					ObjectStore: &testObjectStore{},
					Store: &testStore{
						userExists: tt.userExists,
					},
					Config: &config.Config{
						Jwt_secret: "secret",
					},
				},
			}

			data, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(data))
			rr := httptest.NewRecorder()

			s.HandleLogin(rr, req)

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

	s := &Server{
		Server: &shared.Server{
			Logger:      &testLogger{},
			ObjectStore: &testObjectStore{},
			Store:       &testStore{},
		},
	}

	s.HandleLogout(rr, req)

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

	s := &Server{
		Server: &shared.Server{
			Logger:      &testLogger{},
			ObjectStore: &testObjectStore{},
			Store:       &testStore{},
		},
	}

	s.HandleRefreshToken(rr, req)

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
