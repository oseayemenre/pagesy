package routes

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/oseayemenre/pagesy/internal/shared"
	"github.com/oseayemenre/pagesy/internal/store"
)

func (s *Server) HandleUploadBooks(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)

	if err := r.ParseMultipartForm(8 << 20); err != nil {
		s.Server.Logger.Warn(fmt.Sprintf("error parsing form: %v", err), "service", "HandleUploadBooks")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error parsing form: %v", err))
		return
	}

	defer r.MultipartForm.RemoveAll()

	book := struct {
		Name             string   `validate:"required"`
		Description      string   `validate:"required"`
		Genres           []string `validate:"required,min=1"`
		Release_schedule []struct {
			Day      string
			Chapters int
		} `validate:"required,min=1"`
		Languages    []string `validate:"required,min=1"`
		ChapterDraft string   `validate:"required"`
	}{
		Name:         r.FormValue("name"),
		Description:  r.FormValue("description"),
		Genres:       r.Form["genre"],
		Languages:    r.Form["language"],
		ChapterDraft: r.FormValue("chapter_draft"),
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

		schedule := struct {
			Day      string
			Chapters int
		}{
			Day:      days[i],
			Chapters: ch,
		}

		book.Release_schedule = append(book.Release_schedule,
			schedule)
	}

	if err := shared.Validate.Struct(&book); err != nil {
		s.Server.Logger.Warn(fmt.Sprintf("validation error: %v", err), "service", "HandleUploadBooks")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("validation error: %v", err))
		return
	}

	file, header, err := r.FormFile("book_cover")

	if err != nil {
		s.Server.Logger.Error(fmt.Sprintf("error uploading image: %v", err), "service", "HandleUploadBooks")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error uploading image: %v", err))
		return
	}

	defer file.Close()

	if header.Size > 3<<20 {
		s.Server.Logger.Error("book cover too large", "service", "HandleUploadBooks")
		respondWithError(w, http.StatusRequestEntityTooLarge, fmt.Errorf("book cover too large"))
		return
	}

	fileData, err := io.ReadAll(file)

	if err != nil {
		s.Server.Logger.Error(fmt.Sprintf("error reading bytes: %v", err), "service", "HandleUploadBooks")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error reading bytes: %v", err))
		return
	}

	if contentType := http.DetectContentType(fileData); !strings.HasPrefix(contentType, "image/") {
		s.Server.Logger.Warn("invalid file type", "service", "HandleUploadBooks")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("invalid file type"))
		return
	}

	url, err := s.Server.ObjectStore.UploadFile(r.Context(), "", "")

	if err != nil {
		s.Server.Logger.Error(err.Error(), "service", "HandleUploadBooks")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	schedules := make([]store.Schedule, len(book.Release_schedule))
	for i, rs := range book.Release_schedule {
		schedules[i] = store.Schedule{
			Day:      rs.Day,
			Chapters: rs.Chapters,
		}
	}

	if err := s.Server.Store.UploadBook(r.Context(), &store.Book{
		Name:             book.Name,
		Description:      book.Description,
		Image:            url,
		Author_Id:        "", //TODO: add an actual author id here
		Genres:           book.Genres,
		Chapter_Draft:    book.ChapterDraft,
		Release_schedule: schedules,
	}); err != nil {
		s.Server.Logger.Error(err.Error(), "service", "HandleUploadBooks")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}
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
