package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/oseayemenre/pagesy/internal/models"
)

// HandleMarkBookForSubscription godoc
//
//	@Summary		Mark book for subscription
//	@Description	Mark book for subscription
//	@Tags			subscriptions
//	@Param			bookId			path		string										true	"book id"
//	@Param			subscription	body		models.HandleMarkBookForSubscriptionParams	true	"mark book for subscription body"
//	@Failure		400				{object}	models.ErrorResponse
//	@Failure		500				{object}	models.ErrorResponse
//	@Success		204
//	@Router			/books/{bookId}/subscriptions [patch]
func (a *Api) HandleMarkBookForSubscription(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*models.User)
	bookId := chi.URLParam(r, "bookId")
	var params models.HandleMarkBookForSubscriptionParams

	if err := decodeJson(r, &params); err != nil {
		a.logger.Warn(err.Error(), "service", "HandleMarkBookForSubscription")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if err := validate.Struct(&params); err != nil {
		a.logger.Warn(fmt.Sprintf("error validating fields: %v", err), "service", "HandleMarkBookForSubscription")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("error validating fields: %v", err))
		return
	}

	eligible, err := a.store.CheckIfBookIsEligibleForSubscription(r.Context(), bookId)

	if err != nil {
		a.logger.Error(err.Error(), "service", "HandleMarkBookForSubscription")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	if !eligible {
		a.logger.Warn("book isn't eligible for subscription", "service", "HandleMarkBookForSubscription")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("book isn't eligible for subscription"))
		return
	}

	if err := a.store.MarkBookForSubscription(r.Context(), bookId, user.Id.String(), params.Subscription); err != nil {
		a.logger.Error(err.Error(), "service", "HandleMarkBookForSubscription")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithSuccess(w, http.StatusNoContent, nil)
}
