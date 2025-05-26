package routes

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	// "strings"
	"time"
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
	w.WriteHeader(code)
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

func (s *Server) CheckPermission(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles := map[string][]string{
				"admin": {
					PermissionUploadBooks, PermissionGetCreatorBooks, PermissionGetBookStat,
					PermissionMarkComplete, PermissionUploadChapters, PermissionDeleteChapters,
					PermissionApproveOrDenyBooks, PermissionBanUsers, PermissionGetRecentReads,
					PermissionGetNewlyUpdated, PermissionGetRecommendations, PermissionGetAllBooks,
					PermissionGetSpecificBook, PermissionAddToLibrary, PermissionGetAllBooksFromLibrary,
					PermissionRemoveBookFromLibrary, PermissionBuyCoins,
					PermissionCommentOnBooks, PermissionGetAllCommentsOnBook,
				},
				"creators": {
					PermissionUploadBooks, PermissionGetCreatorBooks, PermissionGetBookStat,
					PermissionMarkComplete, PermissionUploadChapters, PermissionDeleteChapters,
					PermissionGetRecentReads, PermissionGetNewlyUpdated, PermissionGetRecommendations,
					PermissionGetSpecificBook, PermissionAddToLibrary, PermissionGetAllBooksFromLibrary,
					PermissionRemoveBookFromLibrary, PermissionBuyCoins,
					PermissionCommentOnBooks, PermissionGetAllCommentsOnBook, PermissionGetAllBooks,
				},
				"readers": {
					PermissionGetRecentReads, PermissionGetNewlyUpdated, PermissionGetRecommendations,
					PermissionGetSpecificBook, PermissionAddToLibrary, PermissionGetAllBooksFromLibrary,
					PermissionRemoveBookFromLibrary, PermissionBuyCoins,
					PermissionCommentOnBooks, PermissionGetAllCommentsOnBook, PermissionGetAllBooks,
				},
			}
			// header := r.Header.Get("Authorization")
			//
			// if header == "" {
			// 	s.Server.Logger.Error("check permission", "status", "authorization header cannot be empty")
			//
			// 	w.Header().Set("Content-Type", "application/json")
			// 	w.WriteHeader(http.StatusNotFound)
			// 	json.NewEncoder(w).Encode(map[string]string{"error": "authorization header cannot be empty"})
			// 	return
			// }
			//
			// headerSplit := strings.Split(header, " ")
			//
			// if len(headerSplit) < 2 || headerSplit[0] != "Bearer" {
			// 	s.Server.Logger.Error("check permission", "status", "malformed header")
			//
			// 	w.Header().Set("Content-Type", "application/json")
			// 	w.WriteHeader(http.StatusBadRequest)
			// 	json.NewEncoder(w).Encode(map[string]string{"error": "malformed header"})
			// 	return
			// }

			//TODO: add jwt check here, using dummy user for now

			user := struct {
				name string
				role string
			}{
				name: "ose",
				role: "creator",
			}

			rolePermissions := roles[user.role]
			hasPermission := false

			for _, allowedPerms := range permissions {
				for _, rolePerms := range rolePermissions {
					if rolePerms == allowedPerms {
						hasPermission = true
						break
					}
				}
			}

			if !hasPermission {
				s.Server.Logger.Error("check permission", "status", "permission denied")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "permission denied"})
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "user", user)))
		})
	}
}
