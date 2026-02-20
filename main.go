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

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"

	_ "github.com/lib/pq"
	"github.com/oseayemenre/pagesy/docs"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	queueChapterUploaded = "book.chapter_uploaded"
)

type channel interface {
	PublishWithContext(context.Context, string, string, bool, bool, amqp.Publishing) error
}

type server struct {
	router      chi.Router
	logger      *slog.Logger
	store       *sql.DB
	objectStore objectStore
	hub         *hub
	ch          channel
}

func newServer(logger *slog.Logger, store *sql.DB, objectStore objectStore, ch channel) *server {
	s := &server{
		router:      chi.NewRouter(),
		logger:      logger,
		store:       store,
		objectStore: objectStore,
		hub:         newHub(),
		ch:          ch,
	}
	go s.run()
	s.routes()
	return s
}

// @title		Pagesy
// @version	1.0
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

	cloudinaryCfg, err := cloudinary.NewFromParams(os.Getenv("CLOUDINARY_CLOUD"), os.Getenv("CLOUDINARY_KEY"), os.Getenv("CLOUDINARY_SECRET"))

	if err != nil {
		logger.Error(fmt.Sprintf("error configuring cloudinary, %v", err))
		os.Exit(1)
	}

	objectStore := newcloudinaryObject(cloudinaryCfg)

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
	defer db.Close()
	logger.Info("db connected")

	docs.SwaggerInfo.Host = os.Getenv("SWAGGER_HOST")
	docs.SwaggerInfo.Schemes = []string{os.Getenv("SWAGGER_SCHEME")}

	logger.Info("connecting to queue...")
	conn, err := amqp.Dial(os.Getenv("RABBIT_MQ_CONN"))
	if err != nil {
		logger.Error(fmt.Sprintf("error connecting to rabbitmq, %v", err))
		os.Exit(1)
	}
	defer conn.Close()
	logger.Info("queue connected")

	logger.Info("opening channel...")
	ch, err := conn.Channel()
	if err != nil {
		logger.Error(fmt.Sprintf("error opening channel, %v", err))
		os.Exit(1)
	}
	defer ch.Close()
	logger.Info("channel opened")

	_, err = ch.QueueDeclare(queueChapterUploaded, true, false, false, false, nil)
	if err != nil {
		logger.Error(fmt.Sprintf("error declaring queue, %v", err))
		os.Exit(1)
	}

	svr := newServer(logger, db, objectStore, ch)
	port := *flag.String("a", ":3000", "server address")
	flag.Parse()
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
