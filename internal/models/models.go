package models

import (
	"github.com/google/uuid"
	"time"
)

type Schedule struct {
	BookId   uuid.UUID `json:"-"`
	Day      string    `json:"day" validate:"required"`
	Chapters int       `json:"chapters" validate:"required"`
}

type Chapter struct {
	Title   string `json:"title" validate:"required"`
	Content string `json:"content" validate:"required"`
}

type Book struct {
	Id               *uuid.UUID `json:"id"`
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	Image            string     `json:"image"`
	Views            string     `json:"views"`
	Author_Id        *uuid.UUID `json:"author_id,omitempty"`
	Completed        *bool      `json:"completed,omitempty"`
	Approved         *bool      `json:"approved,omitempty"`
	Genres           []string   `json:"genres"`
	No_Of_Chapters   int        `json:"no_of_chapters,omitempty"`
	Chapter_Draft    *Chapter   `json:"chapter_draft,omitempty"`
	Language         string     `json:"language"`
	Release_schedule []Schedule `json:"release_schedule"`
	Created_at       time.Time  `json:"created_at"`
	Updated_at       time.Time  `json:"updated_at"`
}

type HandleUploadBooksRequest struct {
	Name             string   `validate:"required"`
	Description      string   `validate:"required"`
	Genres           []string `validate:"required,min=1"`
	Release_schedule []Schedule
	Language         string `validate:"required"`
	ChapterDraft     *Chapter
}

type HandleGetBooksStatsResponse struct {
	Books []Book `json:"books"`
}
type ErrorResponse struct {
	Error string `json:"error"`
}
