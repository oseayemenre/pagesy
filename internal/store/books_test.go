package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/oseayemenre/pagesy/internal/models"
)

func TestUploadBook(t *testing.T) {
	author_id, err := uuid.Parse("dc5e215a-afd4-4f70-aa80-3e360fa1d9e4") //TODO: fix this later

	if err != nil {
		t.Fatalf("error parsing uuid: %v", err)
	}

	tests := []struct {
		name    string
		book    *models.Book
		wantErr bool
	}{
		{
			name:    "should return an error if fields are missing",
			book:    &models.Book{Name: "test book"},
			wantErr: true,
		},
		{
			name: "should return an error if genre doesn't exist",
			book: &models.Book{
				Name:        "test book",
				Description: "test book description",
				Image:       "test book image",
				Author_Id:   author_id,
				Language:    "English",
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
				Genres: []string{"Action", "Non-existent genre"},
				Chapter_Draft: models.Chapter{
					Title:   "test book chapter",
					Content: "test book content",
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error if author id doesn't exist",
			book: &models.Book{
				Name:        "test book",
				Description: "test book description",
				Image:       "test book image",
				Author_Id:   uuid.New(),
				Language:    "English",
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
			wantErr: true,
		},
		{
			name: "should create the book",
			book: &models.Book{
				Name:        "test book",
				Description: "test book description",
				Image:       "test book image",
				Author_Id:   author_id,
				Language:    "English",
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
			wantErr: false,
		},
	}

	db := setUpTestDb(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				_, err := db.Exec(`TRUNCATE books CASCADE;`)
				if err != nil {
					t.Fatalf("error truncating tables: %v", err)
				}
			})

			err := db.UploadBook(context.TODO(), tt.book)

			if (err != nil) != tt.wantErr {
				t.Fatalf("wanted: %v, got: %v", tt.wantErr, err != nil)
			}
		})
	}
}
