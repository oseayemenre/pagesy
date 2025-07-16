package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	Id string
	*jwt.RegisteredClaims
}

func CreateJWTToken(id string, secret string) (string, error) {
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, &UserClaims{
		Id: id,
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer:    "pagesy",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}).SignedString([]byte(secret))

	if err != nil {
		return "", fmt.Errorf("error decoding jwt token: %v", err)
	}

	return token, nil
}
