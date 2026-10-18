//go:build !wails

package main

// desktop_default.go: the default (cgo-free) build's `fieldkit
// desktop` prints build instructions instead of crashing. Keeps the
// Docker/scratch/CI binary cgo-free while the tagged source stays
// honest about what it needs.

import (
	"fmt"
	"io"
)

func launchDesktop(args []string, stdout, stderr io.Writer) int {
	fmt.Fprint(stderr, `the desktop GUI is not included in this build.

Build it with the wails tag (Linux needs webkit2gtk-4.1 dev headers):

  go build -tags "wails production webkit2_41" -o fieldkit-desktop ./cmd/fieldkit
  ./fieldkit-desktop desktop

Debian/Ubuntu: apt install libwebkit2gtk-4.1-dev
Arch: pacman -S webkit2gtk-4.1

The CLI and the TUI in this binary work without the GUI.
`)
	return 2
}
