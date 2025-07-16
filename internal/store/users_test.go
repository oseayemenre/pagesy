package store

import (
	"context"
	"testing"

	"github.com/oseayemenre/pagesy/internal/models"
)

func TestCheckIfUserExists(t *testing.T) {
	t.Run("should return user id", func(t *testing.T) {
		db := setUpTestDb(t)
		id, _ := db.CreateUser(context.TODO(), &models.User{
			Username:     "test_username",
			Display_name: "test_display_name",
			Email:        "test_email",
		})

		defer db.Exec(`DELETE FROM users WHERE id = $1`, id)

		user_id, _ := db.CheckIfUserExists(context.TODO(), "test_email")

		if user_id.String() != id.String() {
			t.Fatalf("expcted %s, got %s", id.String(), user_id.String())
		}
	})
}
