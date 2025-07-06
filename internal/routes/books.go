package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/oseayemenre/pagesy/internal/models"
	"github.com/oseayemenre/pagesy/internal/shared"
	"github.com/oseayemenre/pagesy/internal/store"
)

// HandleUploadBooks godoc
// @Summary Upload a new book
// @Description Uploads a book with metadata, schedule, draft chapter, and book cover image
// @Tags books
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "Book name"
// @Param description formData string true "Book description"
// @Param genre formData []string true "Genre"
// @Param language formData string true "Book language"
// @Param chapter_title formData string true "Draft chapter title"
// @Param chapter_content formData string true "Draft chapter content"
// @Param release_schedule_day formData []string true "Release days (e.g. Monday, Tuesday)"
// @Param release_schedule_chapter formData []int true "Chapters per day (e.g. 1, 2)"
// @Param book_cover formData file false "Book cover image (max 3MB)"
// @Success 201 {object} models.HandleUploadBooksResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 413 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /books [post]
func (s *Server) HandleUploadBooks(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)

	if err := r.ParseMultipartForm(8 << 20); err != nil {
		s.Server.Logger.Warn(fmt.Sprintf("error parsing form: %v", err), "service", "HandleUploadBooks")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error parsing form: %v", err))
		return
	}

	defer r.MultipartForm.RemoveAll()

	params := models.HandleUploadBooksRequest{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Genres:      r.FormValue("genre"),
		Language:    r.FormValue("language"),
		ChapterDraft: &models.Chapter{
			Title:   r.FormValue("chapter_title"),
			Content: r.FormValue("chapter_content"),
		},
	}

	days := strings.Split(r.FormValue("release_schedule_day"), ",")
	chapters := strings.Split(r.FormValue("release_schedule_chapter"), ",")

	if len(days) != len(chapters) {
		s.Server.Logger.Warn("chapter length and days length must be the same", "service", "HandleUploadBooks")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("chapter length and days length must be the same"))
		return
	}

	for i := range days {
		ch, err := strconv.Atoi(chapters[i])

		if err != nil {
			s.Server.Logger.Warn("error converting type string to int", "service", "HandleUploadBooks")
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

	if err := shared.Validate.Struct(&params); err != nil {
		s.Server.Logger.Warn(fmt.Sprintf("validation error: %v", err), "service", "HandleUploadBooks")
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

	//TODO: Dummy id here. Would handle this properly later
	authorId, err := uuid.Parse("172122bf-e310-42b9-a69f-7382c0d4a74b")

	if err != nil {
		s.Server.Logger.Error(fmt.Sprintf("error parsing uuid: %v", err), "service", "HandleUploadBooks")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error parsing uuid: %v", err))
		return
	}

	bookId, err := s.Server.Store.UploadBook(r.Context(), &models.Book{
		Name:        params.Name,
		Description: params.Description,
		Author_Id:   authorId,
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
			s.Server.Logger.Error(err.Error(), "service", "HandleUploadBooks")
			respondWithError(w, http.StatusNotFound, err)
		}
		s.Server.Logger.Error(err.Error(), "service", "HandleUploadBooks")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	file, header, err := r.FormFile("book_cover")

	var url string

	if err == nil {
		defer file.Close()

		fileData, err := io.ReadAll(file)
		if err != nil {
			s.Server.Logger.Error(fmt.Sprintf("error reading bytes: %v", err), "service", "HandleUploadBooks")
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error reading bytes: %v", err))
			return
		}

		if len(fileData) > 3<<20 {
			s.Server.Logger.Error("book cover too large", "service", "HandleUploadBooks")
			respondWithError(w, http.StatusRequestEntityTooLarge, fmt.Errorf("book cover too large"))
			return
		}

		if contentType := http.DetectContentType(fileData); !strings.HasPrefix(contentType, "image/") {
			s.Server.Logger.Warn("invalid file type", "service", "HandleUploadBooks")
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("invalid file type"))
			return
		}

		url, err = s.Server.ObjectStore.UploadFile(r.Context(), bytes.NewReader(fileData), fmt.Sprintf("%s_%s", bookId, header.Filename))

		if err != nil {
			s.Server.Logger.Error(err.Error(), "service", "HandleUploadBooks")
			respondWithError(w, http.StatusBadRequest, err)
			return
		}

		if err := s.Store.UpdateBookImage(r.Context(), url, bookId); err != nil {
			s.Server.Logger.Error(err.Error(), "service", "HandleUploadBooks")
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}
	}

	if err != nil && err != http.ErrMissingFile {
		s.Server.Logger.Error(fmt.Sprintf("error uploading image: %v", err), "service", "HandleUploadBooks")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error uploading image: %v", err))
		return
	}

	respondWithSuccess(w, http.StatusCreated, &models.HandleUploadBooksResponse{Id: bookId})
}

// HandleGetBooks GoDoc
// @Summary Get All Books Stats
// @Description Get all books by id
// @Tags books
// @Produce json
// @Param creator_id query string false "creator id"
// @Param offset query string true "offset"
// @Param limit query string true "limit"
// @Failure 500 {object} models.ErrorResponse
// @Success 200 {object} models.HandleGetBooksStatsResponse
// @Router /books/stats [get]
func (s *Server) HandleGetBooksStats(w http.ResponseWriter, r *http.Request) {
	user := struct {
		id   string
		role string
	}{
		id:   "dc5e215a-afd4-4f70-aa80-3e360fa1d9e4",
		role: "creator",
	} //TODO: implement get user later

	var id string

	creatorId := r.URL.Query().Get("creator_id")
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))

	if err != nil {
		s.Server.Logger.Warn("error converting string to int", "service", "HandleGetRecentReads")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))

	if err != nil {
		s.Server.Logger.Warn("error converting string to int", "service", "HandleGetRecentReads")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if user.role == "admin" && creatorId != "" {
		id = creatorId
	} else {
		id = user.id
	}

	book, err := s.Store.GetBooksStats(r.Context(), id, offset, limit)

	if err != nil {
		if err == store.ErrCreatorsBooksNotFound {
			respondWithSuccess(w, http.StatusOK, &models.HandleGetBooksStatsResponse{Books: []models.HandleGetBooksResponseBook{}})
			return
		}

		s.Server.Logger.Error(err.Error(), "service", "HandleGetBooksStats")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	var booksResponse []models.HandleGetBooksResponseBook

	for _, b := range *book {
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

func handleGetBooksHelper(w http.ResponseWriter, books *[]models.Book) {
	var booksResponse []models.HandleGetBooksBooks

	for _, b := range *books {
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
// @Summary Get Books
// @Description Get books by genre, language, both or get all books
// @Produce json
// @Tags books
// @Param genre query string false "book genres"
// @Param language query string false "book language"
// @Param offset query string false "offset number"
// @Param limit query string false "limit number"
// @Success 200 {object} models.HandleGetBooksResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /books [get]
func (s *Server) HandleGetBooks(w http.ResponseWriter, r *http.Request) {
	genre := r.URL.Query()["genre"]
	language := r.URL.Query()["language"]
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))

	if err != nil {
		s.Server.Logger.Warn("error converting string to int", "service", "HandleGetRecentReads")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))

	if err != nil {
		s.Server.Logger.Warn("error converting string to int", "service", "HandleGetRecentReads")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if len(genre) > 0 && len(language) < 1 {
		books, err := s.Store.GetBooksByGenre(r.Context(), genre, offset, limit)

		if err != nil {
			if err == store.ErrNoBooksUnderThisGenre {
				respondWithSuccess(w, http.StatusOK, &models.HandleGetBooksResponse{Books: []models.HandleGetBooksBooks{}})
				return
			}
			s.Server.Logger.Error(err.Error(), "service", "HandleGetBooks")
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}

		handleGetBooksHelper(w, books)
		return
	}

	if len(genre) < 1 && len(language) > 0 {
		books, err := s.Store.GetBooksByLanguage(r.Context(), language, offset, limit)

		if err != nil {
			if err == store.ErrNoBooksUnderThisLanguage {
				respondWithSuccess(w, http.StatusOK, &models.HandleGetBooksResponse{Books: []models.HandleGetBooksBooks{}})
				return
			}
			s.Server.Logger.Error(err.Error(), "service", "HandleGetBooks")
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}

		handleGetBooksHelper(w, books)
		return
	}

	if len(genre) > 0 && len(language) > 0 {
		books, err := s.Store.GetBooksByGenreAndLanguage(r.Context(), genre, language, offset, limit)

		if err != nil {
			if err == store.ErrNoBooksUnderThisGenreOrLanguage {
				respondWithSuccess(w, http.StatusOK, &models.HandleGetBooksResponse{Books: []models.HandleGetBooksBooks{}})
				return
			}
			s.Server.Logger.Error(err.Error(), "service", "HandleGetBooks")
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}

		handleGetBooksHelper(w, books)
		return
	}

	books, err := s.Store.GetAllBooks(r.Context(), offset, limit)
	if err != nil {
		s.Server.Logger.Error(err.Error(), "service", "HandleGetBooks")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	handleGetBooksHelper(w, books)
}

// HandleGetBook godoc
// @Summary Get book
// @Description Get book by id
// @Tags books
// @Produce json
// @Param bookId path string true "book id"
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Success 200 {object} models.HandleGetBookResponse
// @Router /books/{bookId} [get]
func (s *Server) HandleGetBook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "bookId")

	book, err := s.Store.GetBook(r.Context(), id)

	if err != nil {
		if err == store.ErrBookNotFound {
			s.Server.Logger.Warn(err.Error(), "service", "HandleGetBook")
			respondWithError(w, http.StatusNotFound, err)
			return

		}
		s.Server.Logger.Error(err.Error(), "service", "HandleGetBook")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	var chaptersForPreview []models.ChaptersBookPreview

	for _, chapters := range book.Chapters {
		chaptersForPreview = append(chaptersForPreview, models.ChaptersBookPreview{
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
// @Summary Delete book
// @Description Delete book by id
// @Tags books
// @Produce json
// @Param bookId path string true "book id"
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Success 204
// @Router /books/{bookId} [delete]
func (s *Server) HandleDeleteBook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "bookId")

	err := s.Store.DeleteBook(r.Context(), id)

	if err != nil {
		s.Server.Logger.Warn(err.Error(), "service", "HandleDeleteBook")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithSuccess(w, http.StatusNoContent, nil)
}

// HandleEditBook Godoc
// @Summary Edit book details
// @Description Edit book name, image, description, genre or release schedule
// @Tags books
// @Accept multipart/form-data
// @Param bookId path string true "Book Id"
// @Param name formData string false "Book name"
// @Param description formData string false "Book Description"
// @Param image formData string false "Book Image"
// @Param genres formData []string false "Book Genres"
// @Param release_schedule_day formData []string false "Release schedule days"
// @Param release_schedule_chapter formData []string false "Release schedule chapters"
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 413 {object} models.ErrorResponse
// @Faiure 500 {object} models.ErrorResponse
// @Success 204
// @Router /books/{bookId} [patch]
func (s *Server) HandleEditBook(w http.ResponseWriter, r *http.Request) {
	bookId := chi.URLParam(r, "bookId")
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)

	if err := r.ParseMultipartForm(8 << 20); err != nil {
		s.Server.Logger.Warn(fmt.Sprintf("error parsing form: %v", err), "service", "HandleEditBook")
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
		s.Server.Logger.Warn("chapter length and days length must be the same", "service", "HandleEditBook")
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
			s.Server.Logger.Error(fmt.Sprintf("error reading bytes: %v", err), "service", "HandleEditBook")
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error reading bytes: %v", err))
			return
		}

		if len(fileData) > 3<<20 {
			s.Server.Logger.Error("book cover too large", "service", "HandleEditBook")
			respondWithError(w, http.StatusRequestEntityTooLarge, fmt.Errorf("book cover too large"))
			return
		}

		if contentType := http.DetectContentType(fileData); !strings.HasPrefix(contentType, "image/") {
			s.Server.Logger.Warn("invalid file type", "service", "HandleEditBook")
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("invalid file type"))
			return
		}

		url, err = s.Server.ObjectStore.UploadFile(r.Context(), bytes.NewReader(fileData), fmt.Sprintf("%s_%s", bookId, header.Filename))

		if err != nil {
			s.Server.Logger.Error(err.Error(), "service", "HandleEditBook")
			respondWithError(w, http.StatusBadRequest, err)
			return
		}
	}

	if err != nil && err != http.ErrMissingFile {
		s.Server.Logger.Error(fmt.Sprintf("error uploading image: %v", err), "service", "HandleEditBook")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error uploading image: %v", err))
		return
	}

	params := &models.HandleEditBookParam{
		Id:          bookId,
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Genres:      genres,
		Image:       url,
	}

	if len(days) > 0 && len(chapters) > 0 {
		for i := range days {
			ch, err := strconv.Atoi(chapters[i])

			if err != nil {
				s.Server.Logger.Warn("error converting type string to int", "service", "HandleEditBook")
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

	if err := s.Store.EditBook(r.Context(), params); err != nil {
		if err == store.ErrShouldAtLeasePassOneFieldToUpdate {
			s.Server.Logger.Warn(err.Error(), "service", "HandleEditBook")
			respondWithError(w, http.StatusBadRequest, err)
			return
		}
		s.Server.Logger.Error(err.Error(), "service", "HandleEditBook")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithSuccess(w, http.StatusNoContent, nil)
}

// HandleApproveBook godoc
// @Summary Approve book
// @Description Approve book by id
// @Tags books
// @Produce json
// @Param bookId path string true "Book ID"
// @Param param body models.ApproveBookParam true "Approve book body"
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Success 204
// @Router /books/{bookId}/approval [patch]
func (s *Server) HandleApproveBook(w http.ResponseWriter, r *http.Request) {
	bookId := chi.URLParam(r, "bookId")
	param := models.ApproveBookParam{}

	if err := json.NewDecoder(r.Body).Decode(&param); err != nil {
		s.Server.Logger.Error(fmt.Sprintf("error decoding json: %v", err), "service", "HandleApproveBook")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("error decoding json: %v", err))
		return
	}

	if err := shared.Validate.Struct(&param); err != nil {
		s.Server.Logger.Warn(fmt.Sprintf("error validating fields: %v", err), "service", "HandleApproveBook")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("error validating fields: %v", err))
		return
	}

	if err := s.Store.ApproveBook(r.Context(), bookId, param.Approve); err != nil {
		s.Server.Logger.Error(err.Error(), "service", "HandleApproveBook")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithSuccess(w, http.StatusNoContent, nil)
}

// HandleMarkBookAsComplete godoc
// @Summary Mark Book As Complete
// @Description Mark book as complete using id
// @Tags books
// @Produce json
// @Param bookId path string true "Book ID"
// @Param param body models.MarkAsCompleteParam true "Mark book as complete body"
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Success 204
// @Router /books/{bookId}/complete [patch]
func (s *Server) HandleMarkBookAsComplete(w http.ResponseWriter, r *http.Request) {
	bookId := chi.URLParam(r, "bookId")
	param := models.MarkAsCompleteParam{}

	if err := json.NewDecoder(r.Body).Decode(&param); err != nil {
		s.Server.Logger.Error(fmt.Sprintf("error decoding json: %v", err), "service", "HandleMarkBookAsComplete")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("error decoding json: %v", err))
		return
	}

	if err := shared.Validate.Struct(&param); err != nil {
		s.Server.Logger.Warn(fmt.Sprintf("error validating fields: %v", err), "service", "HandleMarkBookAsComplete")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("error validating fields: %v", err))
		return
	}

	if err := s.Store.MarkBookAsComplete(r.Context(), bookId, param.Completed); err != nil {
		s.Server.Logger.Error(err.Error(), "service", "HandleMarkBookAsComplete")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithSuccess(w, http.StatusNoContent, nil)
}

// HandleGetRecentReads godoc
// @Summary Get recent reads
// @Description Get user's recent reads
// @Produce json
// @Tags books
// @Param offset query string true "offset"
// @Param limit query string true "limit"
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Success 200 {object} models.HandleGetRecentReadsResponse
// @Router /books/recents [get]
func (s *Server) HandleGetRecentReads(w http.ResponseWriter, r *http.Request) {
	user := struct {
		id string
	}{
		id: "172122bf-e310-42b9-a69f-7382c0d4a74b",
	} //TODO: implement get user later

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))

	if err != nil {
		s.Server.Logger.Warn("error converting string to int", "service", "HandleGetRecentReads")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))

	if err != nil {
		s.Server.Logger.Warn("error converting string to int", "service", "HandleGetRecentReads")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	books, err := s.Store.GetRecentReads(r.Context(), user.id, offset, limit)

	if err != nil {
		if err == store.ErrNoBooksInRecents {
			s.Server.Logger.Error(err.Error(), "service", "HandleGetRecentReads")
			respondWithSuccess(w, http.StatusOK, &models.HandleGetRecentReadsResponse{Books: []models.RecentReadsResponseBooks{}})
			return
		}

		s.Server.Logger.Error(err.Error(), "service", "HandleGetRecentReads")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	var recentBooks []models.RecentReadsResponseBooks

	for _, b := range *books {
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

func (s *Server) HandleGetNewlyUpdated(w http.ResponseWriter, r *http.Request) {
}
