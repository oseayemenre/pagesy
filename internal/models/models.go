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
	Title      string `json:"title" validate:"required"`
	Content    string `json:"content" validate:"required"`
	Created_at string `json:"created_at"`
}

type ChaptersBookPreview struct {
	Title      string `json:"title"`
	Created_at string `json:"created_at"`
}

type Book struct {
	Id               uuid.UUID
	Name             string
	Description      string
	Image            string
	Views            int
	Rating           int
	Author_name      string
	Author_Id        uuid.UUID
	Completed        bool
	Approved         bool
	Genres           []string
	No_Of_Chapters   int
	Chapter_Draft    Chapter
	Chapters         []Chapter
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
	Id string `json:"id"`
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

type HandleGetBooksBooks struct {
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	Image            string     `json:"image"`
	Views            int        `json:"views"`
	Rating           int        `json:"rating"`
	Genres           []string   `json:"genres"`
	No_Of_Chapters   int        `json:"no_of_chapters"`
	Release_schedule []Schedule `json:"release_schedule"`
}

type HandleGetBooksResponse struct {
	Books []HandleGetBooksBooks `json:"books"`
}

type HandleGetBookResponse struct {
	Name             string                `json:"name"`
	Description      string                `json:"description"`
	Image            string                `json:"image"`
	Views            int                   `json:"views"`
	Rating           int                   `json:"rating"`
	Genres           []string              `json:"genres"`
	No_Of_Chapters   int                   `json:"no_of_chapters"`
	Chapters         []ChaptersBookPreview `json:"chapters"`
	Release_schedule []Schedule            `json:"release_schedule"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type HandleEditBookParam struct {
	Id               string
	Name             string
	Description      string
	Genres           []string
	Image            string
	Release_schedule []Schedule
}
