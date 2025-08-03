package api

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/oseayemenre/pagesy/internal/models"
	"github.com/oseayemenre/pagesy/internal/store"
)

// HandleUploadBooks godoc
//
//	@Summary		Upload a new book
//	@Description	Uploads a book with metadata, schedule, draft chapter, and book cover image
//	@Tags			books
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			name						formData	string		true	"Book name"
//	@Param			description					formData	string		true	"Book description"
//	@Param			genre						formData	[]string	true	"Genre"
//	@Param			language					formData	string		true	"Book language"
//	@Param			chapter_title				formData	string		true	"Draft chapter title"
//	@Param			chapter_content				formData	string		true	"Draft chapter content"
//	@Param			release_schedule_day		formData	[]string	true	"Release days (e.g. Monday, Tuesday)"
//	@Param			release_schedule_chapter	formData	[]int		true	"Chapters per day (e.g. 1, 2)"
//	@Param			book_cover					formData	file		false	"Book cover image (max 3MB)"
//	@Success		201							{object}	models.HandleUploadBooksResponse
//	@Failure		400							{object}	models.ErrorResponse
//	@Failure		413							{object}	models.ErrorResponse
//	@Failure		404							{object}	models.ErrorResponse
//	@Failure		500							{object}	models.ErrorResponse
//	@Router			/books [post]
func (a *Api) HandleUploadBook(w http.ResponseWriter, r *http.Request) {
	user_context := r.Context().Value("user")
	user := user_context.(*models.User)

	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)

	if err := r.ParseMultipartForm(8 << 20); err != nil {
		a.logger.Warn(fmt.Sprintf("error parsing form: %v", err), "service", "HandleUploadBooks")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error parsing form: %v", err))
		return
	}

	defer r.MultipartForm.RemoveAll()

	params := models.HandleUploadBooksRequest{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Genres:      r.FormValue("genre"),
		Language:    r.FormValue("language"),
		ChapterDraft: &models.ChapterDraft{
			Title:   r.FormValue("chapter_title"),
			Content: r.FormValue("chapter_content"),
		},
	}

	days := strings.Split(r.FormValue("release_schedule_day"), ",")
	chapters := strings.Split(r.FormValue("release_schedule_chapter"), ",")

	if len(days) != len(chapters) {
		a.logger.Warn("chapter length and days length must be the same", "service", "HandleUploadBooks")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("chapter length and days length must be the same"))
		return
	}

	for i := range days {
		ch, err := strconv.Atoi(chapters[i])

		if err != nil {
			a.logger.Warn("error converting type string to int", "service", "HandleUploadBooks")
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("error converting type string to int"))
			return
		}

		schedule := models.Schedule{
			Day:      days[i],
			Chapters: ch,
		}

		params.Release_schedule = append(params.Release_schedule,
			schedule)
	}

	if err := validate.Struct(&params); err != nil {
		a.logger.Warn(fmt.Sprintf("validation error: %v", err), "service", "HandleUploadBooks")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("validation error: %v", err))
		return
	}

	schedules := make([]models.Schedule, len(params.Release_schedule))
	for i, rs := range params.Release_schedule {
		schedules[i] = models.Schedule{
			Day:      rs.Day,
			Chapters: rs.Chapters,
		}
	}

	file, header, cover_err := r.FormFile("book_cover")

	var url string
	var fileData []byte

	if cover_err == nil {
		defer file.Close()

		fileData, err := io.ReadAll(file)
		if err != nil {
			a.logger.Error(fmt.Sprintf("error reading bytes: %v", err), "service", "HandleUploadBooks")
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error reading bytes: %v", err))
			return
		}

		if len(fileData) > 3<<20 {
			a.logger.Error("book cover too large", "service", "HandleUploadBooks")
			respondWithError(w, http.StatusRequestEntityTooLarge, fmt.Errorf("book cover too large"))
			return
		}

		if contentType := http.DetectContentType(fileData); !strings.HasPrefix(contentType, "image/") {
			a.logger.Warn("invalid file type", "service", "HandleUploadBooks")
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("invalid file type"))
			return
		}
	}

	bookId, err := a.store.UploadBook(r.Context(), &models.Book{
		Name:        params.Name,
		Description: params.Description,
		Author_Id:   user.Id,
		Genres:      strings.Split(params.Genres, ","),
		Chapter_Draft: models.Chapter{
			Title:   params.ChapterDraft.Title,
			Content: params.ChapterDraft.Content,
		},
		Language:         params.Language,
		Release_schedule: schedules,
	})

	if err != nil {
		if err == store.ErrGenresNotFound {
			a.logger.Error(err.Error(), "service", "HandleUploadBooks")
			respondWithError(w, http.StatusNotFound, err)
		}
		a.logger.Error(err.Error(), "service", "HandleUploadBooks")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	if cover_err == nil {
		url, err = a.objectStore.UploadFile(r.Context(), bytes.NewReader(fileData), fmt.Sprintf("%s_%s", bookId.String(), header.Filename))

		if err != nil {
			a.logger.Error(err.Error(), "service", "HandleUploadBooks")
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}

		if err := a.store.UpdateBookImage(r.Context(), url, bookId.String()); err != nil {
			a.logger.Error(err.Error(), "service", "HandleUploadBooks")
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}
	}

	if cover_err != nil && cover_err != http.ErrMissingFile {
		a.logger.Error(fmt.Sprintf("error uploading image: %v", err), "service", "HandleUploadBooks")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error uploading image: %v", err))
		return
	}

	respondWithSuccess(w, http.StatusCreated, &models.HandleUploadBooksResponse{Id: bookId.String()})
}

// HandleGetBooksStats GoDoc
//
//	@Summary		Get books stats
//	@Description	Get all books by id
//	@Tags			books
//	@Produce		json
//	@Param			creator_id	query		string	false	"creator id"
//	@Param			offset		query		string	true	"offset"
//	@Param			limit		query		string	true	"limit"
//	@Failure		500			{object}	models.ErrorResponse
//	@Success		200			{object}	models.HandleGetBooksStatsResponse
//	@Router			/books/stats [get]
func (a *Api) HandleGetBooksStats(w http.ResponseWriter, r *http.Request) {
	user_context := r.Context().Value("user")
	user := user_context.(*models.User)

	var id string

	creatorId := r.URL.Query().Get("creator_id")
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))

	if err != nil {
		a.logger.Warn("error converting string to int", "service", "HandleGetBooksStats")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))

	if err != nil {
		a.logger.Warn("error converting string to int", "service", "HandleGetBooksStats")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if user.Role == "admin" && creatorId != "" {
		id = creatorId
	} else {
		id = user.Id.String()
	}

	book, err := a.store.GetBooksStats(r.Context(), id, offset, limit)

	if err != nil {
		if err == store.ErrCreatorsBooksNotFound {
			respondWithSuccess(w, http.StatusOK, &models.HandleGetBooksStatsResponse{Books: []models.HandleGetBooksResponseBook{}})
			return
		}

		a.logger.Error(err.Error(), "service", "HandleGetBooksStats")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	var booksResponse []models.HandleGetBooksResponseBook

	for _, b := range book {
		newBook := &models.HandleGetBooksResponseBook{
			Id:             b.Id,
			Name:           b.Name,
			Description:    b.Description,
			Views:          b.Views,
			Completed:      b.Completed,
			Approved:       b.Approved,
			No_Of_Chapters: b.No_Of_Chapters,
			Image:          b.Image.String,
			Language:       b.Language,
			Created_at:     b.Created_at,
			Updated_at:     b.Updated_at,
		}

		for _, g := range b.Genres {
			newBook.Genres = append(newBook.Genres, g)
		}

		for _, s := range b.Release_schedule {
			release_schedule := &models.Schedule{
				Day:      s.Day,
				Chapters: s.Chapters,
			}

			newBook.Release_schedule = append(newBook.Release_schedule, *release_schedule)
		}

		booksResponse = append(booksResponse, *newBook)
	}

	respondWithSuccess(w, http.StatusOK, &models.HandleGetBooksStatsResponse{Books: booksResponse})
}

func handleGetBooksHelper(w http.ResponseWriter, books []models.Book) {
	var booksResponse []models.HandleGetBooksBooks

	for _, b := range books {
		newBook := &models.HandleGetBooksBooks{
			Name:           b.Name,
			Description:    b.Description,
			Views:          b.Views,
			Rating:         b.Rating,
			Image:          b.Image.String,
			No_Of_Chapters: b.No_Of_Chapters,
		}

		for _, g := range b.Genres {
			newBook.Genres = append(newBook.Genres, g)
		}

		for _, s := range b.Release_schedule {
			release_schedule := &models.Schedule{
				Day:      s.Day,
				Chapters: s.Chapters,
			}

			newBook.Release_schedule = append(newBook.Release_schedule, *release_schedule)
		}

		booksResponse = append(booksResponse, *newBook)
	}

	respondWithSuccess(w, http.StatusOK, &models.HandleGetBooksResponse{Books: booksResponse})
}

// HandleGetBooks godoc
//
//	@Summary		Get books
//	@Description	Get books by genre, language, both or get all books
//	@Produce		json
//	@Tags			books
//	@Param			genre		query		string	false	"book genres"
//	@Param			language	query		string	false	"book language"
//	@Param			offset		query		string	true	"offset number"
//	@Param			limit		query		string	true	"limit number"
//	@Param			sort		query		string	false	"sort by e.g views or updated"
//	@Param			order		query		string	false	"order e.g asc or desc"
//	@Success		200			{object}	models.HandleGetBooksResponse
//	@Failure		500			{object}	models.ErrorResponse
//	@Router			/books [get]
func (a *Api) HandleGetBooks(w http.ResponseWriter, r *http.Request) {
	genre := r.URL.Query()["genre"]
	language := r.URL.Query()["language"]
	sort := strings.ToLower(r.URL.Query().Get("sort"))
	order := strings.ToLower(r.URL.Query().Get("order"))

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))

	if err != nil {
		a.logger.Warn("error converting string to int", "service", "HandleGetBooks")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))

	if err != nil {
		a.logger.Warn("error converting string to int", "service", "HandleGetBooks")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if sort != "updated" && sort != "views" {
		sort = "views"
	}

	if order != "asc" && order != "desc" {
		order = "desc"
	}

	if len(genre) > 0 && len(language) < 1 {
		books, err := a.store.GetBooksByGenre(r.Context(), genre, offset, limit, sort, order)

		if err != nil {
			if err == store.ErrNoBooksUnderThisGenre {
				respondWithSuccess(w, http.StatusOK, &models.HandleGetBooksResponse{Books: []models.HandleGetBooksBooks{}})
				return
			}
			a.logger.Error(err.Error(), "service", "HandleGetBooks")
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}

		handleGetBooksHelper(w, books)
		return
	}

	if len(genre) < 1 && len(language) > 0 {
		books, err := a.store.GetBooksByLanguage(r.Context(), language, offset, limit, sort, order)

		if err != nil {
			if err == store.ErrNoBooksUnderThisLanguage {
				respondWithSuccess(w, http.StatusOK, &models.HandleGetBooksResponse{Books: []models.HandleGetBooksBooks{}})
				return
			}
			a.logger.Error(err.Error(), "service", "HandleGetBooks")
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}

		handleGetBooksHelper(w, books)
		return
	}

	if len(genre) > 0 && len(language) > 0 {
		books, err := a.store.GetBooksByGenreAndLanguage(r.Context(), genre, language, offset, limit, sort, order)

		if err != nil {
			if err == store.ErrNoBooksUnderThisGenreAndLanguage {
				respondWithSuccess(w, http.StatusOK, &models.HandleGetBooksResponse{Books: []models.HandleGetBooksBooks{}})
				return
			}
			a.logger.Error(err.Error(), "service", "HandleGetBooks")
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}

		handleGetBooksHelper(w, books)
		return
	}

	books, err := a.store.GetAllBooks(r.Context(), offset, limit, sort, order)
	if err != nil {
		a.logger.Error(err.Error(), "service", "HandleGetBooks")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	handleGetBooksHelper(w, books)
}

// HandleGetBook godoc
//
//	@Summary		Get book
//	@Description	Get book by id
//	@Tags			books
//	@Produce		json
//	@Param			bookId	path		string	true	"book id"
//	@Failure		404		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Success		200		{object}	models.HandleGetBookResponse
//	@Router			/books/{bookId} [get]
func (a *Api) HandleGetBook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "bookId")

	book, err := a.store.GetBook(r.Context(), id)

	if err != nil {
		if err == store.ErrBookNotFound {
			a.logger.Warn(err.Error(), "service", "HandleGetBook")
			respondWithError(w, http.StatusNotFound, err)
			return

		}
		a.logger.Error(err.Error(), "service", "HandleGetBook")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	var chaptersForPreview []models.ChaptersBookPreview

	for _, chapters := range book.Chapters {
		chaptersForPreview = append(chaptersForPreview, models.ChaptersBookPreview{
			Chapter_no: chapters.Chapter_no,
			Title:      chapters.Title,
			Created_at: chapters.Created_at,
		})
	}

	response := &models.HandleGetBookResponse{
		Name:             book.Name,
		Description:      book.Description,
		Views:            book.Views,
		Rating:           book.Rating,
		Image:            book.Image.String,
		Genres:           book.Genres,
		No_Of_Chapters:   book.No_Of_Chapters,
		Chapters:         chaptersForPreview,
		Release_schedule: book.Release_schedule,
	}

	respondWithSuccess(w, http.StatusOK, response)
}

// HandleDeleteBook Godoc
//
//	@Summary		Delete book
//	@Description	Delete book by id
//	@Tags			books
//	@Produce		json
//	@Param			bookId	path		string	true	"book id"
//	@Failure		404		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Success		204
//	@Router			/books/{bookId} [delete]
func (a *Api) HandleDeleteBook(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*models.User)
	bookId := chi.URLParam(r, "bookId")

	err := a.store.DeleteBook(r.Context(), bookId, user.Id.String())

	if err != nil {
		a.logger.Warn(err.Error(), "service", "HandleDeleteBook")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithSuccess(w, http.StatusNoContent, nil)
}

// HandleEditBook Godoc
//
//	@Summary		Edit book details
//	@Description	Edit book name, image, description, genre or release schedule
//	@Tags			books
//	@Accept			multipart/form-data
//	@Param			bookId						path		string		true	"Book Id"
//	@Param			name						formData	string		false	"Book name"
//	@Param			description					formData	string		false	"Book Description"
//	@Param			image						formData	string		false	"Book Image"
//	@Param			genres						formData	[]string	false	"Book Genres"
//	@Param			release_schedule_day		formData	[]string	false	"Release schedule days"
//	@Param			release_schedule_chapter	formData	[]string	false	"Release schedule chapters"
//	@Failure		400							{object}	models.ErrorResponse
//	@Failure		404							{object}	models.ErrorResponse
//	@Failure		413							{object}	models.ErrorResponse
//	@Faiure			500 {object} models.ErrorResponse
//	@Success		204
//	@Router			/books/{bookId} [patch]
func (a *Api) HandleEditBook(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*models.User)
	bookId := chi.URLParam(r, "bookId")
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)

	if err := r.ParseMultipartForm(8 << 20); err != nil {
		a.logger.Warn(fmt.Sprintf("error parsing form: %v", err), "service", "HandleEditBook")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error parsing form: %v", err))
		return
	}

	defer r.MultipartForm.RemoveAll()

	days := strings.Split(r.FormValue("release_schedule_day"), ",")
	chapters := strings.Split(r.FormValue("release_schedule_chapter"), ",")

	if len(days) == 1 && days[0] == "" {
		days = []string{}
	}

	if len(chapters) == 1 && chapters[0] == "" {
		chapters = []string{}
	}

	if len(days) != len(chapters) {
		a.logger.Warn("chapter length and days length must be the same", "service", "HandleEditBook")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("chapter length and days length must be the same"))
		return
	}

	genres := strings.Split(r.FormValue("genres"), ",")

	if len(genres) == 1 && genres[0] == "" {
		genres = []string{}
	}

	file, header, err := r.FormFile("book_cover")

	var url string

	if err == nil {
		defer file.Close()

		fileData, err := io.ReadAll(file)
		if err != nil {
			a.logger.Error(fmt.Sprintf("error reading bytes: %v", err), "service", "HandleEditBook")
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error reading bytes: %v", err))
			return
		}

		if len(fileData) > 3<<20 {
			a.logger.Error("book cover too large", "service", "HandleEditBook")
			respondWithError(w, http.StatusRequestEntityTooLarge, fmt.Errorf("book cover too large"))
			return
		}

		if contentType := http.DetectContentType(fileData); !strings.HasPrefix(contentType, "image/") {
			a.logger.Warn("invalid file type", "service", "HandleEditBook")
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("invalid file type"))
			return
		}

		url, err = a.objectStore.UploadFile(r.Context(), bytes.NewReader(fileData), fmt.Sprintf("%s_%s", bookId, header.Filename))

		if err != nil {
			a.logger.Error(err.Error(), "service", "HandleEditBook")
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}
	}

	if err != nil && err != http.ErrMissingFile {
		a.logger.Error(fmt.Sprintf("error uploading image: %v", err), "service", "HandleEditBook")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error uploading image: %v", err))
		return
	}

	parse_id, err := uuid.Parse(bookId)

	if err != nil {
		a.logger.Warn("book id is not a uuid", "service", "HandleEditBook")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("book id is not a uuid"))
		return
	}

	params := &models.Book{
		Id:          parse_id,
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Genres:      genres,
		Author_Id:   user.Id,
		Image: sql.NullString{
			String: url,
			Valid:  true,
		},
	}

	if len(days) > 0 && len(chapters) > 0 {
		for i := range days {
			ch, err := strconv.Atoi(chapters[i])

			if err != nil {
				a.logger.Warn("error converting type string to int", "service", "HandleEditBook")
				respondWithError(w, http.StatusBadRequest, fmt.Errorf("error converting type string to int"))
				return
			}

			schedule := models.Schedule{
				Day:      days[i],
				Chapters: ch,
			}

			params.Release_schedule = append(params.Release_schedule,
				schedule)
		}
	}

	if err := a.store.EditBook(r.Context(), params); err != nil {
		if err == store.ErrShouldAtLeasePassOneFieldToUpdate {
			a.logger.Warn(err.Error(), "service", "HandleEditBook")
			respondWithError(w, http.StatusBadRequest, err)
			return
		}

		if err == store.ErrBookNotFound {
			a.logger.Warn(err.Error(), "service", "HandleEditBook")
			respondWithError(w, http.StatusNotFound, err)
			return
		}

		a.logger.Error(err.Error(), "service", "HandleEditBook")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithSuccess(w, http.StatusNoContent, nil)
}

// HandleApproveBook godoc
//
//	@Summary		Approve book
//	@Description	Approve book by id
//	@Tags			books
//	@Produce		json
//	@Param			bookId	path		string					true	"Book ID"
//	@Param			param	body		models.ApproveBookParam	true	"Approve book body"
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Success		204
//	@Router			/books/{bookId}/approval [patch]
func (a *Api) HandleApproveBook(w http.ResponseWriter, r *http.Request) {
	bookId := chi.URLParam(r, "bookId")
	param := models.ApproveBookParam{}

	if err := decodeJson(r, &param); err != nil {
		a.logger.Warn(err.Error(), "service", "HandleApproveBook")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if err := validate.Struct(&param); err != nil {
		a.logger.Warn(fmt.Sprintf("error validating fields: %v", err), "service", "HandleApproveBook")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("error validating fields: %v", err))
		return
	}

	if err := a.store.ApproveBook(r.Context(), bookId, param.Approve); err != nil {
		a.logger.Error(err.Error(), "service", "HandleApproveBook")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithSuccess(w, http.StatusNoContent, nil)
}

// HandleMarkBookAsComplete godoc
//
//	@Summary		Mark book as complete
//	@Description	Mark book as complete using id
//	@Tags			books
//	@Produce		json
//	@Param			bookId	path		string						true	"Book ID"
//	@Param			param	body		models.MarkAsCompleteParam	true	"Mark book as complete body"
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Success		204
//	@Router			/books/{bookId}/complete [patch]
func (a *Api) HandleMarkBookAsComplete(w http.ResponseWriter, r *http.Request) {
	bookId := chi.URLParam(r, "bookId")
	param := models.MarkAsCompleteParam{}

	if err := decodeJson(r, &param); err != nil {
		a.logger.Warn(err.Error(), "service", "HandleMarkBookAsComplete")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if err := validate.Struct(&param); err != nil {
		a.logger.Warn(fmt.Sprintf("error validating fields: %v", err), "service", "HandleMarkBookAsComplete")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("error validating fields: %v", err))
		return
	}

	if err := a.store.MarkBookAsComplete(r.Context(), bookId, param.Completed); err != nil {
		a.logger.Error(err.Error(), "service", "HandleMarkBookAsComplete")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithSuccess(w, http.StatusNoContent, nil)
}

// HandleGetRecentReads godoc
//
//	@Summary		Get recent reads
//	@Description	Get user's recent reads
//	@Produce		json
//	@Tags			books
//	@Param			offset	query		string	true	"offset"
//	@Param			limit	query		string	true	"limit"
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Success		200		{object}	models.HandleGetRecentReadsResponse
//	@Router			/books/recents [get]
func (a *Api) HandleGetRecentReads(w http.ResponseWriter, r *http.Request) {
	user_context := r.Context().Value("user")
	user := user_context.(*models.User)

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))

	if err != nil {
		a.logger.Warn("error converting string to int", "service", "HandleGetRecentReads")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))

	if err != nil {
		a.logger.Warn("error converting string to int", "service", "HandleGetRecentReads")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	books, err := a.store.GetRecentReads(r.Context(), user.Id.String(), offset, limit)

	if err != nil {
		if err == store.ErrNoBooksInRecents {
			a.logger.Warn(err.Error(), "service", "HandleGetRecentReads")
			respondWithSuccess(w, http.StatusOK, &models.HandleGetRecentReadsResponse{Books: []models.RecentReadsResponseBooks{}})
			return
		}

		a.logger.Error(err.Error(), "service", "HandleGetRecentReads")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	var recentBooks []models.RecentReadsResponseBooks

	for _, b := range books {
		book := models.RecentReadsResponseBooks{
			Name:            b.Name,
			Image:           b.Image.String,
			LastReadChapter: b.ChapterLastRead,
		}

		duration := time.Since(b.TimeLastOpened)

		timeLastOpened := int(duration.Hours() / 24)

		switch {
		case timeLastOpened == 0:
			book.LastRead = "Today"

		case timeLastOpened == 1:
			book.LastRead = "Yesterday"

		case timeLastOpened > 1 && timeLastOpened < 30:
			book.LastRead = fmt.Sprintf("%d days ago", timeLastOpened)

		default:
			book.LastRead = b.TimeLastOpened.Format("January 2, 2006")
		}

		recentBooks = append(recentBooks, book)
	}

	respondWithSuccess(w, http.StatusOK, &models.HandleGetRecentReadsResponse{Books: recentBooks})
}
