package main

import "database/sql"

func checkNullString(str string) *sql.NullString {
	if str == "" {
		return &sql.NullString{}
	}
	return &sql.NullString{String: str, Valid: true}
}
