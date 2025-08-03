package main

import (
	"fmt"
	"github.com/oseayemenre/pagesy/cmd"
	"os"
)

// @title		Pagesy
// @version	1.0
// @host		localhost:8080
// @BasePath	/api/v1
func main() {
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v", err)
		os.Exit(1)
	}
}
