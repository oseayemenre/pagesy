package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/oseayemenre/pagesy/cmd"
)

type Server struct {
	*cmd.Server
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) RegisterRoutes() {
	s.Server.Router.Route("/api/v1", func(r chi.Router) {
		r.Route("/books", func(r chi.Router) {
			r.Post("/", nil)
			r.Get("/", nil)
			r.Get("/{bookId}", nil)
			r.Put("/{bookId}", nil)
			r.Patch("/{bookId}/approval", nil)
			r.Patch("/{bookId}/complete", nil)
			r.Post("/{bookId}/chapters", nil)
			r.Delete("/{bookId}/chapters/{chapterId}", nil)
			r.Get("/{bookId}/chapters/{chapterId}/pages/{pageNumber}", nil)
			r.Get("/recents", nil)
			r.Get("/new", nil)
			r.Get("/recommended", nil)
			r.Post("/{bookId}/comments", nil)
			r.Get("/{bookId}/comments", nil)
		})

		r.Route("/library", func(r chi.Router) {
			r.Get("/", nil)
			r.Put("/books/{bookId}", nil)
			r.Get("/books/{bookId}", nil)
			r.Delete("/books/{bookId}", nil)
		})

		r.Post("/coins", nil)
		r.Patch("/users/{userId}/ban", nil)
	})
}
