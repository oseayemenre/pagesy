package store

import (
	"context"
	"testing"

	"github.com/oseayemenre/pagesy/internal/models"
)

func TestCheckIfUserExists(t *testing.T) {
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
}

func TestGetUserByEmail(t *testing.T) {
	db := setUpTestDb(t)
	new_user := &models.User{
		Username:     "test_username",
		Display_name: "test_display_name",
		Email:        "test_email",
	}

	id, _ := db.CreateUser(context.TODO(), new_user)

	defer db.Exec(`DELETE FROM users WHERE id = $1`, id)

	user, _ := db.GetUserByEmail(context.TODO(), "test_email")

	if user == nil {
		t.Fatalf("user not found")
	}
}

func TestCreateUser(t *testing.T) {
	db := setUpTestDb(t)
	new_user := &models.User{
		Username:     "test_username",
		Display_name: "test_display_name",
		Email:        "test_email",
	}

	id, _ := db.CreateUser(context.TODO(), new_user)

	defer db.Exec(`DELETE FROM users WHERE id = $1`, id)

	if id == nil {
		t.Fatalf("user not found")
	}
}
