package api

import (
	"fmt"
	"github.com/oseayemenre/pagesy/internal/models"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"net/http"
)

// HandleBuyCoins godoc
//
//	@Summary		Buy coins
//	@Description	Buy different coin plans with price id
//	@Tags			coins
//	@Param			price_id	body		models.HandleBuyCoinsParams	true	"price_id"
//	@Failure		400			{object}	models.ErrorResponse
//	@Failure		404			{object}	models.ErrorResponse
//	@Failure		500			{object}	models.ErrorResponse
//	@Success		200			{object}	models.HandleBuyCoinsResponse
//	@Router			/coins [post]
func (a *Api) HandleBuyCoins(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*models.User)
	var params models.HandleBuyCoinsParams
	if err := decodeJson(r, &params); err != nil {
		a.logger.Warn(err.Error(), "service", "HandleBuyCoins")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if err := validate.Struct(&params); err != nil {
		a.logger.Warn(fmt.Sprintf("error validating fields: %v", err), "service", "HandleBuyCoins")
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("error validating fields: %v", err))
		return
	}

	stripe.Key = a.config.Stripe_secret

	price_id_map := map[string]string{
		"price_1Rs4vh3E7Wcf3t7SqG7P3KPi": "50 coins",
		"price_1Rs4v73E7Wcf3t7SS7NhVRKY": "20 coins",
	} //TODO: change this in prod

	if _, ok := price_id_map[params.Price_id]; !ok {
		a.logger.Warn("price id not found", "service", "HandleBuyCoins")
		respondWithError(w, http.StatusNotFound, fmt.Errorf("price id not found"))
		return
	}

	checkout_params := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    &params.Price_id,
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(a.config.Host + "/healthz"), //TODO: put a better url here when there's a frontend
		CancelURL:  stripe.String(a.config.Host + "/healthz"), //TODO: put a better url here when there's a frontend
		Metadata: map[string]string{
			"user_id":   user.Id.String(),
			"coin_plan": price_id_map[params.Price_id],
		},
	}

	s, err := session.New(checkout_params)

	if err != nil {
		a.logger.Error(fmt.Sprintf("error creating stripe session: %v", err), "service", "HandleBuyCoins")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error creating stripe session: %v", err))
		return
	}

	respondWithSuccess(w, http.StatusOK, &models.HandleBuyCoinsResponse{Url: s.URL})
}
