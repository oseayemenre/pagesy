package main

import (
	"fmt"
	"os"
)

// @title Pagesy
// @version 1.0
// @host localhost:8000
// @BasePath /api/v1
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v", err)
		os.Exit(1)
	}
}
