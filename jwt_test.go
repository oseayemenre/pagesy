package main

import (
	"net/http/httptest"
	"testing"
)

func TestCreateJWTToken(t *testing.T) {
	token, _ := createJWTToken("123")

	id, err := decodeJWTToken(token)

	if err != nil {
		t.Fatal(err.Error())
	}

	if id != "123" {
		t.Fatalf("expected 123, got %v", id)
	}
}

func TestDecodeJWTToken(t *testing.T) {
	token, _ := createJWTToken("123")

	tests := []struct {
		name   string
		value  string
		expect bool
	}{
		{
			name:   "fail on invalid/malformed token",
			value:  "invalid token",
			expect: true,
		},
		{
			name:   "succeed on valid token",
			value:  token,
			expect: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id, err := decodeJWTToken(tc.value)
			if (err != nil) != tc.expect {
				t.Fatalf("expected %v, got %v", tc.expect, err != nil)
			}
			if tc.expect == false && id != "123" {
				t.Fatalf("expected 123 got %v", id)
			}
		})
	}
}

func TestCreateAccessAndRefreshToken(t *testing.T) {
	rr := httptest.NewRecorder()

	createAccessAndRefreshTokens(rr, "123")

	hasAccessToken, hasRefreshToken := false, false

	for _, cookies := range rr.Result().Cookies() {
		if cookies.Name == "access_token" {
			hasAccessToken = true
			return
		}
		if cookies.Name == "refresh_token" {
			hasRefreshToken = true
			return
		}
	}

	if hasAccessToken && hasRefreshToken == false {
		t.Fatalf("expected true got false")
	}
}
