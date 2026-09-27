package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/MatusTabi/gitmsg/internal/app"
	gitclient "github.com/MatusTabi/gitmsg/internal/git"
	"github.com/MatusTabi/gitmsg/internal/provider/codex"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	os.Exit(run(ctx, os.Stdout, os.Stderr))
}

func run(ctx context.Context, stdout, stderr io.Writer) int {
	return runWith(ctx, stdout, stderr, gitclient.NewSource(), codex.New())
}

func runWith(ctx context.Context, stdout, stderr io.Writer, source app.DiffSource, generator app.Generator) int {
	result := app.Run(ctx, source, generator)
	if result.IsSuccess() {
		fmt.Fprintln(stdout, result.Value())
		return 0
	}

	if errors.Is(result.Error(), context.Canceled) {
		return 130
	}
	fmt.Fprintln(stderr, result.Error())
	return 1
}
