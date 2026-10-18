package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jonathanngiroux-star/fieldkit/internal/plugin"
)

// cmdPlugin validates plugin manifests and runs contract checks.
//
//	fieldkit plugin validate <dir|manifest.json>
func cmdPlugin(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: fieldkit plugin validate <dir-or-manifest>")
		return 2
	}
	switch args[0] {
	case "validate":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "usage: fieldkit plugin validate <dir-or-manifest>")
			return 2
		}
		return pluginValidate(args[1], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown plugin subcommand %q (want validate)\n", args[0])
		return 2
	}
}

func pluginValidate(target string, stdout, stderr io.Writer) int {
	info, err := os.Stat(target)
	if err != nil {
		fmt.Fprintf(stderr, "stat %s: %v\n", target, err)
		return 1
	}
	dir := target
	if !info.IsDir() {
		dir = filepath.Dir(target)
	} else {
		target = filepath.Join(target, "manifest.json")
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		fmt.Fprintf(stderr, "read manifest: %v\n", err)
		return 1
	}
	m, err := plugin.ParseManifest(raw)
	if err != nil {
		fmt.Fprintf(stderr, "invalid manifest: %v\n", err)
		return 1
	}
	if err := m.Validate(dir); err != nil {
		fmt.Fprintf(stderr, "contract failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "ok %s %s (%s, api_version %d)\n", m.Name, m.Version, m.Kind, m.APIVersion)
	return 0
}
