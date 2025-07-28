package store

import (
	"context"
	"database/sql"
	"testing"

	"github.com/oseayemenre/pagesy/internal/models"
)

func TestCheckIfBookIsEligibleForSubscription(t *testing.T) {
	db := setUpTestDb(t)

	author_id, _ := db.CreateUser(context.TODO(), &models.User{
		Username: "fake_username",
		Email:    "fake_email@email.com",
		Password: "fake_password",
	})

	t.Cleanup(func() {
		db.DB.Exec("DELETE FROM users WHERE id = $1", author_id)
	})

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

	tests := []struct {
		name    string
		bookId  string
		wantErr bool
	}{
		{
			name:    "should return an error if book id isn't a uuid",
			bookId:  "1",
			wantErr: true,
		},
		{
			name:    "should check if book is eligible for subscription",
			bookId:  book_id.String(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eligible, err := db.CheckIfBookIsEligibleForSubscription(context.TODO(), tt.bookId)

			if (err != nil) != tt.wantErr {
				t.Fatalf("expected %v, got %v", tt.wantErr, err != nil)
			}

			if eligible != false {
				t.Fatal("expected false got true")
			}
		})
	}
}
