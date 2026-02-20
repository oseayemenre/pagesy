package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type message struct {
	BookID  string
	Message string
}

const (
	queueChapterUploaded = "book.chapter_uploaded"
)

func main() {
	godotenv.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

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

	queue, err := ch.QueueDeclare(queueChapterUploaded, true, false, false, false, nil)
	if err != nil {
		logger.Error(fmt.Sprintf("error declaring queue, %v", err))
		os.Exit(1)
	}

	msg, err := ch.ConsumeWithContext(context.Background(), queue.Name, "", false, false, false, false, nil)
	if err != nil {
		logger.Error(fmt.Sprintf("error consuming messages from queue, %v", err))
		os.Exit(1)
	}

	for d := range msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		var userIDs []string

		var newMsg message
		if err := json.Unmarshal(d.Body, &newMsg); err != nil {
			d.Nack(false, false)
			continue
		}

		query :=
			`
				SELECT user_id FROM library WHERE book_id = $1;
			`

		rows, err := db.QueryContext(ctx, query, newMsg.BookID)
		if err != nil {
			cancel()
			logger.Error(fmt.Sprintf("error querying library, %v", err))
			d.Nack(false, true)
			continue
		}

		for rows.Next() {
			var userID string
			if err := rows.Scan(&userID); err != nil {
				rows.Close()
				cancel()
				logger.Error(fmt.Sprintf("error scanning user id, %v", err))
				d.Nack(false, true)
				continue
			}
			userIDs = append(userIDs, userID)
		}
		rows.Close()

		if len(userIDs) == 0 {
			cancel()
			d.Nack(false, false)
			continue
		}

		count := 1
		var values []string
		var args []any

		for _, u := range userIDs {
			values = append(values, fmt.Sprintf("($%d, $%d, $%d)", count, count+1, count+2))
			args = append(args, u, newMsg.BookID, newMsg.Message)
			count += 3
		}

		query = fmt.Sprintf("INSERT INTO notifications (user_id, book_id, message) VALUES %v ON CONFLICT DO NOTHING;", strings.Join(values, ","))

		_, err = db.ExecContext(ctx, query, args...)
		if err != nil {
			cancel()
			logger.Error(fmt.Sprintf("error inserting user notifications, %v", err))
			d.Nack(false, true)
			continue
		}
		cancel()

		if err := d.Ack(false); err != nil {
			cancel()
			logger.Error(fmt.Sprintf("error acknowledging message, %v", err))
			d.Nack(false, true)
			continue
		}
	}
}
