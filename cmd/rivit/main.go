package main

import (
	"fmt"
	"os"

	"github.com/jonsampson/rivit/internal"
)

func main() {
	app, err := internal.NewApp(os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	os.Exit(app.Run(os.Args[1:]))
}
