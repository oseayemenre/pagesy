package main

import "database/sql"

type user struct {
	display_name string
	email        string
	password     sql.NullString
	about        sql.NullString
	image        sql.NullString
	roles        []string
}

type releaseSchedule struct {
	Day      string `validate:"required"`
	Chapters int    `validate:"required"`
}

type chapterDraft struct {
	Title   string `validate:"required"`
	Content string `validate:"required"`
}
