package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type userClaims struct {
	Id string
	jwt.RegisteredClaims
}

func createJWTToken(id string) (string, error) {
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, &userClaims{
		Id: id,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "pagesy",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}).SignedString([]byte(os.Getenv("JWT_SECRET")))

	if err != nil {
		return "", fmt.Errorf("error creating jwt token, %v", err)
	}

	return token, nil
}

func decodeJWTToken(token string) (string, error) {
	var user userClaims
	if _, err := jwt.ParseWithClaims(token, &user, func(t *jwt.Token) (any, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	}); err != nil {
		return "", fmt.Errorf("error parsing token, %v", err)
	}

	return user.Id, nil
}

func createAccessAndRefreshTokens(w http.ResponseWriter, id string) error {
	accessToken, err := createJWTToken(id)
	if err != nil {
		return err
	}

	refreshToken, err := createJWTToken(id)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		MaxAge:   24 * 60 * 60,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		MaxAge:   24 * 60 * 60,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}
