package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
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

	createAndCleanUpBook(t, id, db)

	tests := []struct {
		name         string
		cookie_name  string
		cookie_value string
		req          map[string]string
		expectedCode int
	}{
		{
			name:         "no access token cookie",
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "invalid/malformed token",
			cookie_name:  "access_token",
			cookie_value: "invalid token",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "chapter length and days not the same",
			cookie_name:  "access_token",
			cookie_value: token,
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
			name:         "chapter count is not convertible to int",
			cookie_name:  "access_token",
			cookie_value: token,
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
			name:         "validation error",
			cookie_name:  "access_token",
			cookie_value: token,
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
			name:         "book name taken",
			cookie_name:  "access_token",
			cookie_value: token,
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
			name:         "genre not found",
			cookie_name:  "access_token",
			cookie_value: token,
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
			name:         "upload book",
			cookie_name:  "access_token",
			cookie_value: token,
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
			r.AddCookie(&http.Cookie{Name: tc.cookie_name, Value: tc.cookie_value})
			rr := httptest.NewRecorder()

			svr := newServer(nil, db, nil)
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
	createAndCleanUpBook(t, id, db)

	tests := []struct {
		name         string
		path         string
		expectedCode int
	}{
		{
			name:         "offset isn't a valid number",
			path:         "/api/v1/books?offset=invalid",
			expectedCode: 400,
		},
		{
			name:         "limit isn't a valid number",
			path:         "/api/v1/books?offset=1&limit=invalid",
			expectedCode: 400,
		},
		{
			name:         "books under genre",
			path:         "/api/v1/books?genre=Action&offset=1&limit=1",
			expectedCode: 200,
		},
		{
			name:         "books under language",
			path:         "/api/v1/books?language=English&offset=1&limit=1",
			expectedCode: 200,
		},
		{
			name:         "books under genre and language",
			path:         "/api/v1/books?genre=Action&language=English&offset=1&limit=1",
			expectedCode: 200,
		},
		{
			name:         "all books",
			path:         "/api/v1/books?offset=1&limit=1",
			expectedCode: 200,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()

			svr := newServer(nil, db, nil)
			svr.router.ServeHTTP(rr, r)

			if rr.Code != tc.expectedCode {
				t.Fatalf("expected %d, got %d", tc.expectedCode, rr.Code)
			}
		})
	}
}
