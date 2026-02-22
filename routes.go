package main

import (
	"context"
	"fmt"
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"
)

func (s *server) routes() {
	s.router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("server healthy\n"))
	})

	s.router.Get("/swagger/*", httpSwagger.WrapHandler)

	s.router.Get("/api/v1/auth/google", s.handleAuthGoogle)
	s.router.Get("/api/v1/auth/google/callback", s.handleAuthGoogleCallback)

	s.router.Post("/api/v1/auth/onboarding", s.handleAuthOnboarding)
	s.router.Post("/api/v1/auth/register", s.handleAuthRegister)
	s.router.Post("/api/v1/auth/login", s.handleAuthLogin)
	s.router.Post("/api/v1/auth/logout", s.handleAuthLogout)
	s.router.Post("/api/v1/auth/refresh-token", s.handleAuthRefreshToken)

	s.router.Post("/api/v1/books", authenticatedUser(s.handleUploadBook))
	s.router.Get("/api/v1/books", s.handleGetBooks)
	s.router.Get("/api/v1/books/stats", authenticatedUser(s.handleGetBooksStats))
	s.router.Get("/api/v1/books/recently-read", authenticatedUser(s.handleGetRecentlyReadBooks))
	s.router.Get("/api/v1/books/recently-uploaded", authenticatedUser(s.handleGetRecentlyUploadedBooks))

	s.router.Get("/api/v1/books/{bookID}", s.handleGetBook)
	s.router.Delete("/api/v1/books/{bookID}", authenticatedUser(s.handleDeleteBook))
	s.router.Patch("/api/v1/books/{bookID}", authenticatedUser(s.handleEditBook))
	s.router.Patch("/api/v1/books/{bookID}/approve", authenticatedUser(s.handleApproveBook))
	s.router.Patch("/api/v1/books/{bookID}/complete", authenticatedUser(s.handleCompleteBook))

	s.router.Post("/api/v1/books/{bookID}/chapters", authenticatedUser(s.handleUploadChapter))
	s.router.Get("/api/v1/books/chapters/{chapterID}", authenticatedUser(s.handleGetChapter))
	s.router.Delete("/api/v1/books/{bookID}/chapters/{chapterID}", authenticatedUser(s.handleDeleteChapter))
	s.router.Patch("/api/v1/books/{bookID}/chapters/{chapterID}", authenticatedUser(s.handleEditChapter))

	s.router.Post("/api/v1/books/{bookID}/comments", nil)
	s.router.Get("/api/v1/books/{bookID}/comments", nil)
	s.router.Get("/api/v1/books/{bookID}/comments/{commentID}", nil)
	s.router.Delete("/api/v1/books/{bookID}/comments/{commentID}", nil)
	s.router.Patch("/api/v1/books/{bookID}/comments/{commentID}", nil)

	s.router.Post("/api/v1/users/{userID}/follow", authenticatedUser(s.handleFollowUser))
	s.router.Delete("/api/v1/users/{userID}/unfollow", authenticatedUser(s.handleUnfollowUser))
	s.router.Get("/api/v1/users/{userID}/followers", authenticatedUser(s.handleGetUserFollowers))
	s.router.Get("/api/v1/users/{userID}/following", nil)
	s.router.Get("/api/v1/users/me", authenticatedUser(s.handleGetProfile))
	s.router.Get("/api/v1/users/me/following", nil)
	s.router.Get("/api/v1/users/me/followers", nil)

	s.router.Post("/api/v1/books/{bookID}/ratings", nil)
	s.router.Patch("/api/v1/books/{bookID}/subscriptions", nil)

	s.router.Get("/api/v1/library", nil)
	s.router.Put("/api/v1/library/books/{bookID}", nil)
	s.router.Delete("/api/v1/library/books/{bookID}", nil)

	s.router.Post("/api/v1/coins", nil)

	s.router.HandleFunc("/api/v1/ws", authenticatedUser(s.handleWS))
	s.router.Post("/webhook", nil)
	s.router.Patch("/users/{userID}/ban", nil)
	s.router.Get("/users/{userID}/notifications", nil)
}

func authenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("access_token")
		if err != nil {
			encode(w, http.StatusNotFound, &errorResponse{Error: fmt.Sprintf("error retrieving access token, %v", err)})
			return
		}

		id, err := decodeJWTToken(cookie.Value)
		if err != nil {
			encode(w, http.StatusBadRequest, &errorResponse{Error: err.Error()})
			return
		}

		next(w, r.WithContext(context.WithValue(r.Context(), "user", id)))
	}
}
