package jwt

import (
	"testing"
	"time"

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

func TestDecodeJWTToken(t *testing.T) {
	expect := "123456789"
	secret := "secert"

	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, &UserClaims{
		Id: expect,
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer:    "pagesy",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}).SignedString([]byte(secret))

	got, _ := DecodeJWTToken(token, secret)

	if expect != got {
		t.Fatalf("expected %s, got %s", expect, got)
	}
}
