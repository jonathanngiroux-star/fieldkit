package main

// clipboard.go: copy-to-clipboard for the donate addresses (TUI/CLI
// strategy, tried in order):
//
//  1. OSC-52 escape — most modern terminals (kitty, foot,
//     gnome-terminal, tmux passthrough) with zero dependencies; written
//     to /dev/tty so it works when stdout is piped.
//  2. wl-copy (Wayland), xclip / xsel (X11) — terminals that ignore
//     OSC-52.
//
// The GUI uses Wails' runtime clipboard instead (bound method).

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// copyAddress copies addr to the system clipboard for the TUI.
func copyAddress(addr string) error {
	if tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0); err == nil {
		_, writeErr := tty.WriteString(osc52For(addr))
		tty.Close()
		if writeErr == nil {
			return nil
		}
	}
	return copyToClipboardTools(addr)
}

// osc52For builds the terminal clipboard escape sequence:
// ESC ] 52 ; c ; <base64> BEL
func osc52For(text string) string {
	return "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\x07"
}

// copyToClipboardTools pipes text into the first installed tool.
func copyToClipboardTools(text string) error {
	type tool struct {
		name string
		args []string
	}
	tools := []tool{
		{"wl-copy", nil},
		{"xclip", []string{"-selection", "clipboard"}},
		{"xsel", []string{"--clipboard", "--input"}},
	}
	for _, tl := range tools {
		path, err := exec.LookPath(tl.name)
		if err != nil {
			continue
		}
		cmd := exec.Command(path, tl.args...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s failed: %w", tl.name, err)
		}
		return nil
	}
	return fmt.Errorf("no clipboard tool available — install wl-copy (Wayland) or xclip (X11), or copy the address manually")
}
