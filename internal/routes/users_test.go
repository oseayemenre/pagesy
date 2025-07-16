package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestHandleOnboarding(t *testing.T) {}
