package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// handleFollowUser godoc
//
//	@Summary		Follow user
//	@Description	Follow user
//	@Tags			followers
//	@Param			userID	path		string	true	"user id"
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Success		204
//	@Router			/users/{userID}/follow [post]
func (s *server) handleFollowUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	followerID := r.Context().Value("user").(string)

	if userID == followerID {
		encode(w, http.StatusBadRequest, &errorResponse{Error: "user cannot follow themselves"})
		return
	}

	displayName, err := s.followUser(r.Context(), r.Context().Value("user").(string), chi.URLParam(r, "userID"))
	if errors.Is(err, errUserNotFound) {
		encode(w, http.StatusNotFound, &errorResponse{Error: err.Error()})
		return
	}
	if err != nil {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	if client, ok := s.hub.regular[userID]; ok {
		client.send <- fmt.Appendf([]byte{}, "%v followed you", displayName)
	}

	encode(w, http.StatusNoContent, nil)
}

// handleUnfollowUser godoc
//
//	@Summary		Unfollow user
//	@Description	Unfollow user
//	@Tags			followers
//	@Param			userID	path		string	true	"user id"
//	@Failure		400		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Success		204
//	@Router			/users/{userID}/unfollow [DELETE]
func (s *server) handleUnfollowUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	followerID := r.Context().Value("user").(string)

	if userID == followerID {
		encode(w, http.StatusBadRequest, &errorResponse{Error: "user cannot unfollow themselves"})
		return
	}

	if err := s.unfollowUser(r.Context(), r.Context().Value("user").(string), chi.URLParam(r, "userID")); err != nil {
		if errors.Is(err, errUserNotFound) {
			encode(w, http.StatusNotFound, &errorResponse{Error: err.Error()})
			return
		}
		if errors.Is(err, errUserNotFollowed) {
			encode(w, http.StatusBadRequest, &errorResponse{Error: err.Error()})
			return
		}
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	encode(w, http.StatusNoContent, nil)
}

// handleGetUserFollowers godoc
//
//	@Summary		Get user followers
//	@Description	Get user followers
//	@Tags			followers
//	@Param			userID	path		string	true	"user id"
//	@Failure		404		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Success		200		{object}	main.handleGetUserFollowers.response
//	@Router			/users/{userID}/followers [get]
func (s *server) handleGetUserFollowers(w http.ResponseWriter, r *http.Request) {
	type follower struct {
		DisplayName string  `json:"displayName"`
		Image       *string `json:"image"`
		About       *string `json:"about"`
	}

	type response struct {
		Followers []follower `json:"followers"`
	}

	users, err := s.getUserFollowers(r.Context(), chi.URLParam(r, "userID"))
	if errors.Is(err, errUserNotFound) {
		encode(w, http.StatusNotFound, &errorResponse{Error: err.Error()})
		return
	}
	if err != nil {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	var followers []follower

	for _, u := range users {
		var image *string
		if u.image.Valid {
			image = &u.image.String
		}
		var about *string
		if u.about.Valid {
			about = &u.about.String
		}
		followers = append(followers, follower{DisplayName: u.displayName, Image: image, About: about})
	}

	encode(w, http.StatusOK, &response{Followers: followers})
}

// handleGetUserFollowing godoc
//
//	@Summary		Get user following
//	@Description	Get user following
//	@Tags			followers
//	@Param			userID	path		string	true	"user id"
//	@Failure		404		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Success		200		{object}	main.handleGetUserFollowing.response
//	@Router			/users/{userID}/following [get]
func (s *server) handleGetUserFollowing(w http.ResponseWriter, r *http.Request) {
	type following struct {
		DisplayName string  `json:"displayName"`
		Image       *string `json:"image"`
		About       *string `json:"about"`
	}

	type response struct {
		Following []following `json:"following"`
	}

	users, err := s.getUserFollowing(r.Context(), chi.URLParam(r, "userID"))
	if errors.Is(err, errUserNotFound) {
		encode(w, http.StatusNotFound, &errorResponse{Error: err.Error()})
		return
	}
	if err != nil {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	var f []following

	for _, u := range users {
		var image *string
		if u.image.Valid {
			image = &u.image.String
		}
		var about *string
		if u.about.Valid {
			about = &u.about.String
		}
		f = append(f, following{DisplayName: u.displayName, Image: image, About: about})
	}

	encode(w, http.StatusOK, &response{Following: f})
}
