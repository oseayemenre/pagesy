package api

import (
	"net/http"
)

// admin create club, deletes club, link generation to join the club
// a person that creates a club would automatically become an admin
// the person that leaves the club and i the one that created the club and current admin must transfer ownership
// accepts members or make it open
// main admin adds moderators and their abilities
// kick member out
// Anyone
func (a *Api) HandleCreateGroup(w http.ResponseWriter, r *http.Request) {}
func (a *Api) HandleDeleteGroup(w http.ResponseWriter, r *http.Request) {}

// Admin
func (a *Api) HandleManageModerator(w http.ResponseWriter, r *http.Request)    {}
func (a *Api) HandleModeratorAbilities(w http.ResponseWriter, r *http.Request) {}
func (a *Api) HandleAddModerator(w http.ResponseWriter, r *http.Request)       {}
func (a *Api) HandleKickOutMembers(w http.ResponseWriter, r *http.Request)     {}
func (a *Api) HandleGenerateInviteLink(w http.ResponseWriter, r *http.Request) {}
func (a *Api) HandleGetInviteLink(w http.ResponseWriter, r *http.Request)      {}

// Members
func (a *Api) HandleMakePost(w http.ResponseWriter, r *http.Request) {}
func (a *Api) HandleEditPost(w http.ResponseWriter, r *http.Request) {}
