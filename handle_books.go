package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

// handleUploadBook godoc
//
//	@Summary		Upload book
//	@Description	Upload book
//	@Tags			books
//	@Accept			multipart/form-data
//	@Produce		application/json
//	@Param			name						formData	string		true	"Name"
//	@Param			description					formData	string		true	"Description"
//	@Param			genres						formData	[]string	true	"Genre"
//	@Param			language					formData	string		true	"Language"
//	@Param			chapter_title				formData	string		true	"Draft chapter title"
//	@Param			chapter_content				formData	string		true	"Draft chapter content"
//	@Param			release_schedule_day		formData	[]string	true	"Release days (e.g. Monday, Tuesday)"
//	@Param			release_schedule_chapter	formData	[]int		true	"Chapters per day (e.g. 1, 2)"
//	@Param			book_cover					formData	file		false	"Book cover image (max 3MB)"
//	@Failure		400							{object}	errorResponse
//	@Failure		409							{object}	errorResponse
//	@Failure		413							{object}	errorResponse
//	@Failure		404							{object}	errorResponse
//	@Failure		500							{object}	errorResponse
//	@Success		201							{object}	main.handleUploadBook.response
//	@Router			/books [post]
func (s *server) handleUploadBook(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Name            string `validate:"required"`
		Description     string `validate:"required"`
		Genres          string `validate:"required"`
		Language        string `validate:"required"`
		ReleaseSchedule []releaseSchedule
		DraftChapter    draftChapter
	}

	type response struct {
		Id string `json:"id"`
	}

	user_id := r.Context().Value("user").(string)

	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)

	if err := r.ParseMultipartForm(8 << 20); err != nil {
		encode(w, http.StatusInternalServerError, &errorResponse{Error: fmt.Sprintf("error parsing multipart form, %v", err)})
		return
	}
	defer r.MultipartForm.RemoveAll()

	params := request{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Genres:      r.FormValue("genres"),
		Language:    r.FormValue("language"),
		DraftChapter: draftChapter{
			Title:   r.FormValue("chapter_title"),
			Content: r.FormValue("chapter_content"),
		},
	}

	days := strings.Split(r.FormValue("release_schedule_day"), ",")
	chapters := strings.Split(r.FormValue("release_schedule_chapter"), ",")

	if len(days) != len(chapters) {
		encode(w, http.StatusBadRequest, &errorResponse{Error: "chapter length and days length must be the same"})
		return
	}

	for i := range days {
		ch, err := strconv.Atoi(chapters[i])

		if err != nil {
			encode(w, http.StatusBadRequest, &errorResponse{Error: "error converting type string to int"})
			return
		}

		schedule := releaseSchedule{
			Day:      days[i],
			Chapters: ch,
		}

		params.ReleaseSchedule = append(params.ReleaseSchedule,
			schedule)
	}

	if err := validate.Struct(&params); err != nil {
		encode(w, http.StatusBadRequest, &errorResponse{Error: fmt.Sprintf("validation error, %v", err)})
		return
	}

	book_id, err := s.uploadBook(r.Context(), &book{
		name:        params.Name,
		description: params.Description,
		author_id:   user_id,
		genres:      strings.Split(params.Genres, ","),
		draft_chapter: draftChapter{
			Title:   params.DraftChapter.Title,
			Content: params.DraftChapter.Content,
		},
		language:         params.Language,
		release_schedule: params.ReleaseSchedule,
	})

	if errors.Is(err, errBookNameAlreadyTaken) {
		encode(w, http.StatusConflict, &errorResponse{Error: err.Error()})
		return
	}

	if errors.Is(err, errGenresNotFound) {
		encode(w, http.StatusNotFound, &errorResponse{Error: err.Error()})
		return
	}

	if err != nil {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	file, header, err := r.FormFile("book_cover")

	if err != nil && err != http.ErrMissingFile {
		encode(w, http.StatusBadRequest, fmt.Errorf("error uploading image, %v", err))
		return
	}

	if err == nil {
		defer file.Close()

		fileData, err := io.ReadAll(file)
		if err != nil {
			s.logger.Error(fmt.Sprintf("error reading bytes, %v", err))
			encode(w, http.StatusInternalServerError, &errorResponse{Error: fmt.Sprintf("error reading bytes, %v", err)})
			return
		}

		if len(fileData) > 3<<20 {
			encode(w, http.StatusRequestEntityTooLarge, &errorResponse{Error: "book cover too large"})
			return
		}

		if contentType := http.DetectContentType(fileData); !strings.HasPrefix(contentType, "image/") {
			encode(w, http.StatusBadRequest, &errorResponse{Error: "invalid file type"})
			return
		}

		url, err := s.objectStore.upload(r.Context(), fmt.Sprintf("%s_%s", book_id, header.Filename), bytes.NewReader(fileData))
		if err != nil {
			s.logger.Error(err.Error())
			encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
			return
		}

		if err := s.updateBookImage(r.Context(), url, book_id); err != nil {
			s.logger.Error(err.Error())
			encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
			return
		}
	}

	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/api/v1/ws", os.Getenv("WS_HOST")), nil)
	if err != nil {
		s.logger.Error(fmt.Sprintf("error connection to websocket endpoint, %v", err))
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	if err := conn.WriteJSON(&event{Type: NEW_BOOK, Payload: fmt.Sprintf("book %v waiting for approval", book_id)}); err != nil {
		s.logger.Error(fmt.Sprintf("error writing to websocket endpoint, %v", err))
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	encode(w, http.StatusCreated, &response{Id: book_id})
}
