//go:build !wails

// Package desktop: the default (cgo-free) build's desktop entry. The
// Wails GUI needs webkit2gtk/WebView2; in this build Run prints build
// instructions instead of crashing, so the same root binary works in
// Docker/scratch/CI.
package desktop

import (
	"fmt"
	"os"
)

// Run prints desktop build instructions (no GUI in this build).
func Run(dataDir string) int {
	fmt.Fprint(os.Stderr, `the desktop GUI is not included in this build.

Build it with the wails tag (Linux needs webkit2gtk-4.1 dev headers):

  wails build
  # or: go build -tags "wails production webkit2_41" -o fieldkit-desktop .

Debian/Ubuntu: apt install libwebkit2gtk-4.1-dev
Arch: pacman -S webkit2gtk-4.1

The CLI and the TUI work without the GUI.
`)
	return 2
}
