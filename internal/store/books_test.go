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
				_, err := db.DB.Exec(`TRUNCATE books CASCADE;`)
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

func TestGetBooksStats(t *testing.T) {
	author_id := "dc5e215a-afd4-4f70-aa80-3e360fa1d9e4" //TODO: fix this later
	tests := []struct {
		name      string
		author_id string
		offset    int
		wantErr   bool
	}{
		{
			name:      "should return an error if author doesn't exist",
			author_id: "",
			wantErr:   true,
		},
		{
			name:      "should return book stats",
			author_id: author_id,
			offset:    0,
			wantErr:   false,
		},
	}

	db := setUpTestDb(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.GetBooksStats(context.TODO(), tt.author_id, tt.offset)

			if (err != nil && err != ErrCreatorsBooksNotFound) != tt.wantErr {
				t.Fatalf("wanted: %v, got: %v", tt.wantErr, (err != nil && err != ErrCreatorsBooksNotFound))
			}
		})
	}
}

func TestGetBooksByLanguage(t *testing.T) {
	tests := []struct {
		name    string
		genres  []string
		wantErr bool
	}{
		{
			name:    "should return an error if language does not exist",
			genres:  []string{"non-existent language"},
			wantErr: true,
		},
		{
			name:    "should return books by language",
			genres:  []string{"English"},
			wantErr: false,
		},
	}

	db := setUpTestDb(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.GetBooksByLanguage(context.TODO(), tt.genres)

			if (err != nil && err != ErrNoBooksUnderThisLanguage) != tt.wantErr {
				t.Fatalf("wanted: %v, got: %v", tt.wantErr, (err != nil && err != ErrNoBooksUnderThisLanguage))
			}
		})
	}
}
