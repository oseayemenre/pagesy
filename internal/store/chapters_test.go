package store

import (
	"context"
	"testing"
	"database/sql"
	"github.com/oseayemenre/pagesy/internal/models"
)

func TestUploadChapter(t *testing.T) {
	db := setUpTestDb(t)

	author_id, _ := db.CreateUser(context.TODO(), &models.User{
		Username: "fake_username",
		Email:    "fake_email@email.com",
		Password: "fake_password",
	})

	defer func() {
		db.DB.Exec("DELETE FROM users WHERE id = $1", author_id)
	}()

	book_id, err := db.UploadBook(context.TODO(), &models.Book{
		Name:        "test book",
		Description: "test book description",
		Image: sql.NullString{
			String: "test book image",
			Valid:  true,
		},
		Author_Id: *author_id,
		Language:  "English",
		Release_schedule: []models.Schedule{
			{
				Day:      "Monday",
				Chapters: 1,
			},
			{
				Day:      "Tuesday",
				Chapters: 2,
			},
		},
		Genres: []string{"Action"},
		Chapter_Draft: models.Chapter{
			Title:   "test book chapter",
			Content: "test book content",
		},
	},
	)

	if err != nil {
		t.Fatalf("error: %v", err)
	}

	db.DB.Exec(`
			UPDATE books
			SET approved = true
			WHERE id = $1;
		`, book_id)
}
