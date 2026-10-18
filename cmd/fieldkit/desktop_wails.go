//go:build wails

package main

// desktop_wails.go: the wails-tagged half of `fieldkit desktop` —
// forwards to internal/desktop.Run (the Wails app). The default build
// (desktop_default.go) prints build instructions instead.

import (
	"flag"
	"io"
	"os"

	"github.com/jonathanngiroux-star/fieldkit/internal/desktop"
)

func launchDesktop(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("desktop", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dataDir := fs.String("data-dir", os.Getenv("FIELDKIT_DATA_DIR"), "data directory (SQLite)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	return desktop.Run(*dataDir)
}
