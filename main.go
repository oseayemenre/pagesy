package main

import (
	"os"

	"fmt"
	"github.com/oseayemenre/pagesy/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v", err)
		os.Exit(1)
	}
}
