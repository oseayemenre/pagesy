package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	amqp "github.com/rabbitmq/amqp091-go"
)

// handleUploadChapter godoc
//
//	@Summary		Upload chapter
//	@Description	Upload chapter
//	@Tags			chapters
//	@Accept			json
//	@Produce		json
//	@Param			bookID	path		string								true	"book id"
//	@Param			param	body		main.handleUploadChapter.request	true	"upload chapter body"
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Success		200		{object}	main.handleUploadChapter.response
//	@Router			/books/{bookID}/chapters [post]
func (s *server) handleUploadChapter(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Title     string `json:"title" validate:"required"`
		ChapterNo int    `json:"chapterNo" validate:"required"`
		Content   string `json:"content" validate:"required"`
	}

	type response struct {
		Id string `json:"id"`
	}

	var params request

	if err := decode(r, &params); err != nil {
		if errors.Is(err, errValidation) {
			encode(w, http.StatusBadRequest, &errorResponse{Error: fmt.Sprintf("invalid data, %v", err)})
			return
		}
		encode(w, http.StatusBadRequest, &errorResponse{Error: "invalid json"})
		return
	}

	userID := r.Context().Value("user").(string)
	bookID := chi.URLParam(r, "bookID")
	id, err := s.uploadChapter(r.Context(), userID, &chapter{
		title:     params.Title,
		chapterNo: params.ChapterNo,
		content:   params.Content,
		bookID:    bookID,
	})
	if errors.Is(err, errBookNotFound) {
		encode(w, http.StatusNotFound, &errorResponse{Error: err.Error()})
		return
	}
	if err != nil {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	book, err := s.getBook(r.Context(), bookID)
	if err != nil {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	message := fmt.Sprintf("%v chapter %v", book.name, params.ChapterNo)

	messageBody, err := json.Marshal(struct {
		BookID  string
		Message string
	}{
		BookID:  bookID,
		Message: message,
	})

	if err := s.ch.PublishWithContext(r.Context(), "", queueChapterUploaded, false, false, amqp.Publishing{ContentType: "application/json", DeliveryMode: amqp.Persistent, Body: messageBody}); err != nil {
		s.logger.Error(fmt.Sprintf("error publishing message to queue, %v", err))
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	s.hub.broadcast <- &event{Type: CHAPTER_UPLOADED, Payload: chapterUploadEvent{BookId: bookID, Message: message}}

	encode(w, http.StatusCreated, &response{Id: id})
}

// handleGetChapter godoc
//
//	@Summary		Get chapter
//	@Description	Get chapter
//	@Tags			chapters
//	@Produce		json
//	@Param			chapterID	path		string	true	"chapter id"
//	@Failure		404			{object}	errorResponse
//	@Failure		500			{object}	errorResponse
//	@Success		200			{object}	main.handleGetChapter.response
//	@Router			/books/chapters/{chapterID} [get]
func (s *server) handleGetChapter(w http.ResponseWriter, r *http.Request) {
	type response struct {
		ChapterNo int    `json:"chapterNo"`
		Title     string `json:"title"`
		Content   string `json:"content"`
	}

	ch, err := s.getChapter(r.Context(), chi.URLParam(r, "chapterID"))
	if errors.Is(err, errChapterNotFound) {
		encode(w, http.StatusNotFound, &errorResponse{Error: err.Error()})
		return
	}
	if err != nil {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	encode(w, http.StatusOK, &response{Title: ch.title, ChapterNo: ch.chapterNo, Content: ch.content})
}

// handleDeleteChapter godoc
//
//	@Summary		Delete chapter
//	@Description	Delete chapter
//	@Tags			chapters
//	@Produce		json
//	@Param			bookID		path		string	true	"book id"
//	@Param			chapterID	path		string	true	"chapter id"
//	@Failure		404			{object}	errorResponse
//	@Failure		500			{object}	errorResponse
//	@Success		204
//	@Router			/books/{bookID}/chapters/{chapterID} [delete]
func (s *server) handleDeleteChapter(w http.ResponseWriter, r *http.Request) {
	if err := s.deleteChapter(r.Context(), r.Context().Value("user").(string), chi.URLParam(r, "bookID"), chi.URLParam(r, "chapterID")); err != nil {
		if errors.Is(err, errBookNotFound) || errors.Is(err, errChapterNotFound) {
			encode(w, http.StatusNotFound, &errorResponse{Error: err.Error()})
			return
		}
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	encode(w, http.StatusNoContent, nil)
}
