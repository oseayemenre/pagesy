package routes

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oseayemenre/pagesy/internal/models"
	"github.com/oseayemenre/pagesy/internal/shared"
)

// HandleUploadBooks godoc
// @Summary Upload a new book
// @Description Uploads a book with metadata, schedule, draft chapter, and book cover image
// @Tags books
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "Book name"
// @Param description formData string true "Book description"
// @Param genre formData []string true "Genres"
// @Param language formData string true "Book language"
// @Param chapter_title formData string true "Draft chapter title"
// @Param chapter_content formData string true "Draft chapter content"
// @Param release_schedule_day formData []string true "Release days (e.g. Monday, Tuesday)"
// @Param release_schedule_chapter formData []int true "Chapters per day (e.g. 1, 2)"
// @Param book_cover formData file false "Book cover image (max 3MB)"
// @Success 201 {object} models.HandleUploadBooksRequest
// @Failure 400 {object} models.ErrorResponse
// @Failure 413 {object} models.ErrorResponse
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
		Genres:      r.Form["genre"],
		Language:    r.FormValue("language"),
		ChapterDraft: &models.Chapter{
			Title:   r.FormValue("chapter_title"),
			Content: r.FormValue("chapter_content"),
		},
	}

	days := r.Form["release_schedule_day"]
	chapters := r.Form["release_schedule_chapter"]

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

		url, err = s.Server.ObjectStore.UploadFile(r.Context(), bytes.NewReader(fileData), fmt.Sprintf("%s_%v", header.Filename, time.Now().Unix()))

		if err != nil {
			s.Server.Logger.Error(err.Error(), "service", "HandleUploadBooks")
			respondWithError(w, http.StatusBadRequest, err)
			return
		}
	}

	if err != nil && err != http.ErrMissingFile {
		s.Server.Logger.Error(fmt.Sprintf("error uploading image: %v", err), "service", "HandleUploadBooks")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error uploading image: %v", err))
		return
	}

	schedules := make([]models.Schedule, len(params.Release_schedule))
	for i, rs := range params.Release_schedule {
		schedules[i] = models.Schedule{
			Day:      rs.Day,
			Chapters: rs.Chapters,
		}
	}

	//Dummy id here. Would handle this properly later
	authorId, err := uuid.Parse("dc5e215a-afd4-4f70-aa80-3e360fa1d9e4")

	if err != nil {
		s.Server.Logger.Error(fmt.Sprintf("error parsing uuid: %v", err), "service", "HandleUploadBooks")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error parsing uuid: %v", err))
		return
	}

	if err := s.Server.Store.UploadBook(r.Context(), &models.Book{
		Name:        params.Name,
		Description: params.Description,
		Image:       url,
		Author_Id:   authorId,
		Genres:      params.Genres,
		Chapter_Draft: models.Chapter{
			Title:   params.ChapterDraft.Title,
			Content: params.ChapterDraft.Content,
		},
		Language:         params.Language,
		Release_schedule: schedules,
	}); err != nil {
		s.Server.Logger.Error(err.Error(), "service", "HandleUploadBooks")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	respondWithSuccess(w, http.StatusCreated, map[string]string{"message": "new book created"})
}

func (s *Server) HandleGetBooks(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleGetBook(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleDeleteBook(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleEditBook(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleApproveBook(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleMarkBookAsComplete(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleUploadChapters(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleGetChapter(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleDeleteChapter(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleGetPage(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleGetRecents(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleGetNewlyUpdated(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleGetRecommended(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandlePostComments(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleGetComments(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleReplyComments(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleDeleteComments(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleEditComments(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleGetBooksInLibrary(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleAddBookInLibrary(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleDeleteBookFromLibrary(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleCoinse(w http.ResponseWriter, r *http.Request) {}

func (s *Server) HandleBanUser(w http.ResponseWriter, r *http.Request) {}
