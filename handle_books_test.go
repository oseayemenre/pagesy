package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestHandleUploadBook(t *testing.T) {
	type request struct {
		Name            string `validate:"required"`
		Description     string `validate:"required"`
		Genres          string `validate:"required"`
		Language        string `validate:"required"`
		ReleaseSchedule []releaseSchedule
		DraftChapter    draftChapter
	}

	db := connectTestDb(t)
	id := createAndCleanUpUser(t, db)
	token, err := createJWTToken(id)
	if err != nil {
		t.Fatal(err.Error())
	}

	createBook(t, id, db)

	tests := []struct {
		name         string
		cookieName   string
		cookieValue  string
		req          map[string]string
		expectedCode int
	}{
		{
			name:         "no access token cookie",
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "invalid/malformed token",
			cookieName:   "access_token",
			cookieValue:  "invalid token",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:        "chapter length and days not the same",
			cookieName:  "access_token",
			cookieValue: token,
			req: map[string]string{
				"name":                     "test book",
				"description":              "test book description",
				"genres":                   "Fantasy",
				"language":                 "English",
				"release_schedule_day":     "Sunday, Monday",
				"release_schedule_chapter": "2",
				"chapter_title":            "test chapter title",
				"chapter_content":          "test chapter content",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:        "chapter count is not convertible to int",
			cookieName:  "access_token",
			cookieValue: token,
			req: map[string]string{
				"name":                     "test book",
				"description":              "test book description",
				"genres":                   "Fantasy",
				"language":                 "English",
				"release_schedule_day":     "Sunday",
				"release_schedule_chapter": "two",
				"chapter_title":            "test chapter title",
				"chapter_content":          "test chapter content",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:        "validation error",
			cookieName:  "access_token",
			cookieValue: token,
			req: map[string]string{
				"name":                     "test book",
				"description":              "test book description",
				"language":                 "English",
				"release_schedule_day":     "Sunday",
				"release_schedule_chapter": "2",
				"chapter_title":            "test chapter title",
				"chapter_content":          "test chapter content",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:        "book name taken",
			cookieName:  "access_token",
			cookieValue: token,
			req: map[string]string{
				"name":                     "test book taken",
				"description":              "test book description",
				"genres":                   "Fantasy",
				"language":                 "English",
				"release_schedule_day":     "Sunday",
				"release_schedule_chapter": "2",
				"chapter_title":            "test chapter title",
				"chapter_content":          "test chapter content",
			},
			expectedCode: http.StatusConflict,
		},
		{
			name:        "genre not found",
			cookieName:  "access_token",
			cookieValue: token,
			req: map[string]string{
				"name":                     "test book",
				"description":              "test book description",
				"genres":                   "non-existent genre",
				"language":                 "English",
				"release_schedule_day":     "Sunday",
				"release_schedule_chapter": "2",
				"chapter_title":            "test chapter title",
				"chapter_content":          "test chapter content",
			},
			expectedCode: http.StatusNotFound,
		},
		{
			name:        "upload book",
			cookieName:  "access_token",
			cookieValue: token,
			req: map[string]string{
				"name":                     "test book",
				"description":              "test book description",
				"genres":                   "Action",
				"language":                 "English",
				"release_schedule_day":     "Sunday",
				"release_schedule_chapter": "2",
				"chapter_title":            "test chapter title",
				"chapter_content":          "test chapter content",
			},
			expectedCode: http.StatusCreated,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			for key, val := range tc.req {
				writer.WriteField(key, val)
			}
			writer.Close()

			r := httptest.NewRequest(http.MethodPost, "/api/v1/books", body)
			r.Header.Set("Content-Type", writer.FormDataContentType())
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

func TestHandleGetBooks(t *testing.T) {
	db := connectTestDb(t)
	id := createAndCleanUpUser(t, db)
	createBook(t, id, db)

	tests := []struct {
		name         string
		path         string
		expectedCode int
	}{
		{
			name:         "offset isn't a valid number",
			path:         "/api/v1/books?offset=invalid",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "limit isn't a valid number",
			path:         "/api/v1/books?offset=1&limit=invalid",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "books under genre",
			path:         "/api/v1/books?genre=Action&offset=1&limit=1",
			expectedCode: http.StatusOK,
		},
		{
			name:         "books under language",
			path:         "/api/v1/books?language=English&offset=1&limit=1",
			expectedCode: http.StatusOK,
		},
		{
			name:         "books under genre and language",
			path:         "/api/v1/books?genre=Action&language=English&offset=1&limit=1",
			expectedCode: http.StatusOK,
		},
		{
			name:         "all books",
			path:         "/api/v1/books?offset=1&limit=1",
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()

			svr := newServer(nil, db, nil, nil)
			svr.router.ServeHTTP(rr, r)

			if rr.Code != tc.expectedCode {
				t.Fatalf("expected %d, got %d", tc.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleGetBooksStats(t *testing.T) {
	db := connectTestDb(t)
	id := createAndCleanUpUser(t, db)
	token, err := createJWTToken(id)
	if err != nil {
		t.Fatal(err.Error())
	}

	createBook(t, id, db)

	tests := []struct {
		name         string
		cookieName   string
		cookieValue  string
		path         string
		expectedCode int
	}{
		{
			name:         "no access token cookie",
			path:         "/api/v1/books/stats",
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "invalid/malformed token",
			cookieName:   "access_token",
			cookieValue:  "invalid token",
			path:         "/api/v1/books/stats",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "offset isn't a valid number",
			cookieName:   "access_token",
			cookieValue:  token,
			path:         "/api/v1/books/stats?offset=invalid",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "limit isn't a valid number",
			cookieName:   "access_token",
			cookieValue:  token,
			path:         "/api/v1/books/stats?offset=1&limit=invalid",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "get books",
			cookieName:   "access_token",
			cookieValue:  token,
			path:         "/api/v1/books/stats?offset=1&limit=1",
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tc.path, nil)
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

func TestHandleGetRecentlyReadBooks(t *testing.T) {
	db := connectTestDb(t)
	id := createAndCleanUpUser(t, db)
	token, err := createJWTToken(id)
	if err != nil {
		t.Fatal(err.Error())
	}

	createBook(t, id, db)

	tests := []struct {
		name         string
		cookieName   string
		cookieValue  string
		path         string
		expectedCode int
	}{
		{
			name:         "no access token cookie",
			path:         "/api/v1/books/recently-read",
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "invalid/malformed token",
			cookieName:   "access_token",
			cookieValue:  "invalid token",
			path:         "/api/v1/books/recently-read",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "offset isn't a valid number",
			cookieName:   "access_token",
			cookieValue:  token,
			path:         "/api/v1/books/recently-read?offset=invalid",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "limit isn't a valid number",
			cookieName:   "access_token",
			cookieValue:  token,
			path:         "/api/v1/books/recently-read?offset=1&limit=invalid",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "get recently read books",
			cookieName:   "access_token",
			cookieValue:  token,
			path:         "/api/v1/books/recently-read?offset=1&limit=1",
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tc.path, nil)
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

func TestHandleGetRecentlyUploadedBooks(t *testing.T) {
	db := connectTestDb(t)
	id := createAndCleanUpUser(t, db)
	token, err := createJWTToken(id)
	if err != nil {
		t.Fatal(err.Error())
	}

	createBook(t, id, db)

	tests := []struct {
		name         string
		cookieName   string
		cookieValue  string
		admin        bool
		path         string
		expectedCode int
	}{
		{
			name:         "no access token cookie",
			path:         "/api/v1/books/recently-uploaded",
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "invalid/malformed token",
			cookieName:   "access_token",
			cookieValue:  "invalid token",
			path:         "/api/v1/books/recently-uploaded",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "role is not admin",
			cookieName:   "access_token",
			cookieValue:  token,
			path:         "/api/v1/books/recently-uploaded",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "offset isn't a valid number",
			cookieName:   "access_token",
			cookieValue:  token,
			admin:        true,
			path:         "/api/v1/books/recently-uploaded?offset=invalid",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "limit isn't a valid number",
			cookieName:   "access_token",
			cookieValue:  token,
			admin:        true,
			path:         "/api/v1/books/recently-uploaded?offset=1&limit=invalid",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "get recently read books",
			cookieName:   "access_token",
			cookieValue:  token,
			admin:        true,
			path:         "/api/v1/books/recently-uploaded?offset=1&limit=1",
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.admin == true {
				query :=
					`
						UPDATE users SET roles = ARRAY['ADMIN']::role_type[] WHERE display_name = 'test_display';
					`
				if _, err := db.ExecContext(context.Background(), query); err != nil {
					t.Fatalf("error updating users, %v", err)
				}
			}
			r := httptest.NewRequest(http.MethodGet, tc.path, nil)
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

func TestHandleGetBook(t *testing.T) {
	db := connectTestDb(t)
	userID := createAndCleanUpUser(t, db)

	tests := []struct {
		name         string
		bookID       string
		expectedCode int
	}{
		{
			name:         "book not found",
			bookID:       uuid.NewString(),
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "get book",
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svr := newServer(nil, db, nil, nil)

			if tc.bookID == "" {
				bookID, err := svr.uploadBook(context.Background(), &book{name: "test-book", description: "test-book description", authorID: userID, genres: []string{"Action"}, draftChapter: draftChapter{Title: "draft chapter title", Content: "draft chapter content"}, language: "English", releaseSchedule: []releaseSchedule{{Day: "Monday", Chapters: 1}}})
				if err != nil {
					t.Fatal(err)
				}

				if err := svr.approveBook(context.Background(), bookID, true); err != nil {
					t.Fatal(err)
				}
				tc.bookID = bookID
			}

			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/books/%v", tc.bookID), nil)
			rr := httptest.NewRecorder()

			svr.router.ServeHTTP(rr, r)

			if rr.Code != tc.expectedCode {
				t.Fatalf("expected %d, got %d", tc.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleDeleteBook(t *testing.T) {
	db := connectTestDb(t)
	userID := createAndCleanUpUser(t, db)
	token, err := createJWTToken(userID)
	if err != nil {
		t.Fatal(err.Error())
	}

	bookID := createBook(t, userID, db)

	tests := []struct {
		name         string
		cookieName   string
		cookieValue  string
		bookID       string
		expectedCode int
	}{
		{
			name:         "no access token cookie",
			bookID:       uuid.NewString(),
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "invalid/malformed token",
			cookieName:   "access_token",
			cookieValue:  "invalid token",
			bookID:       uuid.NewString(),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "book not found/book does not belong to user",
			cookieName:   "access_token",
			cookieValue:  token,
			bookID:       uuid.NewString(),
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "delete book",
			cookieName:   "access_token",
			cookieValue:  token,
			bookID:       bookID,
			expectedCode: http.StatusNoContent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/books/%v", tc.bookID), nil)
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

func TestHandleEditBook(t *testing.T) {
	db := connectTestDb(t)
	userID := createAndCleanUpUser(t, db)
	token, err := createJWTToken(userID)
	if err != nil {
		t.Fatal(err.Error())
	}

	tests := []struct {
		name         string
		cookieName   string
		cookieValue  string
		bookID       string
		req          map[string]string
		expectedCode int
	}{
		{
			name:         "no access token cookie",
			bookID:       uuid.NewString(),
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "invalid/malformed token",
			cookieName:   "access_token",
			cookieValue:  "invalid token",
			bookID:       uuid.NewString(),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "no field passed to update",
			cookieName:   "access_token",
			cookieValue:  token,
			bookID:       uuid.NewString(),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:        "chapter length and days not the same",
			cookieName:  "access_token",
			cookieValue: token,
			bookID:      uuid.NewString(),
			req: map[string]string{
				"release_schedule_day": "Monday",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:        "book not found",
			cookieName:  "access_token",
			cookieValue: token,
			bookID:      uuid.NewString(),
			req: map[string]string{
				"release_schedule_day":     "Monday",
				"release_schedule_chapter": "1",
			},
			expectedCode: http.StatusNotFound,
		},
		{
			name:        "edit book",
			cookieName:  "access_token",
			cookieValue: token,
			req: map[string]string{
				"release_schedule_day":     "Monday",
				"release_schedule_chapter": "1",
			},
			expectedCode: http.StatusNoContent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			if len(tc.req) > 0 {
				for key, val := range tc.req {
					writer.WriteField(key, val)
				}
				writer.Close()
			}

			svr := newServer(nil, db, nil, nil)

			if tc.bookID == "" {
				bookID, err := svr.uploadBook(context.Background(), &book{name: "test-book", description: "test-book description", authorID: userID, genres: []string{"Action"}, draftChapter: draftChapter{Title: "draft chapter title", Content: "draft chapter content"}, language: "English", releaseSchedule: []releaseSchedule{{Day: "Monday", Chapters: 1}}})
				if err != nil {
					t.Fatal(err)
				}
				tc.bookID = bookID
			}

			r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/books/%v", tc.bookID), body)
			r.Header.Set("Content-Type", writer.FormDataContentType())
			r.AddCookie(&http.Cookie{Name: tc.cookieName, Value: tc.cookieValue})
			rr := httptest.NewRecorder()

			svr.router.ServeHTTP(rr, r)

			if rr.Code != tc.expectedCode {
				t.Fatalf("expected %d, got %d", tc.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleApproveBook(t *testing.T) {
	db := connectTestDb(t)
	userID := createAndCleanUpUser(t, db)
	token, err := createJWTToken(userID)
	if err != nil {
		t.Fatal(err.Error())
	}

	tests := []struct {
		name         string
		cookieName   string
		cookieValue  string
		bookID       string
		admin        bool
		body         any
		expectedCode int
	}{
		{
			name:         "no access token cookie",
			bookID:       uuid.NewString(),
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "invalid/malformed token",
			cookieName:   "access_token",
			cookieValue:  "invalid token",
			bookID:       uuid.NewString(),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "role is not admin",
			cookieName:   "access_token",
			cookieValue:  token,
			bookID:       uuid.NewString(),
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "validation error",
			cookieName:   "access_token",
			cookieValue:  token,
			bookID:       uuid.NewString(),
			admin:        true,
			body:         struct{ name string }{name: "invalid structure"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:        "book not found",
			cookieName:  "access_token",
			cookieValue: token,
			bookID:      uuid.NewString(),
			admin:       true,
			body: struct {
				Approve bool `json:"approve"`
			}{Approve: true},
			expectedCode: http.StatusNotFound,
		},
		{
			name:        "approve book",
			cookieName:  "access_token",
			cookieValue: token,
			bookID:      createBook(t, userID, db),
			admin:       true,
			body: struct {
				Approve bool `json:"approve"`
			}{Approve: true},
			expectedCode: http.StatusNoContent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.admin == true {
				query :=
					`
						UPDATE users SET roles = ARRAY['ADMIN']::role_type[] WHERE display_name = 'test_display';
					`
				if _, err := db.ExecContext(context.Background(), query); err != nil {
					t.Fatalf("error updating users, %v", err)
				}
			}

			body, _ := json.Marshal(tc.body)
			r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/books/%v/approve", tc.bookID), bytes.NewReader(body))
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

func TestHandleCompleteBook(t *testing.T) {
	db := connectTestDb(t)
	userID := createAndCleanUpUser(t, db)
	token, err := createJWTToken(userID)
	if err != nil {
		t.Fatal(err.Error())
	}

	tests := []struct {
		name         string
		cookieName   string
		cookieValue  string
		bookID       string
		body         any
		expectedCode int
	}{
		{
			name:         "no access token cookie",
			bookID:       uuid.NewString(),
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "invalid/malformed token",
			cookieName:   "access_token",
			cookieValue:  "invalid token",
			bookID:       uuid.NewString(),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "validation error",
			cookieName:   "access_token",
			cookieValue:  token,
			bookID:       uuid.NewString(),
			body:         struct{ name string }{name: "invalid structure"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:        "book not found",
			cookieName:  "access_token",
			cookieValue: token,
			bookID:      uuid.NewString(),
			body: struct {
				Complete bool `json:"complete"`
			}{Complete: true},
			expectedCode: http.StatusNotFound,
		},
		{
			name:        "mark book as complete",
			cookieName:  "access_token",
			cookieValue: token,
			bookID:      createBook(t, userID, db),
			body: struct {
				Complete bool `json:"complete"`
			}{Complete: true},
			expectedCode: http.StatusNoContent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.body)
			r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/books/%v/complete", tc.bookID), bytes.NewReader(body))
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
