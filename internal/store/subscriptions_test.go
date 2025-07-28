package store

import (
	"context"
	"testing"
)

func TestCheckIfBookIsEligibleForSubscription(t *testing.T) {
	db := setUpTestDb(t)

	book_id, _, err := createUserAndUploadBook(t, db)

	if err != nil {
		t.Fatal(err)
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

func TestMarkBookForSubscription(t *testing.T) {
	db := setUpTestDb(t)

	book_id, author_id, err := createUserAndUploadBook(t, db)

	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		bookId    string
		author_id string
		wantErr   bool
	}{
		{
			name:    "should return an error if book id isn't a uuid",
			bookId:  "1",
			wantErr: true,
		},
		{
			name:      "should mark book for subscription",
			bookId:    book_id.String(),
			author_id: author_id.String(),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.MarkBookForSubscription(context.TODO(), tt.bookId, tt.author_id, true)

			if (err != nil) != tt.wantErr {
				t.Fatalf("expected %v, got %v", tt.wantErr, err != nil)
			}

			if tt.wantErr == false {
				var subscription bool

				query := `
						SELECT subscription FROM books WHERE id = $1;
				`

				if err := db.QueryRow(query, book_id).Scan(&subscription); err != nil {
					t.Fatal(err)
				}

				if subscription != true {
					t.Fatal("expected true got false")
				}
			}
		})
	}
}
