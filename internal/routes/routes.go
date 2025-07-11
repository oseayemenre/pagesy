package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/oseayemenre/pagesy/internal/shared"
)

type Server struct {
	*shared.Server
}

func NewServer(server *shared.Server) *Server {
	return &Server{Server: server}
}

func (s *Server) RegisterRoutes() {
	s.Server.Router.Route("/api/v1", func(r chi.Router) {
		r.Use(s.LoggingMiddleware)
		r.Route("/auth", func(r chi.Router) {
			r.Route("/google", func(r chi.Router) {
				r.Get("/", s.HandleGoogleSignIn)
				r.Get("/callback", s.HandleGoogleSignInCallback)
			})
		})

		r.Route("/books", func(r chi.Router) {
			r.With(s.CheckPermission(PermissionUploadBooks)).Post("/", s.HandleUploadBooks)
			r.With(s.CheckPermission(PermissionGetCreatorBooks)).Get("/stats", s.HandleGetBooksStats)
			r.With(s.CheckPermission(PermissionGetCreatorBooks, PermissionGetBooks)).Get("/", s.HandleGetBooks)
			r.With(s.CheckPermission(PermissionGetRecentReads)).Get("/recents", s.HandleGetRecentReads)
			r.With(s.CheckPermission(PermissionGetNewlyUpdated)).Get("/new", s.HandleGetNewlyUpdated)
			r.Get("/recommended", nil)

			r.Route("/{bookId}", func(r chi.Router) {
				r.With(s.CheckPermission(PermissionGetBooks)).Get("/", s.HandleGetBook)
				r.With(s.CheckPermission(PermissionDeleteBook)).Delete("/", s.HandleDeleteBook)
				r.With(s.CheckPermission(PermissionEditBook)).Patch("/", s.HandleEditBook)
				r.With(s.CheckPermission(PermissionApproveOrDenyBooks)).Patch("/approval", s.HandleApproveBook)
				r.With(s.CheckPermission(PermissionMarkComplete)).Patch("/complete", s.HandleMarkBookAsComplete)

				r.Route("/chapters", func(r chi.Router) {
					r.Post("/", nil)
					r.Get("/{chapterId}", nil)
					r.Delete("/{chapterId}", nil)
					r.Get("/{chapterId}/pages/{pageNumber}", nil)
				})

				r.Route("/comments", func(r chi.Router) {
					r.Post("/", nil)
					r.Get("/", nil)
					r.Post("/{commentId}", nil)
					r.Delete("/{commentId}", nil)
					r.Patch("/{commentId}", nil)
				})
			})
		})

		r.Route("/library", func(r chi.Router) {
			r.Get("/", nil)
			r.Put("/books/{bookId}", nil)
			r.Delete("/books/{bookId}", nil)
		})

		r.Post("/coins", nil)
		r.Patch("/users/{userId}/ban", nil)
	})
}
