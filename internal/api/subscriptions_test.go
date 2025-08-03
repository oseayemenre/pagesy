package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/oseayemenre/pagesy/internal/models"
)

func TestHandleMarkBookForSubscription(t *testing.T) {
	tests := []struct {
		name                                 string
		body                                 any
		checkIfBookIsEligibleForSubscription func(ctx context.Context, bookId string) (bool, error)
		markBookForSubscription              func(ctx context.Context, bookId string, userId string, eligible bool) error
		expectedCode                         int
	}{
		{
			name:         "should return 400 if json cannot be decoded",
			body:         struct{ Subscription int }{Subscription: 0},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "should return 400 if data could not be validated",
			body:         struct{ key string }{key: "value"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "should return 500 if something went wrong while checking if book is eligible for subscribtion",
			body: &models.HandleMarkBookForSubscriptionParams{Subscription: true},
			checkIfBookIsEligibleForSubscription: func(ctx context.Context, bookId string) (bool, error) {
				return false, errors.New("something went wrong")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "should return 400 if book is not eligible for subscribtion",
			body: &models.HandleMarkBookForSubscriptionParams{Subscription: true},
			checkIfBookIsEligibleForSubscription: func(ctx context.Context, bookId string) (bool, error) {
				return false, nil
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "should return 500 if something went wrong while marking book for subscription",
			body: models.HandleMarkBookForSubscriptionParams{Subscription: true},
			markBookForSubscription: func(ctx context.Context, bookId, userId string, eligible bool) error {
				return errors.New("something went wrong")
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Api{
				logger:      &testLogger{},
				objectStore: &testObjectStore{},
				store: &testStore{
					checkIfBookIsEligibleForSubscriptionFunc: tt.checkIfBookIsEligibleForSubscription,
					markBookForSubscriptionFunc:              tt.markBookForSubscription,
				},
			}

			marshal_body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/books/1/subscriptions", bytes.NewBuffer(marshal_body))
			req = req.WithContext(context.WithValue(context.TODO(), "user", &models.User{
				Id:   uuid.New(),
				Role: "",
			}))
			rr := httptest.NewRecorder()

			a.HandleMarkBookForSubscription(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}

}
