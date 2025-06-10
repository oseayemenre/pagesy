package routes

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oseayemenre/pagesy/internal/models"
	"github.com/oseayemenre/pagesy/internal/shared"
)

type testLogger struct{}

func (l *testLogger) Info(msg string, args ...any)  {}
func (l *testLogger) Error(msg string, args ...any) {}
func (l *testLogger) Warn(msg string, args ...any)  {}

type testObjectStore struct{}

func (s *testObjectStore) UploadFile(ctx context.Context, file io.Reader, id string) (string, error) {
	return "http://mock-url.com", nil
}

type testStore struct{}

func (s *testStore) UploadBook(ctx context.Context, book *models.Book) error {
	return nil
}

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
				if strings.HasPrefix(key, "release_schedule") || key == "genre" {
					writer.WriteField(key, val)
				} else {
					writer.WriteField(key, val)
				}
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
