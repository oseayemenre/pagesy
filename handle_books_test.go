package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
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

			test_svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upgrader := websocket.Upgrader{
					ReadBufferSize:  1024,
					WriteBufferSize: 1024,
					CheckOrigin: func(r *http.Request) bool {
						return true
					},
				}

				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					t.Fatal("error upgrading websocket connection")
				}

				for {
					if _, _, err := conn.ReadMessage(); err != nil {
						t.Fatal("error reading from client")
					}
				}
			}))

			os.Setenv("WS_HOST", strings.TrimPrefix(test_svr.URL, "http://"))
			svr := newServer(nil, db, nil)
			svr.router.ServeHTTP(rr, r)

			if rr.Code != tc.expectedCode {
				t.Fatalf("expected %d, got %d", tc.expectedCode, rr.Code)
			}
		})
	}
}
