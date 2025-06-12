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
	Id               uuid.UUID
	Name             string
	Description      string
	Image            string
	Views            int
	Author_Id        uuid.UUID
	Completed        bool
	Approved         bool
	Genres           []string
	No_Of_Chapters   int
	Chapter_Draft    Chapter
	Language         string
	Release_schedule []Schedule
	Created_at       time.Time
	Updated_at       time.Time
}

type HandleUploadBooksRequest struct {
	Name             string `validate:"required"`
	Description      string `validate:"required"`
	Genres           string `validate:"required"`
	Release_schedule []Schedule
	Language         string `validate:"required"`
	ChapterDraft     *Chapter
}

type HandleUploadBooksResponse struct {
	Message string `json:"message"`
}

type HandleGetBooksResponseBook struct {
	Id               uuid.UUID  `json:"id"`
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	Image            string     `json:"image"`
	Views            int        `json:"views"`
	Completed        bool       `json:"completed"`
	Approved         bool       `json:"approved"`
	Genres           []string   `json:"genres"`
	No_Of_Chapters   int        `json:"no_of_chapters"`
	Language         string     `json:"language"`
	Release_schedule []Schedule `json:"release_schedule"`
	Created_at       time.Time  `json:"created_at"`
	Updated_at       time.Time  `json:"updated_at"`
}

type HandleGetBooksStatsResponse struct {
	Books []HandleGetBooksResponseBook `json:"books"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
