package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/oseayemenre/pagesy/internal/config"
	"github.com/oseayemenre/pagesy/internal/logger"
	"github.com/oseayemenre/pagesy/internal/store"
)

type Api struct {
	router      *chi.Mux
	logger      logger.Logger
	objectStore store.ObjectStore
	store       store.Store
	config      *config.Config
}

func New(
	router *chi.Mux,
	logger logger.Logger,
	objectStore store.ObjectStore,
	store store.Store,
	config *config.Config,
) *Api {
	return &Api{
		router:      router,
		logger:      logger,
		objectStore: objectStore,
		store:       store,
		config:      config,
	}
}

func (a *Api) RegisterRoutes() {
	a.router.Route("/api/v1", func(r chi.Router) {
		r.Use(a.LoggingMiddleware)
		r.Route("/auth", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(a.RedirectIfCookieExistsAndIsValid)
				r.Route("/google", func(r chi.Router) {
					r.Get("/", a.HandleGoogleSignIn)
					r.Get("/callback", a.HandleGoogleSignInCallback)
				})

				r.Post("/onboarding", a.HandleOnboarding)
				r.Post("/register", a.HandleRegister)
				r.Post("/login", a.HandleLogin)
			})

			r.Post("/logout", a.HandleLogout)
			r.Post("/refresh-token", a.HandleRefreshToken)
		})

		r.Route("/books", func(r chi.Router) {
			r.With(a.CheckPermission(PermissionUploadBooks)).Post("/", a.HandleUploadBook)
			r.With(a.CheckPermission(PermissionGetCreatorBooks)).Get("/stats", a.HandleGetBooksStats)
			r.With(a.CheckPermission(PermissionGetCreatorBooks, PermissionGetBooks)).Get("/", a.HandleGetBooks)
			r.With(a.CheckPermission(PermissionGetRecentReads)).Get("/recents", a.HandleGetRecentReads)
			r.Get("/recommended", nil)

			r.Route("/{bookId}", func(r chi.Router) {
				r.With(a.CheckPermission(PermissionGetBooks)).Get("/", a.HandleGetBook)
				r.With(a.CheckPermission(PermissionDeleteBook)).Delete("/", a.HandleDeleteBook)
				r.With(a.CheckPermission(PermissionEditBook)).Patch("/", a.HandleEditBook)
				r.With(a.CheckPermission(PermissionApproveOrDenyBooks)).Patch("/approval", a.HandleApproveBook)
				r.With(a.CheckPermission(PermissionMarkComplete)).Patch("/complete", a.HandleMarkBookAsComplete)

				r.Route("/chapters", func(r chi.Router) {
					r.With(a.CheckPermission(PermissionUploadChapters)).Post("/", a.HandleUploadChapter)
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

				r.Route("/subscriptions", func(r chi.Router) {
					r.Patch("/", a.HandleMarkBookForSubscription)
				})
			})
		})

		r.Route("/library", func(r chi.Router) {
			r.Get("/", nil)
			r.Put("/books/{bookId}", nil)
			r.Delete("/books/{bookId}", nil)
		})

		r.Route("/coins", func(r chi.Router) {
			r.With(a.CheckPermission(PermissionBuyCoins)).Post("/", a.HandleBuyCoins)
		})

		r.Post("/webhook", a.HandleWebHook)
		r.Patch("/users/{userId}/ban", nil)
	})
}
