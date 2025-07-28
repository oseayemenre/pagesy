package store

import (
	"testing"
)

func TestUploadChapter(t *testing.T) {
	db := setUpTestDb(t)

	_, _, err := createUserAndUploadBook(t, db)

	if err != nil {
		t.Fatal(err)
	}
}
