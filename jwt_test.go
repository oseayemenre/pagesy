package main

import (
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
