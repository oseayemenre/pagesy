package bcrypt

import (
	"golang.org/x/crypto/bcrypt"
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "password"

	hash, _ := HashPassword(password)

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	if err != nil {
		t.Fatalf("Password comparison failed: %v", err)
	}
}

func TestComparePassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("test-password"), bcrypt.DefaultCost)

	if err := ComparePassword("test-password", string(hash)); err != nil {
		t.Fatal(err)
	}
}
