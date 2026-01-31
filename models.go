package main

import (
	"database/sql"
	"time"
)

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

type draftChapter struct {
	Title   string `validate:"required"`
	Content string `validate:"required"`
}

type book struct {
	name             string
	description      string
	image            sql.NullString
	release_schedule []releaseSchedule
	opened_last      time.Time
	author_id        string
	views            int
	language         string
	genres           []string
	draft_chapter    draftChapter
	rating           int
	completed        bool
	approved         bool
	created_at       time.Time
	updated_at       time.Time
}
