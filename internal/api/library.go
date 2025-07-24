package api

import (
	"net/http"
)

func (a *Api) HandleGetBooksInLibrary(w http.ResponseWriter, r *http.Request) {}

func (a *Api) HandleAddBookInLibrary(w http.ResponseWriter, r *http.Request) {}

func (a *Api) HandleDeleteBookFromLibrary(w http.ResponseWriter, r *http.Request) {}
