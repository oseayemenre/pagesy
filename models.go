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
