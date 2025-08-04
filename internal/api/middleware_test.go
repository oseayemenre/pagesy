package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/oseayemenre/pagesy/internal/config"
	appjwt "github.com/oseayemenre/pagesy/internal/jwt"
	"github.com/oseayemenre/pagesy/internal/models"
	"github.com/oseayemenre/pagesy/internal/store"
)

func TestCheckPermissionMiddleware(t *testing.T) {
	secret := "secret"

	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, &appjwt.UserClaims{
		Id: uuid.New().String(),
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer:    "pagesy",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}).SignedString([]byte(secret))

	tests := []struct {
		name            string
		hasCookie       bool
		token           string
		getUserByIdFunc func(ctx context.Context, id string) (*models.User, error)
		expectedCode    int
	}{
		{
			name:         "should return 404 if no cookie is found",
			hasCookie:    false,
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "should return 401 if token is invalid",
			hasCookie:    true,
			token:        "token",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:      "should return 404 if user is not found",
			hasCookie: true,
			token:     token,
			getUserByIdFunc: func(ctx context.Context, id string) (*models.User, error) {
				return nil, store.ErrUserNotFound
			},
			expectedCode: http.StatusNotFound,
		},
		{
			name:      "should return 500 if something went wrong while checking if user exists",
			hasCookie: true,
			token:     token,
			getUserByIdFunc: func(ctx context.Context, id string) (*models.User, error) {
				return nil, errors.New("something went wrong")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:      "should return 403 if user doesnt have permission to access route",
			hasCookie: true,
			token:     token,
			getUserByIdFunc: func(ctx context.Context, id string) (*models.User, error) {
				return &models.User{
					Privileges: []string{},
				}, nil
			},
			expectedCode: http.StatusForbidden,
		},
		{
			name:      "should grant user access to handler and return 200",
			hasCookie: true,
			token:     token,
			getUserByIdFunc: func(ctx context.Context, id string) (*models.User, error) {
				return &models.User{
					Privileges: []string{PermissionUploadBooks},
				}, nil
			},
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Api{
				logger:      &testLogger{},
				objectStore: &testObjectStore{},
				store: &testStore{
					getUserByIdFunc: tt.getUserByIdFunc,
				},
				config: &config.Config{
					Jwt_secret: secret,
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/test-handler", nil)
			rr := httptest.NewRecorder()

			if tt.hasCookie == true {
				req.AddCookie(&http.Cookie{
					Name:     "access_token",
					Value:    tt.token,
					Path:     "/",
					MaxAge:   5 * 1000,
					Secure:   false,
					SameSite: http.SameSiteLaxMode,
				})
			}

			handler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}

			a.CheckPermission(PermissionUploadBooks)(http.HandlerFunc(handler)).ServeHTTP(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestRedirectIfCookieExistsAndIsValid(t *testing.T) {
	secret := "secret"

	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, &appjwt.UserClaims{
		Id: uuid.New().String(),
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer:    "pagesy",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}).SignedString([]byte(secret))

	tests := []struct {
		name         string
		hasCookie    bool
		token        string
		expectedCode int
	}{
		{
			name:         "move to the next handler if cookie doesn't exist",
			hasCookie:    false,
			expectedCode: http.StatusOK,
		},
		{
			name:         "should return 401 if token is invalid",
			hasCookie:    true,
			token:        "token",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "should return 302 if token is valid",
			hasCookie:    true,
			token:        token,
			expectedCode: http.StatusFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Api{
				logger:      &testLogger{},
				objectStore: &testObjectStore{},
				store:       &testStore{},
				config: &config.Config{
					Jwt_secret: secret,
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/test-handler", nil)
			rr := httptest.NewRecorder()

			if tt.hasCookie == true {
				req.AddCookie(&http.Cookie{
					Name:     "access_token",
					Value:    tt.token,
					Path:     "/",
					MaxAge:   5 * 1000,
					Secure:   false,
					SameSite: http.SameSiteLaxMode,
				})
			}

			handler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}

			a.RedirectIfCookieExistsAndIsValid(http.HandlerFunc(handler)).ServeHTTP(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}
