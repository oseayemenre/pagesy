package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/oseayemenre/pagesy/internal/shared"
)

func (s *Server) HandleUploadBooks(w http.ResponseWriter, r *http.Request) {
	book := &struct {
		Name        string   `json:"name" validate:"required"`
		Description string   `json:"description" validate:"required"`
		Genre       []string `json:"genre" validate:"required,min=1"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		s.Server.Logger.Warn("error decoding json", "service", "HandleUploadBooks")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "error decoding json"})
		return
	}

	if err := shared.Validate.Struct(&book); err != nil {
		s.Server.Logger.Warn(fmt.Sprintf("validation error: %v", err), "service", "HandleUploadBooks")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("validation error: %v", err)})
		return
	}

	if err := r.ParseMultipartForm(3 << 20); err != nil {
		s.Server.Logger.Warn("file is too large", "service", "HandleUploadBooks")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "file is too large"})
		return
	}

	file, _, err := r.FormFile("book_picture")

	if err != nil {
		s.Server.Logger.Error(fmt.Sprintf("error uploading image: %v", err), "service", "HandleUploadBooks")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("error uploading image: %v", err)})
		return
	}

	fileData, err := io.ReadAll(file)

	if err != nil {
		s.Server.Logger.Error(fmt.Sprintf("error reading bytes: %v", err), "service", "HandleUploadBooks")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("error reading bytes: %v", err)})
		return
	}

	if contentType := http.DetectContentType(fileData); strings.HasPrefix(contentType, "image/") {
		s.Server.Logger.Warn("invalid file type", "service", "HandleUploadBooks")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid file type"})
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
