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

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"

	_ "github.com/lib/pq"
	_ "github.com/oseayemenre/pagesy/docs"
)

type server struct {
	router    chi.Router
	validator *validator.Validate
	logger    *slog.Logger
	store     *sql.DB
	s3        *s3.Client
}

func newServer(logger *slog.Logger, store *sql.DB, s3 *s3.Client) *server {
	s := &server{
		router:    chi.NewRouter(),
		validator: validator.New(),
		logger:    logger,
		store:     store,
		s3:        s3,
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
	goth.UseProviders(
		google.New(os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"), fmt.Sprintf("%s/api/v1/auth/google/callback", os.Getenv("HOST"))),
	)

	store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	store.MaxAge(86400)
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	if os.Getenv("STORE_SECURE") == "true" {
		store.Options.Secure = true
	} else {
		store.Options.Secure = false
	}
	store.Options.SameSite = http.SameSiteLaxMode

	gothic.Store = store
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
	if err != nil {
		logger.Error(fmt.Sprintf("unable to load SDK config, %v", err))
		os.Exit(1)
	}

	s3client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	logger.Info("connecting to db...")
	db, err := sql.Open("postgres", os.Getenv("DB_CONN"))

	if err != nil {
		logger.Error(fmt.Sprintf("error connecting db, %v", err))
		os.Exit(1)
	}

	if err := db.Ping(); err != nil {
		logger.Error(fmt.Sprintf("error pinging db, %v", err))
		os.Exit(1)
	}
	logger.Info("db connected")

	svr := newServer(logger, db, s3client)
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
			logger.Error(fmt.Sprintf("server error, %v", err))
			os.Exit(1)
		}
	}()
	logger.Info("server up and running")

	<-ctx.Done()
	logger.Info("kill signal recieved...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpSvr.Shutdown(shutdownCtx); err != nil {
		logger.Error(fmt.Sprintf("error shutting down server, %v", err))
		os.Exit(1)
	}
	logger.Info("shutdown complete")
}
