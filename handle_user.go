package main

import (
	"net/http"
)

// handleGetProfile godoc
//
//	@Summary		Get current user profile
//	@Description	Get current user profile
//	@Tags			users
//	@Produce		json
//	@Failure		404	{object}	errorResponse
//	@Success		200	{object}	main.handleGetProfile.response
//	@Router			/users/me [get]
func (s *server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Email        string   `json:"email"`
		Display_name string   `json:"display_name"`
		Image        *string  `json:"image"`
		About        *string  `json:"about"`
		Roles        []string `json:"roles"`
	}

	id := r.Context().Value("user").(string)

	user, err := s.getUser(r.Context(), id)
	if err != nil {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	var about *string
	if user.about.Valid {
		about = &user.about.String
	}

	var image *string
	if user.image.Valid {
		image = &user.image.String
	}

	encode(w, http.StatusOK, &response{Email: user.email, Display_name: user.displayName, Image: image, About: about, Roles: user.roles})
}
