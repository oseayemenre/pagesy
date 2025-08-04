package store

import (
	"context"
	"github.com/oseayemenre/pagesy/internal/models"
	"testing"
)

func TestUpdateUserCoinCount(t *testing.T) {
	db := setUpTestDb(t)
	id, _ := db.CreateUser(context.TODO(), &models.User{
		Username:     "test_username",
		Display_name: "test_display_name",
		Email:        "test_email",
	})

	t.Cleanup(func() {
		db.Exec(`DELETE FROM users WHERE id = $1`, id)
	})

	tests := []struct {
		name        string
		user_id     string
		expectError bool
	}{
		{
			name:        "should return an error if user id is not a uuid",
			user_id:     "",
			expectError: true,
		},
		{
			name:        "should update user's coin count",
			user_id:     id.String(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.UpdateUserCoinCount(context.TODO(), tt.user_id, 50)

			if (err != nil) != tt.expectError {
				t.Fatalf("expected %v, got %v", tt.expectError, err != nil)
			}

			if tt.expectError == false {
				query := `
					SELECT coins FROM users WHERE id = $1;
				`

				var coins int

				if err := db.DB.QueryRow(query, id).Scan(&coins); err != nil {
					t.Fatal(err)
				}

				if coins != 50 {
					t.Fatalf("expected 50, got %d", coins)
				}
			}
		})
	}
}
