package api

import (
	"net/http"
)

func (a *Api) HandlePostComments(w http.ResponseWriter, r *http.Request) {}

func (a *Api) HandleGetComments(w http.ResponseWriter, r *http.Request) {}

func (a *Api) HandleReplyComments(w http.ResponseWriter, r *http.Request) {}

func (a *Api) HandleDeleteComments(w http.ResponseWriter, r *http.Request) {}

func (a *Api) HandleEditComments(w http.ResponseWriter, r *http.Request) {}
