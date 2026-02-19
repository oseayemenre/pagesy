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

	s.hub.broadcast <- &event{Type: CHAPTER_UPLOADED, Payload: &chapterUploadEvent{BookId: bookID, Message: message}}

	encode(w, http.StatusCreated, &response{Id: id})
}
