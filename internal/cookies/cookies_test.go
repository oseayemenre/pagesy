package cookies

import (
	"net/http/httptest"
	"testing"
)

func TestCreateAccessAndRefreshTokens(t *testing.T) {
	w := httptest.NewRecorder()

	if err := CreateAccessAndRefreshTokens(w, "123", "secret", "test-service"); err != nil {
		t.Fatal(err)
	}

	cookies := w.Result().Cookies()

	var hasaccesstoken, hashrefreshtoken bool
	var access_token, refresh_token string

	for _, c := range cookies {
		if c.Name == "access_token" {
			access_token = c.Value
			hasaccesstoken = true
		}
		if c.Name == "refresh_token" {
			refresh_token = c.Value
			hashrefreshtoken = true
		}
	}

	if !hasaccesstoken && !hashrefreshtoken {
		t.Fatal("expected true got false")
	}

	if access_token == "" && refresh_token == "" {
		t.Fatal("no token set in cookies")
	}
}
