package routes

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestCreateJWTToken(t *testing.T) {
	expect := "12345"

	secret := "secret"

	token, err := CreateJWTToken(expect, secret)

	if err != nil {
		t.Fatal(err)
	}

	got := &UserClaims{}

	jwt.ParseWithClaims(token, got, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if got.Id != expect {
		t.Fatalf("expected %s, got %s", expect, got.ID)
	}
}
