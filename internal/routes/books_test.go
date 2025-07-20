package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/oseayemenre/pagesy/internal/models"
	"github.com/oseayemenre/pagesy/internal/shared"
)

func TestHandleUploadBooksService(t *testing.T) {
	s := &Server{
		Server: &shared.Server{
			Logger:      &testLogger{},
			ObjectStore: &testObjectStore{},
			Store:       &testStore{},
		},
	}

	tests := []struct {
		name           string
		formFields     map[string]string
		coverSizeBytes int
		coverType      string
		expectedCode   int
	}{
		{
			name: "it should throw an error if form data could not be validated",
			formFields: map[string]string{
				"name":                     "test book",
				"description":              "test book description",
				"genre":                    "Romance",
				"language":                 "English",
				"release_schedule_day":     "Sunday",
				"release_schedule_chapter": "2",
				"chapter_content":          "test chapter content",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "it should throw an error if book cover is too large",
			formFields: map[string]string{
				"name":                     "test book",
				"description":              "test book description",
				"genre":                    "Romance",
				"language":                 "English",
				"release_schedule_day":     "Sunday",
				"release_schedule_chapter": "2",
				"chapter_title":            "test chapter title",
				"chapter_content":          "test chapter content",
			},
			coverSizeBytes: 4 * 1024 * 1024,
			coverType:      "cover.pdf",
			expectedCode:   http.StatusRequestEntityTooLarge,
		},
		{
			name: "it should throw an error if book cover type could not be validated",
			formFields: map[string]string{
				"name":                     "test book",
				"description":              "test book description",
				"genre":                    "Romance",
				"language":                 "English",
				"release_schedule_day":     "Sunday",
				"release_schedule_chapter": "2",
				"chapter_title":            "test chapter title",
				"chapter_content":          "test chapter content",
			},
			coverSizeBytes: 2 * 1024 * 1024,
			coverType:      "cover.pdf",
			expectedCode:   http.StatusBadRequest,
		},
		{
			name: "it should succesfully upload book",
			formFields: map[string]string{
				"name":                     "test book",
				"description":              "test book description",
				"genre":                    "Romance",
				"language":                 "English",
				"release_schedule_day":     "Sunday",
				"release_schedule_chapter": "2",
				"chapter_title":            "test chapter title",
				"chapter_content":          "test chapter content",
			},
			coverSizeBytes: 2 * 1024 * 1024,
			coverType:      "cover.jpg",
			expectedCode:   http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			for key, val := range tt.formFields {
				writer.WriteField(key, val)
			}

			file, _ := writer.CreateFormFile("book_cover", tt.coverType)

			if strings.HasSuffix(tt.coverType, "jpg") {
				file.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46})
			} else {
				file.Write(make([]byte, tt.coverSizeBytes))
			}

			writer.Close()

			req := httptest.NewRequest(http.MethodPost, "/api/v1/books", body)
			req = req.WithContext(context.WithValue(context.TODO(), "user", &models.User{
				Id:   uuid.New(),
				Role: "",
			}))

			req.Header.Set("Content-Type", writer.FormDataContentType())

			rr := httptest.NewRecorder()

			s.HandleUploadBooks(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func helperTestGetBooks(t *testing.T, path string, handler http.HandlerFunc) {
	t.Helper()
	tests := []struct {
		name         string
		offset       string
		limit        string
		expectedCode int
	}{
		{
			name:         "should throw an error if offset is not set",
			offset:       "",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "should throw an error if limit is not set",
			offset:       "0",
			limit:        "",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "should get books",
			offset:       "0",
			limit:        "10",
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf(path, tt.offset, tt.limit), nil)

			req = req.WithContext(context.WithValue(context.TODO(), "user", &models.User{
				Id:   uuid.New(),
				Role: "",
			}))

			rr := httptest.NewRecorder()

			handler(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleGetBooksService(t *testing.T) {
	s := &Server{
		Server: &shared.Server{
			Logger:      &testLogger{},
			ObjectStore: &testObjectStore{},
			Store:       &testStore{},
		},
	}

	helperTestGetBooks(t, "/api/v1/books/?offset=%s&limit=%s", s.HandleGetBooks)
}

func TestHandleGetBooksStatsService(t *testing.T) {
	s := &Server{
		Server: &shared.Server{
			Logger:      &testLogger{},
			ObjectStore: &testObjectStore{},
			Store:       &testStore{},
		},
	}

	helperTestGetBooks(t, "/api/v1/books/stats?offset=%s&limit=%s", s.HandleGetBooksStats)
}

func TestHandleEditBookService(t *testing.T) {
	s := &Server{
		Server: &shared.Server{
			Logger:      &testLogger{},
			ObjectStore: &testObjectStore{},
			Store:       &testStore{},
		},
	}

	tests := []struct {
		name           string
		formFields     map[string]string
		coverSizeBytes int
		coverType      string
		errCode        int
	}{
		{
			name:           "should throw an error if book cover is too large",
			formFields:     map[string]string{},
			coverSizeBytes: 4 * 1024 * 1024,
			coverType:      "image.pdf",
			errCode:        http.StatusRequestEntityTooLarge,
		},
		{
			name:           "should throw an error if book cover is not an image",
			formFields:     map[string]string{},
			coverSizeBytes: 2 * 1024 * 1024,
			coverType:      "image.pdf",
			errCode:        http.StatusBadRequest,
		},
		{
			name: "should throw an error if release schedule days and release schedule chapters are not the same",
			formFields: map[string]string{
				"release_schedule_day":     "Sunday",
				"release_schedule_chapter": "2,4",
			},
			coverSizeBytes: 2 * 1024 * 1024,
			coverType:      "cover.jpg",
			errCode:        http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.Buffer{}
			writer := multipart.NewWriter(&buf)
			file, _ := writer.CreateFormFile("book_cover", tt.coverType)

			for key, val := range tt.formFields {
				writer.WriteField(key, val)
			}

			if strings.HasSuffix(tt.coverType, "jpg") {
				file.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46})
			} else {
				file.Write(make([]byte, tt.coverSizeBytes))
			}

			writer.Close()

			req := httptest.NewRequest(http.MethodPatch, "/api/v1/books/1", &buf)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			rr := httptest.NewRecorder()

			s.HandleEditBook(rr, req)

			if rr.Code != tt.errCode {
				t.Fatalf("expected %d, got %d", tt.errCode, rr.Code)
			}
		})
	}
}

func TestHandleMarkBookAsCompleteService(t *testing.T) {
	s := &Server{
		Server: &shared.Server{
			Logger:      &testLogger{},
			ObjectStore: &testObjectStore{},
			Store:       &testStore{},
		},
	}

	tests := []struct {
		name         string
		body         any
		expectedCode int
	}{
		{
			name:         "should return an error if data could not be validated",
			body:         struct{ key string }{key: "value"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "should mark book as complete",
			body:         &models.MarkAsCompleteParam{Completed: true},
			expectedCode: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marshal_body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/books/1/complete", bytes.NewBuffer(marshal_body))
			rr := httptest.NewRecorder()

			s.HandleMarkBookAsComplete(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleApproveBookService(t *testing.T) {
	s := &Server{
		Server: &shared.Server{
			Logger:      &testLogger{},
			ObjectStore: &testObjectStore{},
			Store:       &testStore{},
		},
	}

	tests := []struct {
		name         string
		body         any
		expectedCode int
	}{
		{
			name:         "should return an error if data could not be validated",
			body:         struct{ key string }{key: "value"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "should approve book",
			body:         &models.ApproveBookParam{Approve: true},
			expectedCode: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marshal_body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/books/1/approval", bytes.NewBuffer(marshal_body))
			rr := httptest.NewRecorder()

			s.HandleApproveBook(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleGetRecentReads(t *testing.T) {
	s := &Server{
		Server: &shared.Server{
			Logger:      &testLogger{},
			ObjectStore: &testObjectStore{},
			Store:       &testStore{},
		},
	}

	helperTestGetBooks(t, "/api/v1/books/recents?offset=%s&limit=%s", s.HandleGetRecentReads)
}
