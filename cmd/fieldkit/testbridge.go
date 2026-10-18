//go:build wails

package main

// testbridge.go: in-app test bridge (the warmline/proofspan pattern —
// see the go-gui-audit-testing skill). When the app is launched with
// FIELDKIT_GUI_TESTBRIDGE=1, a goroutine polls a command file (one
// JSON object per line) and forwards each command into the webview as
// a "test:cmd" event. The frontend interpreter (testbridge.js)
// executes it against the real DOM — real handlers, real bindings —
// and returns the result through the TestResult binding, which appends
// `TEST-RESULT {json}` to a log the external Python driver reads.
//
// Why a bridge: a Wayland webview cannot be driven from outside
// (xdotool never reaches WebKitGTK; no browser driver attaches to the
// wails:// origin). The app tests itself.
//
// Security: the interpreter is a fixed switch — no eval — so the CSP
// holds. The bridge arms ONLY on the env var; production launches
// never poll.

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	wr "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	bridgeDefaultCmdFile    = "/tmp/fieldkit-gui-cmd.jsonl"
	bridgeDefaultResultFile = "/tmp/fieldkit-gui-result.log"
	bridgePollInterval      = 100 * time.Millisecond
)

type bridgeConfig struct {
	enabled    bool
	cmdFile    string
	resultFile string
}

var bridge struct {
	mu     sync.Mutex
	config bridgeConfig
	ctxMu  sync.Mutex
	ctx    context.Context
}

// startTestBridge arms the bridge when FIELDKIT_GUI_TESTBRIDGE=1 and
// starts the command poller. On start it removes the stale command file
// and truncates the result log — stale-command replay is a real bug
// class.
func (a *DesktopApp) startTestBridge() {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	if bridge.config.enabled {
		return
	}
	if os.Getenv("FIELDKIT_GUI_TESTBRIDGE") != "1" {
		return
	}
	bridge.config = bridgeConfig{
		enabled:    true,
		cmdFile:    envDefault("FIELDKIT_GUI_CMD_FILE", bridgeDefaultCmdFile),
		resultFile: envDefault("FIELDKIT_GUI_RESULT_FILE", bridgeDefaultResultFile),
	}
	os.Remove(bridge.config.cmdFile)
	os.Truncate(bridge.config.resultFile, 0)

	bridge.ctxMu.Lock()
	bridge.ctx = a.ctx
	bridge.ctxMu.Unlock()

	go bridgePollLoop()
	fmt.Fprintf(os.Stderr, "testbridge: armed (cmd=%s result=%s)\n",
		bridge.config.cmdFile, bridge.config.resultFile)
}

func bridgePollLoop() {
	for {
		bridge.mu.Lock()
		cfg := bridge.config
		bridge.mu.Unlock()
		if !cfg.enabled {
			return
		}
		processCommands(cfg)
		time.Sleep(bridgePollInterval)
	}
}

// processCommands reads all pending command lines and truncates the
// file (read-all-then-truncate, commands handled in order).
func processCommands(cfg bridgeConfig) {
	f, err := os.Open(cfg.cmdFile)
	if err != nil {
		return
	}
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	f.Close()
	if len(lines) == 0 {
		return
	}
	os.Truncate(cfg.cmdFile, 0)
	for _, line := range lines {
		bridge.ctxMu.Lock()
		ctx := bridge.ctx
		bridge.ctxMu.Unlock()
		if ctx == nil {
			continue
		}
		wr.EventsEmit(ctx, "test:cmd", line)
	}
}

// TestResult is the binding the frontend calls with each command's
// result. It appends `TEST-RESULT {json}` to the result log.
func (a *DesktopApp) TestResult(resultJSON string) {
	bridge.mu.Lock()
	cfg := bridge.config
	bridge.mu.Unlock()
	if !cfg.enabled {
		return
	}
	f, err := os.OpenFile(cfg.resultFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "TEST-RESULT %s\n", strings.TrimSpace(resultJSON))
}

// Ping is the driver's readiness probe.
func (a *DesktopApp) Ping() string { return "pong" }

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
