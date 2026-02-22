package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func createAndCleanUpFollowed(t *testing.T, db *sql.DB) string {
	var id string
	hash, _ := bcrypt.GenerateFromPassword([]byte("follower_password"), bcrypt.DefaultCost)
	query :=
		`
			INSERT INTO users (display_name, email, password) VALUES ('follower display', 'follower@follower.com', $1) RETURNING id;
		`
	if err := db.QueryRowContext(context.Background(), query, hash).Scan(&id); err != nil {
		t.Errorf("error creating new user, %v", err)
	}

	t.Cleanup(func() {
		query :=
			`
				DELETE FROM users WHERE email = 'follower@follower.com';
			`
		if _, err := db.ExecContext(context.Background(), query); err != nil {
			t.Errorf("error deleting users, %v", err)
		}
	})
	return id
}

func TestHandleFollowUser(t *testing.T) {
	db := connectTestDb(t)
	followerID := createAndCleanUpUser(t, db)
	token, err := createJWTToken(followerID, 5*time.Second)
	if err != nil {
		t.Fatal(err.Error())
	}

	tests := []struct {
		name         string
		cookieName   string
		cookieValue  string
		followerID   string
		userID       string
		expectedCode int
	}{
		{
			name:         "no access token cookie",
			userID:       uuid.NewString(),
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "invalid/malformed token",
			cookieName:   "access_token",
			cookieValue:  "invalid token",
			userID:       uuid.NewString(),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "user cannot follow themselves",
			cookieName:   "access_token",
			cookieValue:  token,
			followerID:   followerID,
			userID:       followerID,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "user not found",
			cookieName:   "access_token",
			cookieValue:  token,
			followerID:   followerID,
			userID:       uuid.NewString(),
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "follow user",
			cookieName:   "access_token",
			cookieValue:  token,
			followerID:   followerID,
			userID:       createAndCleanUpFollowed(t, db),
			expectedCode: http.StatusNoContent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%v/follow", tc.userID), nil)
			r.AddCookie(&http.Cookie{Name: tc.cookieName, Value: tc.cookieValue})
			rr := httptest.NewRecorder()

			svr := newServer(nil, db, nil, nil)
			svr.router.ServeHTTP(rr, r)

			if rr.Code != tc.expectedCode {
				t.Fatalf("expected %d, got %d", tc.expectedCode, rr.Code)
			}
		})
	}
}
