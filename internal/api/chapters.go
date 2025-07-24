package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/oseayemenre/pagesy/internal/models"
	"github.com/oseayemenre/pagesy/internal/store"
)

// HandleUploadChapter godoc
//
//	@Summary		Upload chapter
//	@Description	Upload chapter using title, chapter number, content and book id
//	@Tags			chapters
//	@Accept			json
//	@Produce		json
//	@Param			bookId	path		string								true	"book id"
//	@Param			chapter	body		models.HandleUploadChapterParams	true	"chapter"
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		404		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Success		201		{object}	models.HandleUploadChapterResponse
//	@Router			/books/{bookId}/chapters [post]
func (a *Api) HandleUploadChapter(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*models.User)
	book_id := chi.URLParam(r, "bookId")
	var params models.HandleUploadChapterParams

	if err := decodeJson(r, &params); err != nil {
		a.logger.Warn(err.Error(), "service", "HandleUploadChapter")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	if err := validate.Struct(&params); err != nil {
		a.logger.Warn(fmt.Sprintf("error validating fields: %v", err), "service", "HandleUploadChapter")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("error validating fields: %v", err))
		return
	}

	uuid_book_id, err := uuid.Parse(book_id)

	if err != nil {
		a.logger.Warn("book id is not a valid uuid", "service", "HandleUploadChapter")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("book id is not a valid uuid"))
		return
	}

	id, err := a.store.UploadChapter(r.Context(), user.Id.String(), &models.Chapter{
		Title:      params.Title,
		Chapter_no: params.Chapter_no,
		Content:    params.Content,
		Book_Id:    uuid_book_id,
	})

	if err != nil {
		if err == store.ErrBookNotFound {
			a.logger.Warn(err.Error(), "service", "HandleUploadChapter")
			respondWithError(w, http.StatusNotFound, err)
			return
		}

		a.logger.Error(err.Error(), "service", "HandleUploadChapter")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithSuccess(w, http.StatusCreated, &models.HandleUploadChapterResponse{Id: id.String()})
}

func (a *Api) HandleGetChapter(w http.ResponseWriter, r *http.Request) {}

func (a *Api) HandleDeleteChapter(w http.ResponseWriter, r *http.Request) {}

func (a *Api) HandleGetPage(w http.ResponseWriter, r *http.Request) {}
