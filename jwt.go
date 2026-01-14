package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type userClaims struct {
	id string
	jwt.RegisteredClaims
}

func createJWTToken(id string, secret string) (string, error) {
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, &userClaims{
		id: id,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "pagesy",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}).SignedString([]byte(secret))

	if err != nil {
		return "", fmt.Errorf("error creating jwt token: %v", err)
	}

	return token, nil
}

func createAccessAndRefreshTokens(w http.ResponseWriter, id string, secret string) error {
	accessToken, err := createJWTToken(id, secret)
	if err != nil {
		return err
	}

	refreshToken, err := createJWTToken(id, secret)
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
