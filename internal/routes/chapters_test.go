package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/oseayemenre/pagesy/internal/models"
	"github.com/oseayemenre/pagesy/internal/shared"
)

func TestHandleUploadChapter(t *testing.T) {
	s := &Server{
		Server: &shared.Server{
			Logger:      &testLogger{},
			ObjectStore: &testObjectStore{},
			Store:       &testStore{},
		},
	}

	tests := []struct {
		name         string
		book_id      string
		chapter      *models.HandleUploadChapterParams
		expectedCode int
	}{
		{
			name: "it should fail if fields could not be validated",
			chapter: &models.HandleUploadChapterParams{
				Title: "",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:    "it should fail if book id is not a valid uuid",
			book_id: "1",
			chapter: &models.HandleUploadChapterParams{
				Title:      "test chapter title",
				Chapter_no: 1,
				Content:    "test chapter content",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:    "it should upload chapter succesfully",
			book_id: uuid.NewString(),
			chapter: &models.HandleUploadChapterParams{
				Title:      "test chapter title",
				Chapter_no: 1,
				Content:    "test chapter content",
			},
			expectedCode: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.chapter)

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/books/%s/chapters", tt.book_id), bytes.NewBuffer(body))
			req = req.WithContext(context.WithValue(context.TODO(), "user", &models.User{
				Id:   uuid.New(),
				Role: "",
			}))

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("bookId", tt.book_id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()

			s.HandleUploadChapter(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}

}
