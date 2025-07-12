package routes

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/markbates/goth/gothic"
)

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriterWrapper(w http.ResponseWriter) *responseWriterWrapper {
	return &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}
}

func (w *responseWriterWrapper) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (s *Server) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := newResponseWriterWrapper(w)

		next.ServeHTTP(ww, r)

		duration := time.Since(start)

		s.Server.Logger.Info(
			"request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.String()),
			slog.Int("status", ww.statusCode),
			slog.String("duration", duration.String()),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
		)
	})
}

const (
	PermissionUploadBooks            = "books:upload"
	PermissionGetCreatorBooks        = "creator:books"
	PermissionGetBookStat            = "books:stats"
	PermissionMarkComplete           = "mark:complete"
	PermissionUploadChapters         = "upload:chapters"
	PermissionDeleteChapters         = "delete:chapters"
	PermissionApproveOrDenyBooks     = "books:approve"
	PermissionBanUsers               = "ban:users"
	PermissionGetRecentReads         = "recent:reads"
	PermissionGetNewlyUpdated        = "newly:updated"
	PermissionGetRecommendations     = "get:recommendations"
	PermissionGetBooks               = "get:books"
	PermissionAddToLibrary           = "add:library:books"
	PermissionGetAllBooksFromLibrary = "get:library:books"
	PermissionRemoveBookFromLibrary  = "remove:library:book"
	PermissionBuyCoins               = "coins"
	PermissionCommentOnBooks         = "books:comment"
	PermissionGetAllCommentsOnBook   = "get:book:comment"
	PermissionDeleteBook             = "books:delete"
	PermissionEditBook               = "books:edit"
)

func (s *Server) CheckPermission(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), "provider", "google"))

			session, _ := gothic.Store.Get(r, "app_session")

			id, ok := session.Values["user_id"].(string)

			if !ok || id == "" {
				s.Logger.Warn("no user in session", "status", "permission denied")
				respondWithError(w, http.StatusNotFound, fmt.Errorf("no user in session"))
				return
			}

			db_user, err := s.Store.GetUserById(r.Context(), id)

			if err != nil {
				s.Logger.Warn(err.Error(), "service", "middleware")
				respondWithError(w, http.StatusNotFound, err)
				return
			}

			hasPermission := false

			for _, perm := range permissions {
				for _, roles_perm := range db_user.Privileges {
					if roles_perm == perm {
						hasPermission = true
						break
					}
				}
			}

			if !hasPermission {
				s.Server.Logger.Warn("role does not have permission to access this route", "status", "permission denied")
				respondWithError(w, http.StatusForbidden, fmt.Errorf("role does not have permission to access this route"))
				return
			}

			user := struct {
				id   uuid.UUID
				role string
			}{
				id:   db_user.Id,
				role: db_user.Role,
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "user", user)))
		})
	}
}
