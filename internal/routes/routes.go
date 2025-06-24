package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/oseayemenre/pagesy/internal/shared"
)

const (
	PermissionUploadBooks            = "upload:books"
	PermissionGetCreatorBooks        = "get:creator:books"
	PermissionGetBookStat            = "get:stats:book"
	PermissionMarkComplete           = "mark:complete"
	PermissionUploadChapters         = "upload:chapters"
	PermissionDeleteChapters         = "delete:chapters"
	PermissionApproveOrDenyBooks     = "approve:books"
	PermissionBanUsers               = "ban:users"
	PermissionGetRecentReads         = "get:recent:reads"
	PermissionGetNewlyUpdated        = "get:newly:updated"
	PermissionGetRecommendations     = "get:recommendations"
	PermissionGetAllBooks            = "get:books"
	PermissionGetSpecificBook        = "get:book"
	PermissionAddToLibrary           = "add:library"
	PermissionGetAllBooksFromLibrary = "get:library:books"
	PermissionRemoveBookFromLibrary  = "remove:library:book"
	PermissionBuyCoins               = "coins"
	PermissionCommentOnBooks         = "book:comment"
	PermissionGetAllCommentsOnBook   = "book:comments"
	PermissionDeleteBook             = "book:delete"
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
		r.Route("/books", func(r chi.Router) {
			r.With(s.CheckPermission(PermissionUploadBooks)).Post("/", s.HandleUploadBooks)
			r.With(s.CheckPermission(PermissionGetCreatorBooks)).Get("/stats", s.HandleGetBooksStats)
			r.With(s.CheckPermission(PermissionGetCreatorBooks, PermissionGetAllBooks)).Get("/", s.HandleGetBooks)
			r.With(s.CheckPermission(PermissionGetSpecificBook)).Get("/{bookId}", s.HandleGetBook)
			r.With(s.CheckPermission(PermissionDeleteBook)).Delete("/{bookId}", s.HandleDeleteBook)
			r.Put("/{bookId}", nil)
			r.Patch("/{bookId}/approval", nil)
			r.Patch("/{bookId}/complete", nil)
			r.Post("/{bookId}/chapters", nil)
			r.Get("/{bookId}/chapters/{chapterId}", nil)
			r.Delete("/{bookId}/chapters/{chapterId}", nil)
			r.Get("/{bookId}/chapters/{chapterId}/pages/{pageNumber}", nil)
			r.Get("/recents", nil)
			r.Get("/new", nil)
			r.Get("/recommended", nil)
			r.Post("/{bookId}/comments", nil)
			r.Get("/{bookId}/comments", nil)
			r.Post("/{bookId}/comments/{commentId}", nil)
			r.Delete("/{bookId}/comments/{commentId}", nil)
			r.Patch("/{bookId}/comments/{commentId}", nil)
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
