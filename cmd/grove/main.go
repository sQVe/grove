package main

import (
	"fmt"
	"os"

	"github.com/sqve/grove/internal/app"
)

func main() {
	rootCmd := app.NewRootCommand()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
