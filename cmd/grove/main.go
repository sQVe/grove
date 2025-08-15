package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sqve/grove/internal/app"
)

func main() {
	rootCmd := app.NewRootCommand()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		// Add help suggestion for unknown command errors
		if strings.Contains(err.Error(), "unknown command") {
			fmt.Fprintf(os.Stderr, "\nRun 'grove --help' for usage information\n")
		}

		os.Exit(1)
	}
}
