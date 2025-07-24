package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/oseayemenre/pagesy/internal/jwt"
	"github.com/oseayemenre/pagesy/internal/models"
	"github.com/oseayemenre/pagesy/internal/store"
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

func (a *Api) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := newResponseWriterWrapper(w)

		next.ServeHTTP(ww, r)

		duration := time.Since(start)

		a.logger.Info(
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

func (a *Api) CheckPermission(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := r.Cookie("access_token")

			if err != nil {
				a.logger.Warn("access token cookie not found", "status", "permission denied")
				respondWithError(w, http.StatusNotFound, fmt.Errorf("access token cookie not found"))
				return
			}

			id, err := jwt.DecodeJWTToken(token.Value, a.config.Jwt_secret)

			if err != nil {
				a.logger.Warn(err.Error(), "status", "permission denied")
				respondWithError(w, http.StatusUnauthorized, err)
				return
			}

			db_user, err := a.store.GetUserById(r.Context(), id)

			if err != nil {
				if err == store.ErrUserNotFound {
					a.logger.Error(err.Error(), "service", "middleware")
					respondWithError(w, http.StatusNotFound, err)
					return
				}
				a.logger.Error(err.Error(), "service", "middleware")
				respondWithError(w, http.StatusInternalServerError, err)
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
				a.logger.Warn("role does not have permission to access this route", "status", "permission denied")
				respondWithError(w, http.StatusForbidden, fmt.Errorf("role does not have permission to access this route"))
				return
			}

			uuid_id, err := uuid.Parse(id)

			if err != nil {
				a.logger.Error("id is not a valid uuid", "service", "middleware")
				respondWithError(w, http.StatusBadRequest, fmt.Errorf("id is not a valid uuid"))
				return
			}

			user := &models.User{
				Id:   uuid_id,
				Role: db_user.Role,
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "user", user)))
		})
	}
}

func (a *Api) RedirectIfCookieExistsAndIsValid(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := r.Cookie("access_token")

		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		_, err = jwt.DecodeJWTToken(token.Value, a.config.Jwt_secret)

		if err != nil {
			a.logger.Warn(err.Error(), "status", "permission denied")
			respondWithError(w, http.StatusUnauthorized, err)
			return
		}

		http.Redirect(w, r, "/healthz", http.StatusFound) //TODO: change when there's a frontend
	})
}
