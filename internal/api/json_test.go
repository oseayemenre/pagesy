package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRespondWithSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	data := struct {
		Name string
	}{
		Name: "fake_data",
	}

	respondWithSuccess(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, w.Code)
	}

	header, ok := w.Header()["Content-Type"]

	if !ok {
		t.Fatal("expected application/json, got \"\"")
	}

	if header[0] != "application/json" {
		t.Fatalf("expected application/json, got %s", header[0])
	}

	var got struct {
		Name string
	}

	err := json.Unmarshal(w.Body.Bytes(), &got)
	if err != nil {
		t.Fatalf("error unmarshalling response: %v", err)
	}

	if !reflect.DeepEqual(got, data) {
		t.Fatalf("expected %+v, got %+v", data, got)
	}
}

func TestDecodeJson(t *testing.T) {
	expect := struct {
		Name string
	}{
		Name: "fake_data",
	}

	body, err := json.Marshal(&expect)

	if err != nil {
		t.Fatalf("error marshalling body: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer(body))

	got := struct{ Name string }{}

	if err := decodeJson(req, &got); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expect, got) {
		t.Fatalf("expected %+v, got %+v", expect, got)
	}
}
