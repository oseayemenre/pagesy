package main

import (
	"testing"
)

func TestCheckNullString(t *testing.T) {
	tests := []struct {
		name   string
		val    string
		expect bool
	}{
		{
			name:   "valid should be false",
			val:    "",
			expect: false,
		},
		{
			name:   "valid should be true",
			val:    "valid",
			expect: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val := checkNullString(tc.val)
			if val.Valid != tc.expect {
				t.Fatalf("expected %v, got %v", tc.expect, val.Valid)
			}
		})
	}
}
