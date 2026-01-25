package main

import (
	"testing"
)

func TestHandleGetProfile(t *testing.T) {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	db := connectTestDb(t)
	createAndCleanUpUser(t, db)
}
