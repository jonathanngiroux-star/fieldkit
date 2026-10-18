package main

// launcher.go: what `fieldkit tui` does. The TUI needs an interactive
// terminal; pipes/CI never hang — they get a hint and exit.

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/jonathanngiroux-star/fieldkit/internal/store"
)

// isTerminal reports whether f is an interactive terminal.
var isTerminal = func(f *os.File) bool {
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

// launchTUI runs the terminal UI when interactive; otherwise prints a
// hint (tests, pipes, CI — never a hang).
func launchTUI(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dataDir := fs.String("data-dir", os.Getenv("FIELDKIT_DATA_DIR"), "data directory (SQLite)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *dataDir == "" {
		*dataDir = defaultDataDir()
	}
	if !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
		fmt.Fprintln(stderr, "fieldkit: the TUI needs an interactive terminal.")
		fmt.Fprintln(stderr, "headless? try: fieldkit surveys list, fieldkit migrate, fieldkit logic, fieldkit serve, fieldkit export csv")
		fmt.Fprintln(stderr)
		usage(stderr)
		return 2
	}
	st, err := store.Open(*dataDir)
	if err != nil {
		fmt.Fprintf(stderr, "tui: %v\n", err)
		return 1
	}
	defer st.Close()
	return runTUI(newAppModel(st))
}
