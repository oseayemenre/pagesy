package shared

import (
	"golang.org/x/crypto/bcrypt"
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "password"

	hash, _ := HashPassword(password)

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	if err != nil {
		t.Errorf("Password comparison failed: %v", err)
	}
}
