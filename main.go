// Package main is the Wails desktop entry point: `wails build` and
// `wails dev` require the main package at the project root. The real
// surface (CLI, TUI, desktop) lives in ./cmd/fieldkit; this root binary
// only exists so the Wails CLI can build and dev the GUI. In the
// default (cgo-free) build the desktop package compiles to a stub that
// prints build instructions, so this root binary remains harmless.
package main

import (
	"os"

	"github.com/jonathanngiroux-star/fieldkit/internal/desktop"
)

func main() {
	os.Exit(desktop.Run(""))
}
