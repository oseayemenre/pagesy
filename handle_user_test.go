package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleGetProfile(t *testing.T) {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	db := connectTestDb(t)
	id := createAndCleanUpUser(t, db)
	token, err := createJWTToken(id)
	if err != nil {
		t.Fatal(err.Error())
	}

	tests := []struct {
		name         string
		cookie_name  string
		cookie_value string
		expectedCode int
	}{
		{
			name:         "no access token cookie",
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "invalid/malformed token",
			cookie_name:  "access_token",
			cookie_value: "invalid token",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "get profile",
			cookie_name:  "access_token",
			cookie_value: token,
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
			r.AddCookie(&http.Cookie{Name: tc.cookie_name, Value: tc.cookie_value})

			rr := httptest.NewRecorder()

			svr := newServer(nil, db, nil, nil)
			svr.router.ServeHTTP(rr, r)

			if rr.Code != tc.expectedCode {
				t.Fatalf("expected %d, got %d", tc.expectedCode, rr.Code)
			}
		})
	}
}
