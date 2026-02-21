package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type mc struct{}

func (*mc) PublishWithContext(_ context.Context, _, _ string, _, _ bool, _ amqp.Publishing) error {
	return nil
}

func TestHandleUploadChapter(t *testing.T) {
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
		mockChannel  channel
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
				Title     string `json:"title"`
				ChapterNo int    `json:"chapterNo"`
				Content   string `json:"content"`
			}{
				Title:     "test chapter",
				ChapterNo: 1,
				Content:   "test chapter content",
			},
			expectedCode: http.StatusNotFound,
		},
		{
			name:        "upload chapter",
			cookieName:  "access_token",
			cookieValue: token,
			bookID:      createBook(t, userID, db),
			body: struct {
				Title     string `json:"title"`
				ChapterNo int    `json:"chapterNo"`
				Content   string `json:"content"`
			}{
				Title:     "test chapter",
				ChapterNo: 1,
				Content:   "test chapter content",
			},
			mockChannel:  &mc{},
			expectedCode: http.StatusCreated,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.body)
			r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/books/%v/chapters", tc.bookID), bytes.NewReader(body))
			r.AddCookie(&http.Cookie{Name: tc.cookieName, Value: tc.cookieValue})
			rr := httptest.NewRecorder()

			svr := newServer(nil, db, nil, tc.mockChannel)
			svr.router.ServeHTTP(rr, r)

			if rr.Code != tc.expectedCode {
				t.Fatalf("expected %d, got %d", tc.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleGetChapter(t *testing.T) {
	db := connectTestDb(t)
	userID := createAndCleanUpUser(t, db)
	token, err := createJWTToken(userID)
	if err != nil {
		t.Fatal(err.Error())
	}

	svr := newServer(nil, db, nil, nil)
	chapterID, err := svr.uploadChapter(context.Background(), userID, &chapter{title: "test chapter", chapterNo: 1, content: "test chapter content", bookID: createBook(t, userID, db)})
	if err != nil {
		t.Fatal(err.Error())
	}

	tests := []struct {
		name         string
		cookieName   string
		cookieValue  string
		chapterID    string
		mockChannel  channel
		expectedCode int
	}{
		{
			name:         "no access token cookie",
			chapterID:    uuid.NewString(),
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "invalid/malformed token",
			cookieName:   "access_token",
			cookieValue:  "invalid token",
			chapterID:    uuid.NewString(),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "chapter not found",
			cookieName:   "access_token",
			cookieValue:  token,
			chapterID:    uuid.NewString(),
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "get chapter",
			cookieName:   "access_token",
			cookieValue:  token,
			chapterID:    chapterID,
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/books/chapters/%v", tc.chapterID), nil)
			r.AddCookie(&http.Cookie{Name: tc.cookieName, Value: tc.cookieValue})
			rr := httptest.NewRecorder()

			svr.router.ServeHTTP(rr, r)

			if rr.Code != tc.expectedCode {
				t.Fatalf("expected %d, got %d", tc.expectedCode, rr.Code)
			}
		})
	}
}

func TestHandleDeleteChapter(t *testing.T) {
	db := connectTestDb(t)
	userID := createAndCleanUpUser(t, db)
	token, err := createJWTToken(userID)
	if err != nil {
		t.Fatal(err.Error())
	}

	svr := newServer(nil, db, nil, nil)
	bookID := createBook(t, userID, db)
	chapterID, err := svr.uploadChapter(context.Background(), userID, &chapter{title: "test chapter", chapterNo: 1, content: "test chapter content", bookID: bookID})
	if err != nil {
		t.Fatal(err.Error())
	}

	tests := []struct {
		name         string
		cookieName   string
		cookieValue  string
		bookID       string
		chapterID    string
		mockChannel  channel
		expectedCode int
	}{
		{
			name:         "no access token cookie",
			bookID:       uuid.NewString(),
			chapterID:    uuid.NewString(),
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "invalid/malformed token",
			cookieName:   "access_token",
			cookieValue:  "invalid token",
			bookID:       uuid.NewString(),
			chapterID:    uuid.NewString(),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "book not found",
			cookieName:   "access_token",
			cookieValue:  token,
			bookID:       uuid.NewString(),
			chapterID:    uuid.NewString(),
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "chapter not found",
			cookieName:   "access_token",
			cookieValue:  token,
			bookID:       bookID,
			chapterID:    uuid.NewString(),
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "delete book",
			cookieName:   "access_token",
			cookieValue:  token,
			bookID:       bookID,
			chapterID:    chapterID,
			expectedCode: http.StatusNoContent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/books/%v/chapters/%v", tc.bookID, tc.chapterID), nil)
			r.AddCookie(&http.Cookie{Name: tc.cookieName, Value: tc.cookieValue})
			rr := httptest.NewRecorder()

			svr.router.ServeHTTP(rr, r)

			if rr.Code != tc.expectedCode {
				t.Fatalf("expected %d, got %d", tc.expectedCode, rr.Code)
			}
		})
	}
}
