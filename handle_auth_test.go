package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleAuthRegister(t *testing.T) {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	db := connectTestDb(t)
	createAndCleanUpUser(t, db)

	tests := []struct {
		name         string
		body         any
		expectedCode int
	}{
		{
			name:         "invalid/malformed json",
			body:         "bad request body",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "validation error",
			body:         struct{ name string }{name: "invalid structure"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "user already exists",
			body:         request{Email: "test@test.com", Password: "test_password"},
			expectedCode: http.StatusConflict,
		},
		{
			name:         "redirect",
			body:         request{Email: "user@user.com", Password: "user_password"},
			expectedCode: http.StatusFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload, _ := json.Marshal(tc.body)
			r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(payload))
			rr := httptest.NewRecorder()

			svr := newServer(nil, db, nil)
			svr.router.ServeHTTP(rr, r)

			if rr.Code != tc.expectedCode {
				t.Fatalf("expected %d, got %d", tc.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleAuthLogin(t *testing.T) {
	type request struct {
		Email    string `json:"email"`
		Username string `json:"username"`
		Password string `json:"password"`
	}

	db := connectTestDb(t)
	createAndCleanUpUser(t, db)

	tests := []struct {
		name         string
		body         any
		expectedCode int
	}{
		{
			name:         "invalid/malformed json",
			body:         "bad request body",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "validation error",
			body:         struct{ name string }{name: "invalid structure"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "user or email not in body",
			body:         request{Password: "test-password"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "user not found",
			body:         request{Email: "notfound@notfound.com", Password: ""},
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "incorrect password",
			body:         request{Email: "test@test.com", Password: "incorrect"},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "log in user",
			body:         request{Email: "test@test.com", Password: "test_password"},
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload, _ := json.Marshal(tc.body)
			r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(payload))
			rr := httptest.NewRecorder()

			svr := newServer(nil, db, nil)
			svr.router.ServeHTTP(rr, r)

			if rr.Code != tc.expectedCode {
				t.Fatalf("expected %d, got %d", tc.expectedCode, rr.Code)
			}
		})
	}
}
