package routes

import "net/http"

func CreateAccessAndRefreshTokens(w http.ResponseWriter, id string, secret string, service string) error {
	access_token, err := CreateJWTToken(id, secret)

	if err != nil {
		return err
	}

	refresh_token, err := CreateJWTToken(id, secret)

	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    access_token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		MaxAge:   24 * 60 * 60,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refresh_token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		MaxAge:   30 * 24 * 60 * 60,
	})

	return nil
}
