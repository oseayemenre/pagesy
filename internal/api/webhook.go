package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

func (a *Api) HandleWebHook(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
	}

	event := stripe.Event{}

	if err := json.Unmarshal(payload, &event); err != nil {
		a.logger.Error(fmt.Sprintf("error unmarshalling payload: %v", err), "service", "HandleWebHook")
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error unmarshalling payload: %v", err))
		return
	}

	endpointSecret := a.config.Stripe_webhook_secret

	signatureHeader := r.Header.Get("Stripe-Signature")
	event, err = webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
	if err != nil {
	}

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			a.logger.Error(fmt.Sprintf("error unmarshalling event data: %v", err), "service", "HandleWebHook")
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("error unmarshalling event data: %v", err))
			return
		}

		user_id := session.Metadata["user_id"]
		coin_plan := session.Metadata["coin_plan"]

		switch coin_plan {
		case "50 coins":
			if err := a.store.UpdateUserCoinCount(r.Context(), user_id, 50); err != nil {
				a.logger.Error(err.Error(), "service", "HandleWebHook")
				respondWithError(w, http.StatusInternalServerError, err)
				return
			}

		case "20 coins":
			if err := a.store.UpdateUserCoinCount(r.Context(), user_id, 20); err != nil {
				a.logger.Error(err.Error(), "service", "HandleWebHook")
				respondWithError(w, http.StatusInternalServerError, err)
				return
			}
		}
	}
}
