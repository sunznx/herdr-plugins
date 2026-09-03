package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/app"
)

func main() {
	if err := app.NewCommand().ExecuteContext(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "herdr-plugin: %v\n", err)
		os.Exit(1)
	}
}
