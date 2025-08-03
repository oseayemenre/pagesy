package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Schedule struct {
	BookId   uuid.UUID `json:"-"`
	Day      string    `json:"day" validate:"required"`
	Chapters int       `json:"chapters" validate:"required"`
}

type Chapter struct {
	Title      string
	Chapter_no int
	Content    string
	Book_Id    uuid.UUID
	Created_at string
}

type ChapterDraft struct {
	Title      string `json:"title" validate:"required"`
	Chapter_no int    `json:"chapter_no,omitempty"`
	Content    string `json:"content" validate:"required"`
	Created_at string `json:"created_at"`
}

type ChaptersBookPreview struct {
	Chapter_no int    `json:"chapter_no"`
	Title      string `json:"title"`
	Created_at string `json:"created_at"`
}

type Book struct {
	Id               uuid.UUID
	Name             string
	Description      string
	Image            sql.NullString
	Views            int
	Rating           int
	ChapterLastRead  int
	TimeLastOpened   time.Time
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
	ChapterDraft     *ChapterDraft
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

type ApproveBookParam struct {
	Approve bool `json:"approve" validate:"required"`
}

type MarkAsCompleteParam struct {
	Completed bool `json:"completed" validate:"required"`
}

type RecentReadsResponseBooks struct {
	Name            string `json:"name"`
	Image           string `json:"image"`
	LastReadChapter int    `json:"last_read_chapter"`
	LastRead        string `json:"last_read"`
}

type HandleGetRecentReadsResponse struct {
	Books []RecentReadsResponseBooks `json:"books"`
}

type User struct {
	Id              uuid.UUID
	Username        string
	Display_name    string
	Email           string
	Password        string
	Image           string
	Role            string
	Privileges      []string
	About           string
	Follower_count  int
	Following_count int
}

type HandleOnboardingParams struct {
	Username     string `validate:"required"`
	Display_name string `validate:"required"`
	Image        string
	About        string
}

type HandleRegisterParams struct {
	Email    string `json:"email" validate:"email,required"`
	Password string `json:"password" validate:"required,min=8"`
}

type HandleRegisterResponse struct {
	Id string `json:"id"`
}

type HandleLoginParams struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password" validate:"required"`
}

type HandleUploadChapterParams struct {
	Title      string `json:"title" validate:"required"`
	Chapter_no int    `json:"chapter_no" validate:"required"`
	Content    string `json:"content" validate:"required"`
}

type HandleUploadChapterResponse struct {
	Id string `json:"id"`
}

type HandleMarkBookForSubscriptionParams struct {
	Subscription bool `json:"subscription" validate:"required"`
}

type HandleBuyCoinsParams struct {
	Price_id string `json:"price_id" validate:"required"`
}

type HandleBuyCoinsResponse struct {
	Url string `json:"url"`
}
