package routes

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
			req.Header.Set("Content-Type", writer.FormDataContentType())

			rr := httptest.NewRecorder()

			s.HandleUploadBooks(rr, req)

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
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/books/?offset=%s&limit=%s", tt.offset, tt.limit), nil)
			rr := httptest.NewRecorder()

			s.HandleGetBooks(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleGetBooksStatsService(t *testing.T) {
	s := &Server{
		Server: &shared.Server{
			Logger:      &testLogger{},
			ObjectStore: &testObjectStore{},
			Store:       &testStore{},
		},
	}

	tests := []struct {
		name         string
		offset       string
		limit        string
		expectedCode int
	}{
		{
			name:         "it should throw an error if offset isn't sent in query",
			offset:       "",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "it should throw an error if limit isn't sent in query",
			offset:       "0",
			limit:        "",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "it should get book stats succesfully",
			offset:       "0",
			limit:        "4",
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/books/stats?offset=%s&limit=%s", tt.offset, tt.limit), nil)
			rr := httptest.NewRecorder()

			s.HandleGetBooksStats(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
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

func HandleApproveBookService(t *testing.T) {
	s := &Server{
		Server: &shared.Server{
			Logger:      &testLogger{},
			ObjectStore: &testObjectStore{},
			Store:       &testStore{},
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/books/1/approval", bytes.NewBuffer([]byte("")))
	rr := httptest.NewRecorder()

	s.HandleApproveBook(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func HandleMarkBookAsCompleteService(t *testing.T) {
	s := &Server{
		Server: &shared.Server{
			Logger:      &testLogger{},
			ObjectStore: &testObjectStore{},
			Store:       &testStore{},
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/books/1/complete", bytes.NewBuffer([]byte("")))
	rr := httptest.NewRecorder()

	s.HandleApproveBook(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
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
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/books/recents?offset=%s&limit=%s", tt.offset, tt.limit), nil)
			rr := httptest.NewRecorder()

			s.HandleGetRecentReads(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}
