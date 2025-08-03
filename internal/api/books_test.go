package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/oseayemenre/pagesy/internal/models"
	"github.com/oseayemenre/pagesy/internal/store"
)

func TestHandleUploadBooksService(t *testing.T) {
	tests := []struct {
		name            string
		formFields      map[string]string
		coverSizeBytes  int
		coverType       string
		uploadFileFunc  func(ctx context.Context, file io.Reader, id string) (string, error)
		uploadBookFunc  func(ctx context.Context, book *models.Book) (*uuid.UUID, error)
		updateImageFunc func(ctx context.Context, url string, id string) error
		expectedCode    int
	}{
		{
			name: "should return 400 if form data could not be validated",
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
			name: "should return 413 if book cover is too large",
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
			name: "should return 400 if book cover type could not be validated",
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
			name: "should return 404 if book genre doesn't exist",
			formFields: map[string]string{
				"name":                     "test book",
				"description":              "test book description",
				"genre":                    "non-existent genre",
				"language":                 "English",
				"release_schedule_day":     "Sunday",
				"release_schedule_chapter": "2",
				"chapter_title":            "test chapter title",
				"chapter_content":          "test chapter content",
			},
			coverSizeBytes: 2 * 1024 * 1024,
			coverType:      "cover.jpg",
			uploadBookFunc: func(ctx context.Context, book *models.Book) (*uuid.UUID, error) {
				return nil, store.ErrGenresNotFound
			},
			expectedCode: http.StatusNotFound,
		},
		{
			name: "should return 500 if something went wrong while uploading book",
			formFields: map[string]string{
				"name":                     "test book",
				"description":              "test book description",
				"genre":                    "non-existent genre",
				"language":                 "English",
				"release_schedule_day":     "Sunday",
				"release_schedule_chapter": "2",
				"chapter_title":            "test chapter title",
				"chapter_content":          "test chapter content",
			},
			coverSizeBytes: 2 * 1024 * 1024,
			coverType:      "cover.jpg",
			uploadBookFunc: func(ctx context.Context, book *models.Book) (*uuid.UUID, error) {
				return nil, errors.New("random-error")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "should return 500 something went wrong while uploading file to object store",
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
			uploadFileFunc: func(ctx context.Context, file io.Reader, id string) (string, error) {
				return "", errors.New("random-error")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "should return 500 something went wrong while updating book image",
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
			updateImageFunc: func(ctx context.Context, url, id string) error {
				return errors.New("random-error")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "should return 201 and upload book",
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
			a := &Api{
				logger: &testLogger{},
				objectStore: &testObjectStore{
					uploadFileFunc: tt.uploadFileFunc,
				},
				store: &testStore{
					uploadBookFunc:  tt.uploadBookFunc,
					updateImageFunc: tt.updateImageFunc,
				},
			}

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

			a.HandleUploadBook(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleGetBooksStatsService(t *testing.T) {
	tests := []struct {
		name              string
		url               string
		getBooksStatsFunc func(ctx context.Context, id string, offset int, limit int) ([]models.Book, error)
		expectedCode      int
	}{
		{
			name:         "should return 400 if offset is not set",
			url:          "/api/v1/books/stats",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "should return 400 if limit is not set",
			url:          "/api/v1/books/stats?offset=0",
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "should return 200 if creator has no books",
			url:  "/api/v1/books/stats?offset=0&limit=5",
			getBooksStatsFunc: func(ctx context.Context, id string, offset, limit int) ([]models.Book, error) {
				return nil, store.ErrCreatorsBooksNotFound
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "should return 500 if something went wrong while getting books",
			url:  "/api/v1/books/stats?offset=0&limit=5",
			getBooksStatsFunc: func(ctx context.Context, id string, offset, limit int) ([]models.Book, error) {
				return nil, errors.New("something went wrong")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "should return 200 and get books",
			url:          "/api/v1/books/stats?offset=0&limit=5",
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Api{
				logger:      &testLogger{},
				objectStore: &testObjectStore{},
				store: &testStore{
					getBooksStatsFunc: tt.getBooksStatsFunc,
				},
			}

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)

			req = req.WithContext(context.WithValue(context.TODO(), "user", &models.User{
				Id:   uuid.New(),
				Role: "",
			}))

			rr := httptest.NewRecorder()

			a.HandleGetBooksStats(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleGetBooksService(t *testing.T) {
	tests := []struct {
		name                           string
		url                            string
		getBooksByGenreFunc            func(ctx context.Context, genre []string, offset int, limit int, sort string, order string) ([]models.Book, error)
		getBooksByLanguageFunc         func(ctx context.Context, language []string, offset int, limit int, sort string, order string) ([]models.Book, error)
		getBooksByGenreAndLanguageFunc func(ctx context.Context, genre []string, language []string, offset int, limit int, sort string, order string) ([]models.Book, error)
		getAllBooksFunc                func(ctx context.Context, offset int, limit int, sort string, order string) ([]models.Book, error)
		expectedCode                   int
	}{
		{
			name:         "should return 400 if offset is not set",
			url:          "/api/v1/books/",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "should return 400 if limit is not set",
			url:          "/api/v1/books/?offset=0",
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "should return 500 if something went wrong getting books under a genre",
			getBooksByGenreFunc: func(ctx context.Context, genre []string, offset, limit int, sort, order string) ([]models.Book, error) {
				return nil, errors.New("something went wrong")
			},
			url:          "/api/v1/books/?offset=0&limit=5&genre=action",
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "should return 200 if there's no book under genre",
			url:  "/api/v1/books/?offset=0&limit=5&genre=action",
			getBooksByGenreFunc: func(ctx context.Context, genre []string, offset, limit int, sort, order string) ([]models.Book, error) {
				return nil, store.ErrNoBooksUnderThisGenre
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "should return 500 if something went wrong getting books under a language",
			url:  "/api/v1/books/?offset=0&limit=5&language=English",
			getBooksByLanguageFunc: func(ctx context.Context, language []string, offset, limit int, sort, order string) ([]models.Book, error) {
				return nil, errors.New("something went wrong")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "should return 200 if there's no book under language",
			url:  "/api/v1/books/?offset=0&limit=5&language=English",
			getBooksByLanguageFunc: func(ctx context.Context, language []string, offset, limit int, sort, order string) ([]models.Book, error) {
				return nil, store.ErrNoBooksUnderThisLanguage
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "should return 500 if something went wrong getting books under genre and language",
			url:  "/api/v1/books/?offset=0&limit=5&genre=Action&language=English",
			getBooksByGenreAndLanguageFunc: func(ctx context.Context, genre, language []string, offset, limit int, sort, order string) ([]models.Book, error) {
				return nil, errors.New("something went wrong")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "should return 200 if there's no book under genre and language",
			url:  "/api/v1/books/?offset=0&limit=5&genre=Action&language=English",
			getBooksByGenreAndLanguageFunc: func(ctx context.Context, genre, language []string, offset, limit int, sort, order string) ([]models.Book, error) {
				return nil, store.ErrNoBooksUnderThisGenreAndLanguage
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "should return 500 if something went wrong getting books",
			url:  "/api/v1/books/?offset=0&limit=5",
			getAllBooksFunc: func(ctx context.Context, offset, limit int, sort, order string) ([]models.Book, error) {
				return nil, errors.New("something went wrong")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "should return 200 get books",
			url:          "/api/v1/books/?offset=0&limit=5",
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Api{
				logger:      &testLogger{},
				objectStore: &testObjectStore{},
				store: &testStore{
					getBooksByGenreFunc:            tt.getBooksByGenreFunc,
					getBooksByLanguageFunc:         tt.getBooksByLanguageFunc,
					getBooksByGenreAndLanguageFunc: tt.getBooksByGenreAndLanguageFunc,
					getAllBooksFunc:                tt.getAllBooksFunc,
				},
			}

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)

			req = req.WithContext(context.WithValue(context.TODO(), "user", &models.User{
				Id:   uuid.New(),
				Role: "",
			}))

			rr := httptest.NewRecorder()

			a.HandleGetBooks(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleGetBookService(t *testing.T) {
	tests := []struct {
		name         string
		getBookFunc  func(ctx context.Context, id string) (*models.Book, error)
		expectedCode int
	}{
		{
			name: "should return 404 if book is not found",
			getBookFunc: func(ctx context.Context, id string) (*models.Book, error) {
				return nil, store.ErrBookNotFound
			},
			expectedCode: http.StatusNotFound,
		},
		{
			name: "should return 500 if something went wrong",
			getBookFunc: func(ctx context.Context, id string) (*models.Book, error) {
				return nil, errors.New("something went wrong")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "should return 200 and get book",
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Api{
				logger:      &testLogger{},
				objectStore: &testObjectStore{},
				store: &testStore{
					getBookFunc: tt.getBookFunc,
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/books/1", nil)

			req = req.WithContext(context.WithValue(context.TODO(), "user", &models.User{
				Id:   uuid.New(),
				Role: "",
			}))

			rr := httptest.NewRecorder()

			a.HandleGetBook(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleDeleteBookService(t *testing.T) {
	tests := []struct {
		name           string
		expectedCode   int
		deleteBookFunc func(ctx context.Context, bookId string, userId string) error
	}{
		{
			name: "should return 500 if something went wrong",
			deleteBookFunc: func(ctx context.Context, bookId string, userId string) error {
				return errors.New("something went wrong")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "should return 204 and delete book",
			expectedCode: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Api{
				logger:      &testLogger{},
				objectStore: &testObjectStore{},
				store: &testStore{
					deleteBookFunc: tt.deleteBookFunc,
				},
			}

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/books/1", nil)

			req = req.WithContext(context.WithValue(context.TODO(), "user", &models.User{
				Id:   uuid.New(),
				Role: "",
			}))

			rr := httptest.NewRecorder()

			a.HandleDeleteBook(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleEditBookService(t *testing.T) {
	tests := []struct {
		name           string
		formFields     map[string]string
		coverSizeBytes int
		coverType      string
		bookId         string
		uploadFileFunc func(ctx context.Context, file io.Reader, id string) (string, error)
		editBookFunc   func(ctx context.Context, book *models.Book) error
		expectedCode   int
	}{
		{
			name: "should return 400 if release schedule days and release schedule chapters are not the same",
			formFields: map[string]string{
				"release_schedule_day":     "Sunday",
				"release_schedule_chapter": "2,4",
			},
			coverSizeBytes: 2 * 1024 * 1024,
			coverType:      "cover.jpg",
			expectedCode:   http.StatusBadRequest,
		},
		{
			name:           "should return 413 if book cover is too large",
			formFields:     map[string]string{},
			coverSizeBytes: 4 * 1024 * 1024,
			coverType:      "image.pdf",
			expectedCode:   http.StatusRequestEntityTooLarge,
		},
		{
			name:           "should return 400 if book cover is not an image",
			formFields:     map[string]string{},
			coverSizeBytes: 2 * 1024 * 1024,
			coverType:      "image.pdf",
			expectedCode:   http.StatusBadRequest,
		},
		{
			name:           "should return 500 something went wrong while uploading file to object store",
			formFields:     map[string]string{},
			coverSizeBytes: 2 * 1024 * 1024,
			coverType:      "cover.jpg",
			uploadFileFunc: func(ctx context.Context, file io.Reader, id string) (string, error) {
				return "", errors.New("random-error")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "should return 400 if chapter is not a string that can be converted to an int",
			formFields: map[string]string{
				"release_schedule_day":     "Sunday",
				"release_schedule_chapter": "two",
			},
			bookId:       uuid.New().String(),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:       "should return 400 if no field was passed to be edited",
			formFields: map[string]string{},
			editBookFunc: func(ctx context.Context, book *models.Book) error {
				return store.ErrShouldAtLeasePassOneFieldToUpdate
			},
			bookId:       uuid.New().String(),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:       "should return 404 if book was not found",
			formFields: map[string]string{},
			editBookFunc: func(ctx context.Context, book *models.Book) error {
				return store.ErrBookNotFound
			},
			bookId:       uuid.New().String(),
			expectedCode: http.StatusNotFound,
		},
		{
			name: "should return 500 if something went wrong",
			formFields: map[string]string{
				"release_schedule_day":     "Sunday",
				"release_schedule_chapter": "2",
			},
			bookId: uuid.New().String(),
			editBookFunc: func(ctx context.Context, book *models.Book) error {
				return errors.New("something went wrong")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "should return 200 and edit book",
			formFields: map[string]string{
				"release_schedule_day":     "Sunday",
				"release_schedule_chapter": "2",
			},
			bookId:       uuid.New().String(),
			expectedCode: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Api{
				logger: &testLogger{},
				objectStore: &testObjectStore{
					uploadFileFunc: tt.uploadFileFunc,
				},
				store: &testStore{
					editBookFunc: tt.editBookFunc,
				},
			}

			var file io.Writer

			buf := bytes.Buffer{}
			writer := multipart.NewWriter(&buf)

			if tt.coverType != "" {
				file, _ = writer.CreateFormFile("book_cover", tt.coverType)
				if strings.HasSuffix(tt.coverType, "jpg") {
					file.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46})
				} else {
					file.Write(make([]byte, tt.coverSizeBytes))
				}
			}

			for key, val := range tt.formFields {
				writer.WriteField(key, val)
			}

			writer.Close()

			req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/books/%s", tt.bookId), &buf)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req = req.WithContext(context.WithValue(context.TODO(), "user", &models.User{
				Id:   uuid.New(),
				Role: "",
			}))

			ctx := chi.NewRouteContext()
			ctx.URLParams.Add("bookId", tt.bookId)

			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))

			rr := httptest.NewRecorder()

			a.HandleEditBook(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleApproveBookService(t *testing.T) {
	tests := []struct {
		name            string
		body            any
		approveBookFunc func(ctx context.Context, id string, approve bool) error
		expectedCode    int
	}{
		{
			name:         "should return 400 if json couldn't be decoded",
			body:         struct{ Approve int }{Approve: 0},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "should return 400 if data could not be validated",
			body:         struct{ key string }{key: "value"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "should return 500 if there was an error approving book",
			body: &models.ApproveBookParam{Approve: true},
			approveBookFunc: func(ctx context.Context, id string, approve bool) error {
				return errors.New("something went wrong")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "should return 204 and approve book",
			body:         &models.ApproveBookParam{Approve: true},
			expectedCode: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Api{
				logger:      &testLogger{},
				objectStore: &testObjectStore{},
				store: &testStore{
					approveBookFunc: tt.approveBookFunc,
				},
			}

			marshal_body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/books/1/approval", bytes.NewBuffer(marshal_body))
			rr := httptest.NewRecorder()

			a.HandleApproveBook(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleMarkBookAsCompleteService(t *testing.T) {
	tests := []struct {
		name             string
		body             any
		expectedCode     int
		completeBookFunc func(ctx context.Context, id string, complete bool) error
	}{
		{
			name:         "should return 400 if json couldn't be decoded",
			body:         struct{ Completed int }{Completed: 0},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "should return 400 if data could not be validated",
			body:         struct{ key string }{key: "value"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "should return 500 if there was an error marking book as comlete",
			body: &models.MarkAsCompleteParam{Completed: true},
			completeBookFunc: func(ctx context.Context, id string, complete bool) error {
				return errors.New("something went wrong")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "should return 204 and mark book as complete",
			body:         &models.MarkAsCompleteParam{Completed: true},
			expectedCode: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Api{
				logger:      &testLogger{},
				objectStore: &testObjectStore{},
				store: &testStore{
					completeBookFunc: tt.completeBookFunc,
				},
			}

			marshal_body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/books/1/complete", bytes.NewBuffer(marshal_body))
			rr := httptest.NewRecorder()

			a.HandleMarkBookAsComplete(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleGetRecentReads(t *testing.T) {
	tests := []struct {
		name               string
		url                string
		getRecentReadsFunc func(ctx context.Context, id string, offset int, limit int) ([]models.Book, error)
		expectedCode       int
	}{
		{
			name:         "should return 400 if offset is not set",
			url:          "/api/v1/books/recents",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "should return 400 if limit is not set",
			url:          "/api/v1/books/recents?offset=0",
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "should return 200 if there are no books in recents",
			url:  "/api/v1/books/recents?offset=0&limit=5",
			getRecentReadsFunc: func(ctx context.Context, id string, offset, limit int) ([]models.Book, error) {
				return nil, store.ErrNoBooksInRecents
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "should return 500 if something went wrong",
			url:  "/api/v1/books/recents?offset=0&limit=5",
			getRecentReadsFunc: func(ctx context.Context, id string, offset, limit int) ([]models.Book, error) {
				return nil, errors.New("something went wrong")
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "should return 200 and get recent reads",
			url:          "/api/v1/books/recents?offset=0&limit=5",
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Api{
				logger:      &testLogger{},
				objectStore: &testObjectStore{},
				store: &testStore{
					getRecentReadsFunc: tt.getRecentReadsFunc,
				},
			}

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)

			req = req.WithContext(context.WithValue(context.TODO(), "user", &models.User{
				Id:   uuid.New(),
				Role: "",
			}))

			rr := httptest.NewRecorder()

			a.HandleGetRecentReads(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}
