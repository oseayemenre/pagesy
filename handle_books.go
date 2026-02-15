package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
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
	type requestReleaseSchedule struct {
		Day      string `validate:"required"`
		Chapters int    `validate:"required"`
	}

	type request struct {
		Name            string `validate:"required"`
		Description     string `validate:"required"`
		Genres          string `validate:"required"`
		Language        string `validate:"required"`
		ReleaseSchedule []requestReleaseSchedule
		DraftChapter    draftChapter
	}

	type response struct {
		Id string `json:"id"`
	}

	userID := r.Context().Value("user").(string)

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

		schedule := requestReleaseSchedule{
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

	var schedule []releaseSchedule

	for _, rs := range params.ReleaseSchedule {
		schedule = append(schedule, releaseSchedule{Day: rs.Day, Chapters: rs.Chapters})
	}

	bookID, err := s.uploadBook(r.Context(), &book{
		name:        params.Name,
		description: params.Description,
		authorID:    userID,
		genres:      strings.Split(params.Genres, ","),
		draftChapter: draftChapter{
			Title:   params.DraftChapter.Title,
			Content: params.DraftChapter.Content,
		},
		language:        params.Language,
		releaseSchedule: schedule,
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

		url, err := s.objectStore.upload(r.Context(), fmt.Sprintf("%s_%s", bookID, header.Filename), bytes.NewReader(fileData))
		if err != nil {
			s.logger.Error(err.Error())
			encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
			return
		}

		if err := s.updateBookImage(r.Context(), url, bookID); err != nil {
			s.logger.Error(err.Error())
			encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
			return
		}
	}

	s.hub.broadcast <- &event{Type: NEW_BOOK, Payload: fmt.Sprintf("%v waiting for approval", params.Name)}

	encode(w, http.StatusCreated, &response{Id: bookID})
}

type responseReleaseSchedule struct {
	Day      string `json:"day"`
	Chapters int    `json:"chapters"`
}

type getResponseBook struct {
	Name            string                    `json:"name"`
	Description     string                    `json:"description"`
	Image           *string                   `json:"image"`
	Views           int                       `json:"views"`
	Rating          float32                   `json:"rating"`
	ChapterCount    int                       `json:"chapterCount"`
	Genres          []string                  `json:"genres"`
	ReleaseSchedule []responseReleaseSchedule `json:"releaseSchedule"`
}

func mapToGetBooks(books []book) []getResponseBook {
	var responseBooks []getResponseBook

	for _, book := range books {
		var image *string
		if book.image.Valid != false {
			image = &book.image.String
		}

		newBook := getResponseBook{Name: book.name, Description: book.description, Image: image, Views: book.views, Rating: book.rating, ChapterCount: book.chapterCount, Genres: book.genres}

		for _, rs := range book.releaseSchedule {
			newBook.ReleaseSchedule = append(newBook.ReleaseSchedule, responseReleaseSchedule{Day: rs.Day, Chapters: rs.Chapters})
		}

		responseBooks = append(responseBooks, newBook)
	}

	return responseBooks
}

// handleGetbooks godoc
//
//	@Summary		Get all books
//	@Description	Get all books
//	@Tags			books
//	@Produce		json
//	@Param			genre		query		string	false	"genre"
//	@Param			language	query		string	false	"language"
//	@Param			sort		query		string	false	"sort"
//	@Param			order		query		string	false	"order"
//	@Param			offset		query		string	true	"offset"
//	@Param			limit		query		string	true	"limit"
//	@Failure		400			{object}	errorResponse
//	@Failure		500			{object}	errorResponse
//	@Success		200			{object}	main.handleGetBooks.response
//	@Router			/books [get]
func (s *server) handleGetBooks(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Books []getResponseBook `json:"books"`
	}

	genre := r.URL.Query()["genre"]
	language := r.URL.Query()["language"]
	sort := strings.ToLower(r.URL.Query().Get("sort"))
	order := strings.ToLower(r.URL.Query().Get("order"))

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		encode(w, http.StatusBadRequest, &errorResponse{Error: "offset should be a valid number"})
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		encode(w, http.StatusBadRequest, &errorResponse{Error: "limit should be a valid number"})
		return
	}

	if sort != "updated" && sort != "views" {
		sort = "views"
	}

	if order != "asc" && order != "desc" {
		order = "desc"
	}

	if len(genre) > 0 && len(language) < 1 {
		books, err := s.getBooksByGenre(r.Context(), genre, offset, limit, sort, order)
		if err != nil && !errors.Is(err, errNoBooksUnderGenre) {
			s.logger.Error(err.Error())
			encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
			return
		}

		encode(w, http.StatusOK, &response{Books: mapToGetBooks(books)})
		return
	}

	if len(genre) < 1 && len(language) > 0 {
		books, err := s.getBooksByLanguage(r.Context(), language, offset, limit, sort, order)
		if err != nil && !errors.Is(err, errNoBooksUnderLanguage) {
			s.logger.Error(err.Error())
			encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
			return
		}

		encode(w, http.StatusOK, &response{Books: mapToGetBooks(books)})
		return
	}

	if len(genre) > 0 && len(language) > 0 {
		books, err := s.getBooksByGenreAndLanguage(r.Context(), genre, language, offset, limit, sort, order)
		if err != nil && !errors.Is(err, errNoBooksUnderGenreAndLanguage) {
			s.logger.Error(err.Error())
			encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
			return
		}

		encode(w, http.StatusOK, &response{Books: mapToGetBooks(books)})
		return
	}

	books, err := s.getAllBooks(r.Context(), offset, limit, sort, order)
	if err != nil {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	encode(w, http.StatusOK, &response{Books: mapToGetBooks(books)})
}

type bookStats struct {
	Name            string                    `json:"name"`
	Description     string                    `json:"description"`
	Image           *string                   `json:"image"`
	Views           int                       `json:"views"`
	Rating          float32                   `json:"rating"`
	ChapterCount    int                       `json:"chapterCount"`
	Completed       bool                      `json:"completed"`
	Approved        bool                      `json:"approved"`
	Genres          []string                  `json:"genres"`
	Language        string                    `json:"language"`
	ReleaseSchedule []responseReleaseSchedule `json:"releaseSchedule"`
	CreatedAt       time.Time                 `json:"createdAt"`
	UpdatedAt       time.Time                 `json:"updatedAt"`
}

func mapToBooksStats(books []book) []bookStats {
	var responseBooks []bookStats

	for _, book := range books {
		var image *string
		if book.image.Valid != false {
			image = &book.image.String
		}

		newBook := bookStats{Name: book.name, Description: book.description, Image: image, Views: book.views, Rating: book.rating, ChapterCount: book.chapterCount, Completed: book.completed, Approved: book.approved, Genres: append([]string{}, book.genres...), Language: book.language, CreatedAt: book.createdAt, UpdatedAt: book.updatedAt}

		for _, rs := range book.releaseSchedule {
			newBook.ReleaseSchedule = append(newBook.ReleaseSchedule, responseReleaseSchedule{Day: rs.Day, Chapters: rs.Chapters})
		}

		responseBooks = append(responseBooks, newBook)
	}

	return responseBooks
}

// handleGetBooksStats godoc
//
//	@Summary		Get books stats
//	@Description	Get books stats
//	@Tags			books
//	@Produce		json
//	@Param			offset	query		string	true	"offset"
//	@Param			limit	query		string	true	"limit"
//	@Failure		400		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Success		200		{object}	main.handleGetBooksStats.response
//	@Router			/books/stats [get]
func (s *server) handleGetBooksStats(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Books []bookStats `json:"books"`
	}

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		encode(w, http.StatusBadRequest, &errorResponse{Error: "offset should be a valid number"})
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		encode(w, http.StatusBadRequest, &errorResponse{Error: "limit should be a valid number"})
		return
	}

	books, err := s.getBooksStats(r.Context(), r.Context().Value("user").(string), offset, limit)
	if err != nil && !errors.Is(err, errUserHasNoBooks) {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	encode(w, http.StatusOK, &response{Books: mapToBooksStats(books)})
}

// handleGetRecentlyReadBooks godoc
//
//	@Summary		Get recently read books
//	@Description	Get recently read books
//	@Tags			books
//	@Produce		json
//	@Param			offset	query		string	true	"offset"
//	@Param			limit	query		string	true	"limit"
//	@Failure		400		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Success		200		{object}	main.handleGetRecentlyReadBooks.response
//	@Router			/books/recently-read [get]
func (s *server) handleGetRecentlyReadBooks(w http.ResponseWriter, r *http.Request) {
	type responseBooks struct {
		Name            string  `json:"name"`
		Image           *string `json:"image"`
		LastReadChapter int     `json:"lastReadChapter"`
		LastReadTime    string  `json:"lastReadTime"`
	}

	type response struct {
		Books []responseBooks `json:"books"`
	}

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		encode(w, http.StatusBadRequest, &errorResponse{Error: "offset should be a valid number"})
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		encode(w, http.StatusBadRequest, &errorResponse{Error: "limit should be a valid number"})
		return
	}

	books, err := s.getRecentlyReadBooks(r.Context(), r.Context().Value("user").(string), offset, limit)
	if err != nil {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	var bksResponse []responseBooks

	for _, book := range books {
		var img *string
		if book.image.Valid != false {
			img = &book.image.String
		}

		var lastReadTime string

		dur := time.Since(book.updatedAt)

		switch {
		case dur <= 24*time.Hour:
			lastReadTime = "Today"
		case dur > 24*time.Hour && dur <= 7*24*time.Hour:
			lastReadTime = fmt.Sprintf("%v days ago", math.Floor(dur.Hours()/24))
		default:
			lastReadTime = fmt.Sprint(book.updatedAt.Format("Jan 2, 2006"))
		}

		bksResponse = append(bksResponse, responseBooks{Name: book.name, Image: img, LastReadChapter: book.lastReadChapter, LastReadTime: lastReadTime})
	}

	encode(w, http.StatusOK, &response{Books: bksResponse})
}

// handleGetRecentlyUploadedBooks godoc
//
//	@Summary		Get recently uploaded books
//	@Description	Get recently uploaded books
//	@Tags			books
//	@Param			offset	query		string	true	"offset"
//	@Param			limit	query		string	true	"limit"
//	@Failure		400		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Success		200		{object}	main.handleGetRecentlyUploadedBooks.response
//	@Router			/books/recently-uploaded [get]
func (s *server) handleGetRecentlyUploadedBooks(w http.ResponseWriter, r *http.Request) {
	type responseBooks struct {
		Name   string  `json:"name"`
		Image  *string `json:"image"`
		Author string  `json:"author"`
	}

	type response struct {
		Books []responseBooks `json:"books"`
	}

	user, err := s.getUser(r.Context(), r.Context().Value("user").(string))
	if err != nil {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	var isAdmin bool
	for _, role := range user.roles {
		if role == "ADMIN" {
			isAdmin = true
			break
		}
	}

	if !isAdmin {
		encode(w, http.StatusUnauthorized, &errorResponse{Error: "role is not admin"})
		return
	}

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		encode(w, http.StatusBadRequest, &errorResponse{Error: "offset should be a valid number"})
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		encode(w, http.StatusBadRequest, &errorResponse{Error: "limit should be a valid number"})
		return
	}

	books, err := s.getRecentlyUploadBooks(r.Context(), offset, limit)
	if err != nil {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	var bksResponse []responseBooks
	for _, book := range books {
		var img *string
		if book.image.Valid != false {
			img = &book.image.String
		}

		bksResponse = append(bksResponse, responseBooks{Name: book.name, Image: img, Author: book.authorName})
	}

	encode(w, http.StatusOK, &response{Books: bksResponse})
}

// handleGetBook godoc
//
//	@Summary		Get a single book
//	@Description	Get a single book
//	@Tags			books
//	@Param			bookID	path		string	true	"book id"
//	@Failure		404		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Success		200		{object}	main.handleGetBook.response
//	@Router			/books/{bookID} [get]
func (s *server) handleGetBook(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL.String())
	type chaptersBookPreview struct {
		ChapterNo  int    `json:"chapterNo"`
		Title      string `json:"title"`
		Created_at string `json:"createdAt"`
	}

	type releaseSchedule struct {
		Day      string `validate:"required"`
		Chapters int    `validate:"required"`
	}

	type response struct {
		Name             string                `json:"name"`
		Description      string                `json:"description"`
		Image            *string               `json:"image"`
		Views            int                   `json:"views"`
		Rating           float32               `json:"rating"`
		Genres           []string              `json:"genres"`
		Completed        bool                  `json:"completed"`
		ChapterCount     int                   `json:"chapterCount"`
		Chapters         []chaptersBookPreview `json:"chapters"`
		Release_schedule []releaseSchedule     `json:"release_schedule"`
	}

	book, err := s.getBook(r.Context(), chi.URLParam(r, "bookID"))
	if errors.Is(err, errBookNotFound) {
		encode(w, http.StatusNotFound, &errorResponse{Error: errBookNotFound.Error()})
		return
	}
	if err != nil {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	var image *string
	if book.image.Valid != false {
		image = &book.image.String
	}

	var chaptersPreviews []chaptersBookPreview
	for _, ch := range book.chapters {
		chaptersPreviews = append(chaptersPreviews, chaptersBookPreview{ChapterNo: ch.chapterNo, Title: ch.title, Created_at: ch.createdAt.Format("Jan 2, 2006")})
	}

	var schedule []releaseSchedule
	for _, rs := range book.releaseSchedule {
		schedule = append(schedule, releaseSchedule{Day: rs.Day, Chapters: rs.Chapters})
	}

	encode(w, http.StatusOK, &response{Name: book.name, Description: book.description, Image: image, Views: book.views, Rating: book.rating, Genres: book.genres, Completed: book.completed, ChapterCount: book.chapterCount, Chapters: chaptersPreviews, Release_schedule: schedule})
}

func (s *server) handleDeleteBook(w http.ResponseWriter, r *http.Request) {
	if err := s.deleteBook(r.Context(), r.Context().Value("user").(string), chi.URLParam(r, "bookID")); err != nil {
		if errors.Is(err, errUserCannotDeleteBook) {
			encode(w, http.StatusBadRequest, &errorResponse{Error: err.Error()})
			return
		}
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}
	encode(w, http.StatusNoContent, nil)
}
