package main

import (
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

	s.router.Post("/api/v1/books", nil)
	s.router.Get("/api/v1/books", nil)
	s.router.Get("/api/v1/books/stats", nil)
	s.router.Get("/api/v1/books/recents", nil)

	s.router.Get("/api/v1/books/{bookId}", nil)
	s.router.Delete("/api/v1/books/{bookId}", nil)
	s.router.Patch("/api/v1/books/{bookId}", nil)
	s.router.Patch("/api/v1/books/{bookId}/approval", nil)
	s.router.Patch("/api/v1/books/{bookId}/complete", nil)

	s.router.Post("/api/v1/books/{bookId}/chapters", nil)
	s.router.Get("/api/v1/books/{bookId}/chapters/{chapterId}", nil)
	s.router.Delete("/api/v1/books/{bookId}/chapters/{chapterId}", nil)
	s.router.Get("/api/v1/books/{bookId}/chapters/{chapterId}/pages/{pageNumber}", nil)

	s.router.Post("/api/v1/books/{bookId}/comments", nil)
	s.router.Get("/api/v1/books/{bookId}/comments", nil)
	s.router.Get("/api/v1/books/{bookId}/comments/{commentId}", nil)
	s.router.Delete("/api/v1/books/{bookId}/comments/{commentId}", nil)
	s.router.Patch("/api/v1/books/{bookId}/comments/{commentId}", nil)

	s.router.Patch("/api/v1/books/{bookId}/subscriptions", nil)

	s.router.Get("/api/v1/users/me", s.handleGetProfile)

	s.router.Get("/api/v1/library", nil)
	s.router.Put("/api/v1/library/books/{bookId}", nil)
	s.router.Delete("/api/v1/library/books/{bookId}", nil)

	s.router.Post("/api/v1/coins", nil)

	s.router.Post("/webhook", nil)
	s.router.Patch("/users/{userId}/ban", nil)
}
