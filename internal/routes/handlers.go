package routes

import "net/http"

func (s *Server) HandleUploadBooks(w http.ResponseWriter, r *http.Request) {}

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
