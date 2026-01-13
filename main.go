package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	_ "github.com/oseayemenre/pagesy/docs"
)

type server struct {
	router chi.Router
	logger *slog.Logger
	store  *sql.DB
}

func newServer(logger *slog.Logger, store *sql.DB) *server {
	s := &server{
		router: chi.NewRouter(),
		logger: logger,
		store:  store,
	}
	s.routes()
	return s
}

// @title		Pagesy
// @version	1.0
// @host		localhost:3000
// @BasePath	/api/v1
func main() {
	godotenv.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	logger.Info("connecting to db...")
	store, err := sql.Open("postgres", os.Getenv("DB_CONN"))

	if err != nil {
		logger.Error(fmt.Sprintf("error connecting db: %v", err))
		os.Exit(1)
	}

	if err := store.Ping(); err != nil {
		logger.Error(fmt.Sprintf("error pinging db: %v", err))
		os.Exit(1)
	}

	logger.Info("db connected")

	svr := newServer(logger, store)
	port := *flag.String("a", ":3000", "server address")
	httpSvr := &http.Server{
		Addr:    port,
		Handler: svr.router,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	logger.Info(fmt.Sprintf("server starting on port %v...", strings.Trim(port, ":")))
	go func() {
		if err := httpSvr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(fmt.Sprintf("server error: %v", err))
			os.Exit(1)
		}
	}()
	logger.Info("server up and running")

	<-ctx.Done()
	logger.Info("kill signal recieved...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpSvr.Shutdown(shutdownCtx); err != nil {
		logger.Error(fmt.Sprintf("error shutting down server: %v", err))
		os.Exit(1)
	}
	logger.Info("shutdown complete")
}
