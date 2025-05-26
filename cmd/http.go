package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/oseayemenre/pagesy/internal/logger"
	"github.com/oseayemenre/pagesy/internal/routes"
	"github.com/oseayemenre/pagesy/internal/shared"
	"github.com/spf13/cobra"
)

type Server struct {
	*shared.Server
}

func NewServer(logger logger.Logger) *Server {
	return &Server{
		Server: &shared.Server{
			Logger: logger,
		},
	}
}

func (s *Server) Mount() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("woosh! 🚀🚀\n"))
	})

	s.Server.Router = r

	server := routes.NewServer(s.Server)

	server.RegisterRoutes()

	return r
}

func HTTPCommand(ctx context.Context) *cobra.Command {
	var addr int
	var env string

	cmd := &cobra.Command{
		Use:   "http",
		Short: "run pagesy http server",
		RunE: func(cmd *cobra.Command, args []string) error {
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

			var handler slog.Handler

			switch env {
			case "dev":
				handler = slog.Handler(slog.NewTextHandler(os.Stderr, nil))
				break
			case "prod":
				handler = slog.Handler(slog.NewJSONHandler(os.Stderr, nil))
				break

			default:
				return fmt.Errorf("environment can only be dev or prod")
			}

			baseLogger := slog.New(handler).With(
				slog.String("app", "pagesly"),
				slog.String("runtime", runtime.Version()),
				slog.String("os", runtime.GOOS),
				slog.String("architecture", runtime.GOARCH),
				slog.String("version", "1.0"),
			)

			logger := logger.NewSlogLogger(baseLogger)

			baseServer := NewServer(logger)

			httpServer := &http.Server{
				Addr:         fmt.Sprintf(":%d", addr),
				Handler:      baseServer.Mount(),
				ReadTimeout:  15 * time.Second,
				WriteTimeout: 10 * time.Second,
			}
			errCh := make(chan error, 1)

			logger.Info("server startup", "status", fmt.Sprintf("server starting on port: %d", addr))
			go func() {
				if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					errCh <- err
				}
			}()

			select {
			case err := <-errCh:
				return err

			case <-sig:
				logger.Info("server shutdown", "status", "kill signal recieved")
				ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
				defer cancel()

				if err := httpServer.Shutdown(ctx); err != nil {
					return fmt.Errorf("error shutting down server: %v", err)
				}

				logger.Info("server shutdown", "status", "shutdown complete...")
				return nil
			}
		},
	}

	cmd.Flags().IntVarP(&addr, "addr", "a", 8080, "server address")
	cmd.Flags().StringVarP(&env, "env", "e", "dev", "current working environment")

	return cmd
}
