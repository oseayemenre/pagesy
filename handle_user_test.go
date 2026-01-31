package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHandleGetProfile(t *testing.T) {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	db := connectTestDb(t)
	var id string

	hash, _ := bcrypt.GenerateFromPassword([]byte("test_password"), bcrypt.DefaultCost)
	query :=
		`
			INSERT INTO users (display_name, email, password) VALUES ('test_display', 'test@test.com', $1) RETURNING id;
		`
	if err := db.QueryRowContext(context.Background(), query, hash).Scan(&id); err != nil {
		t.Errorf("error creating new user, %v", err)
	}

	t.Cleanup(func() {
		query :=
			`
				DELETE FROM users WHERE email = 'test@test.com' OR email = 'user@user.com';
			`
		if _, err := db.ExecContext(context.Background(), query); err != nil {
			t.Errorf("error deleting users, %v", err)
		}
	})

	token, err := createJWTToken(id)
	if err != nil {
		t.Fatal(err.Error())
	}

	tests := []struct {
		name         string
		cookie_name  string
		cookie_value string
		expectCode   int
	}{
		{
			name:       "no access token cookie",
			expectCode: http.StatusNotFound,
		},
		{
			name:         "invalid/malformed token",
			cookie_name:  "access_token",
			cookie_value: "invalid token",
			expectCode:   http.StatusBadRequest,
		},
		{
			name:         "get profile",
			cookie_name:  "access_token",
			cookie_value: token,
			expectCode:   http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
			r.AddCookie(&http.Cookie{Name: tc.cookie_name, Value: tc.cookie_value})

			rr := httptest.NewRecorder()

			svr := newServer(nil, db, nil, nil)
			svr.router.ServeHTTP(rr, r)

			if rr.Code != tc.expectCode {
				t.Fatalf("expected %d, got %d", tc.expectCode, rr.Code)
			}
		})
	}
}
