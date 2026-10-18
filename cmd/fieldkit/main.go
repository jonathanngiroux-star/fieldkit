// Command fieldkit is the single binary: survey backend, migration CLI,
// logic runner, plugin contract checker. One process, one SQLite file.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jonathanngiroux-star/fieldkit/internal/store"
)

// version is the single source of truth, overridden at build time via
// -ldflags "-X main.version=...".
var version = "0.1.0"

// Donation addresses — binding. Changes require the GOVERNANCE.md
// process (unanimous maintainer consent + 30-day public notice).
const (
	donateEthereum = "0x85ee7E71f762d772599cbF1EC20E651B30657521"
	donateBitcoin  = "bc1qxe2zx5tv3hdreaej6s2x4p7han85uey828rrhg"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "version":
		return cmdVersion(stdout)
	case "donate":
		return cmdDonate(rest, stdout, stderr)
	case "desktop":
		return launchDesktop(rest, stdout, stderr)
	case "surveys":
		return cmdSurveys(rest, stdout, stderr)
	case "logic":
		return cmdLogic(rest, stdout, stderr)
	case "serve":
		return cmdServe(rest, stdout, stderr)
	case "export":
		return cmdExport(rest, stdout, stderr)
	case "plugin":
		return cmdPlugin(rest, stdout, stderr)
	case "migrate":
		return cmdMigrate(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", cmd)
		usage(stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `usage: fieldkit <command> [flags]

commands:
  version   print version
  donate    print donation addresses
  surveys   create and list surveys (create --file, list)
  serve     run the HTTP server (public submit + /donate)
  migrate   dry-run an import from a survey platform (typeform)
  logic     run survey logic against an answer file
  export    export responses (csv)
  plugin    validate plugin manifests and run contract checks

Fieldkit is free to self-host; donations only — see docs/donate.md.
`)
}

func cmdVersion(stdout io.Writer) int {
	fmt.Fprintf(stdout, "fieldkit %s\n", version)
	return 0
}

func cmdDonate(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		fmt.Fprintln(stderr, "usage: fieldkit donate")
		return 2
	}
	fmt.Fprint(stdout, `Fieldkit is free to self-host. If it replaces a survey-platform
line item for you, donations are welcome:

Ethereum / USDC (ERC-20): `+donateEthereum+`
Bitcoin:                  `+donateBitcoin+`

No feature is gated on donations. Details: docs/donate.md
`)
	return 0
}

func cmdSurveys(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: fieldkit surveys create --file <survey.json> --data-dir <dir> | fieldkit surveys list --data-dir <dir>")
		return 2
	}
	switch args[0] {
	case "create":
		return cmdSurveysCreate(args[1:], stdout, stderr)
	case "list":
		return cmdSurveysList(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown surveys subcommand %q\n", args[0])
		return 2
	}
}

// newFlagSet builds a ContinueOnError flag set that routes errors to
// stderr without printing usage twice.
func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

// defaultDataDir is where the CLI stores its SQLite file when --data-dir
// is not given: ~/.local/share/fieldkit (XDG), with a fallback for
// non-Linux.
func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "fieldkit-data"
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "AppData", "Local", "Fieldkit", "Data")
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "Fieldkit")
	}
	return filepath.Join(home, ".local", "share", "fieldkit")
}

// jsonUnmarshal is a strict JSON decode used across the CLI.
func jsonUnmarshal(raw []byte, v any) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	return dec.Decode(v)
}

func cmdSurveysCreate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("surveys create", stderr)
	file := fs.String("file", "", "survey definition JSON file")
	name := fs.String("name", "", "survey name (defaults to the definition title)")
	dataDir := fs.String("data-dir", defaultDataDir(), "data directory for the SQLite store")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *file == "" {
		fmt.Fprintln(stderr, "--file is required")
		return 2
	}
	raw, err := os.ReadFile(*file)
	if err != nil {
		fmt.Fprintf(stderr, "read %s: %v\n", *file, err)
		return 1
	}
	var def map[string]any
	if err := jsonUnmarshal(raw, &def); err != nil {
		fmt.Fprintf(stderr, "parse %s: %v\n", *file, err)
		return 1
	}
	if *name == "" {
		if t, ok := def["title"].(string); ok && t != "" {
			*name = t
		} else {
			fmt.Fprintln(stderr, "--name required (definition has no title)")
			return 2
		}
	}
	st, err := store.Open(*dataDir)
	if err != nil {
		fmt.Fprintf(stderr, "open store: %v\n", err)
		return 1
	}
	defer st.Close()
	newID, err := st.CreateSurvey(*name, def)
	if err != nil {
		fmt.Fprintf(stderr, "create survey: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "created survey %d %q\n", newID, *name)
	return 0
}

func cmdSurveysList(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("surveys list", stderr)
	dataDir := fs.String("data-dir", defaultDataDir(), "data directory for the SQLite store")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	st, err := store.Open(*dataDir)
	if err != nil {
		fmt.Fprintf(stderr, "open store: %v\n", err)
		return 1
	}
	defer st.Close()
	list, err := st.ListSurveys()
	if err != nil {
		fmt.Fprintf(stderr, "list surveys: %v\n", err)
		return 1
	}
	if len(list) == 0 {
		fmt.Fprintln(stdout, "no surveys")
		return 0
	}
	for _, si := range list {
		fmt.Fprintf(stdout, "%d\t%s\n", si.ID, si.Name)
	}
	return 0
}
