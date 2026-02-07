package main

import (
	"database/sql"
	"time"
)

type user struct {
	displayName string
	email       string
	password    sql.NullString
	about       sql.NullString
	image       sql.NullString
	roles       []string
}

type releaseSchedule struct {
	BookID   string
	Day      string
	Chapters int
}

type draftChapter struct {
	Title   string `validate:"required"`
	Content string `validate:"required"`
}

type book struct {
	id              string
	name            string
	description     string
	image           sql.NullString
	releaseSchedule []releaseSchedule
	chapterCount    int
	openedLast      time.Time
	authorID        string
	views           int
	language        string
	genres          []string
	draftChapter    draftChapter
	rating          float32
	completed       bool
	approved        bool
	createdAt       time.Time
	updatedAt       time.Time
}
