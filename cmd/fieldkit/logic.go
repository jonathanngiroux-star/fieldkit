package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/jonathanngiroux-star/fieldkit/internal/logic"
)

// cmdLogic runs the survey logic engine offline against an answers
// file. The same engine backs the submit endpoint; this is the CI mode:
// `fieldkit logic --survey s.json --answers a.json --format=json`.
func cmdLogic(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("logic", stderr)
	surveyPath := fs.String("survey", "", "survey definition JSON")
	answersPath := fs.String("answers", "", "answers JSON file")
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *surveyPath == "" {
		fmt.Fprintln(stderr, "--survey is required")
		return 2
	}
	if *answersPath == "" {
		fmt.Fprintln(stderr, "--answers is required")
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(stderr, "ambiguous input: --answers and a positional answers file — not both")
		return 2
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintf(stderr, "unknown --format %q (want text or json)\n", *format)
		return 2
	}
	raw, err := os.ReadFile(*surveyPath)
	if err != nil {
		fmt.Fprintf(stderr, "read survey: %v\n", err)
		return 1
	}
	s, err := logic.Parse(raw)
	if err != nil {
		fmt.Fprintf(stderr, "parse survey: %v\n", err)
		return 1
	}
	ansRaw, err := os.ReadFile(*answersPath)
	if err != nil {
		fmt.Fprintf(stderr, "read answers: %v\n", err)
		return 1
	}
	var answers map[string]any
	if err := json.Unmarshal(ansRaw, &answers); err != nil {
		fmt.Fprintf(stderr, "parse answers: %v\n", err)
		return 1
	}
	res := logic.Evaluate(s, answers)
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(res); err != nil {
		fmt.Fprintf(stderr, "encode result: %v\n", err)
		return 1
	}
	if len(res.Errors) > 0 {
		return 1
	}
	return 0
}
