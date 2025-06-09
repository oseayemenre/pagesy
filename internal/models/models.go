package models

import (
	"github.com/google/uuid"
	"time"
)

type Schedule struct {
	Day      string `json:"day" validate:"required"`
	Chapters int    `json:"chapters" validate:"required"`
}

type Chapter struct {
	Title   string `json:"title" validate:"required"`
	Content string `json:"content" validate:"required"`
}

type Book struct {
	Id               string     `json:"id"`
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	Image            string     `json:"image"`
	Author_Id        uuid.UUID  `json:"author_id"`
	Genres           []string   `json:"genres"`
	Chapter_Draft    Chapter    `json:"chapter_draft"`
	Language         string     `json:"language"`
	Release_schedule []Schedule `json:"release_schedule"`
	Created_at       time.Time  `json:"created_at"`
}

type HandleUploadBooksRequest struct {
	Name             string   `validate:"required"`
	Description      string   `validate:"required"`
	Genres           []string `validate:"required,min=1"`
	Release_schedule []Schedule
	Language         string `validate:"required"`
	ChapterDraft     *Chapter
}

type ErrorResponse struct {
	Error string `json:"error"`
}
