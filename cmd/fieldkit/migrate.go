package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/jonathanngiroux-star/fieldkit/internal/migrate/typeform"
)

// cmdMigrate dry-runs a survey-platform import and prints the report.
//
//	fieldkit migrate --from=typeform --dry-run --format=json --input export.json
//
// Qualtrics is refused by name until the Typeform fixture is green and
// the qualtrics converter exists (see AGENTS.md refusals).
func cmdMigrate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("migrate", stderr)
	from := fs.String("from", "", "source platform: typeform (qualtrics: planned, gated on typeform fidelity)")
	dryRun := fs.Bool("dry-run", false, "dry-run: report only, nothing imported")
	format := fs.String("format", "text", "output format: text or json")
	input := fs.String("input", "", "path to the platform export file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *from == "" {
		fmt.Fprintln(stderr, "--from is required (typeform)")
		return 2
	}
	if *input == "" {
		fmt.Fprintln(stderr, "--input is required (path to the export file)")
		return 2
	}
	switch *from {
	case "typeform":
	case "qualtrics":
		fmt.Fprintln(stderr, "qualtrics migrate is not available yet: it is gated on a green typeform fixture first (see docs/fidelity/typeform.md)")
		return 2
	default:
		fmt.Fprintf(stderr, "unknown --from %q (want typeform)\n", *from)
		return 2
	}
	if !*dryRun {
		fmt.Fprintln(stderr, "only --dry-run is supported: apply mode is not part of v0.1 (see docs/release-v0.1.md)")
		return 2
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintf(stderr, "unknown --format %q (want text or json)\n", *format)
		return 2
	}
	raw, err := os.ReadFile(*input)
	if err != nil {
		fmt.Fprintf(stderr, "read %s: %v\n", *input, err)
		return 1
	}
	rep, err := typeform.Convert(raw)
	if err != nil {
		fmt.Fprintf(stderr, "convert: %v\n", err)
		return 1
	}
	if *format == "json" {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rep); err != nil {
			fmt.Fprintf(stderr, "encode report: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "typeform dry-run: %q -> fieldkit\n", rep.Survey.Title)
	fmt.Fprintf(stdout, "  %d question(s) mapped\n", len(rep.Survey.Questions))
	fmt.Fprintf(stdout, "  %d logic rule(s) mapped\n", len(rep.Survey.Logic))
	fmt.Fprintf(stdout, "  %d unmapped item(s):\n", len(rep.Unmapped))
	for _, u := range rep.Unmapped {
		fmt.Fprintf(stdout, "    - %s (%s): %s\n", u.Key, u.Type, u.Why)
	}
	fmt.Fprintf(stdout, "  %d ending(s) listed\n", len(rep.Endings))
	fmt.Fprintln(stdout, "nothing imported (dry-run). Donate: docs/donate.md")
	return 0
}
