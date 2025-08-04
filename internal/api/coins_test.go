package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/oseayemenre/pagesy/internal/config"
	"github.com/oseayemenre/pagesy/internal/models"
)

func TestHandleBuyCoins(t *testing.T) {
	tests := []struct {
		name         string
		body         any
		expectedCode int
	}{
		{
			name:         "should return 400 if json cannot be decoded",
			body:         struct{ Price_id int }{Price_id: 0},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "should return 400 if data could not be validated",
			body:         struct{ key string }{key: "value"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "should return 404 if price id is invalid",
			body:         &models.HandleBuyCoinsParams{Price_id: "invalid price id"},
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Api{
				logger:      &testLogger{},
				objectStore: &testObjectStore{},
				store:       &testStore{},
				config: &config.Config{
					Stripe_webhook_secret: "",
				},
			}

			marshal_body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/books/coins", bytes.NewBuffer(marshal_body))
			req = req.WithContext(context.WithValue(context.TODO(), "user", &models.User{
				Id:   uuid.New(),
				Role: "",
			}))
			rr := httptest.NewRecorder()

			a.HandleBuyCoins(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}
