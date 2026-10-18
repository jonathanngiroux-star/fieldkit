package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The brief's exact invocation:
// fieldkit migrate --from=typeform --dry-run --format=json --input <file>
func TestMigrateTypeformDryRunJSON(t *testing.T) {
	root := repoRoot(t)
	fixture := filepath.Join(root, "testdata", "fixtures", "typeform", "sample-export.json")

	var out, errb bytes.Buffer
	code := run([]string{"migrate", "--from=typeform", "--dry-run", "--format=json", "--input", fixture}, &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, errb.String())
	}
	var rep map[string]any
	if err := json.Unmarshal(out.Bytes(), &rep); err != nil {
		t.Fatalf("not json: %v\n%s", err, out.String())
	}
	if rep["source"] != "typeform" || rep["dry_run"] != true {
		t.Fatalf("source/dry_run: %v %v", rep["source"], rep["dry_run"])
	}
	unmapped, _ := rep["unmapped"].([]any)
	if len(unmapped) != 3 {
		t.Fatalf("unmapped = %d, want 3 (statement, hidden, matrix)", len(unmapped))
	}
}

// --from=qualtrics must refuse with a named reason until the typeform
// fixture is green and qualtrics work is in scope.
func TestMigrateQualtricsRefused(t *testing.T) {
	var out, errb bytes.Buffer
	code := run([]string{"migrate", "--from=qualtrics", "--dry-run", "--format=json", "--input", "x.json"}, &out, &errb)
	if code == 0 {
		t.Fatal("qualtrics must be refused for now")
	}
	if !strings.Contains(errb.String(), "qualtrics") {
		t.Fatalf("refusal must name qualtrics: %q", errb.String())
	}
}

// Missing --input is a usage error.
func TestMigrateRequiresInput(t *testing.T) {
	var out, errb bytes.Buffer
	code := run([]string{"migrate", "--from=typeform", "--dry-run", "--format=json"}, &out, &errb)
	if code == 0 {
		t.Fatal("missing --input must fail")
	}
	if !strings.Contains(errb.String(), "--input") {
		t.Fatalf("error must name --input: %q", errb.String())
	}
}

// Non-dry-run must refuse (apply mode lands with the GUI step).
func TestMigrateApplyRefused(t *testing.T) {
	root := repoRoot(t)
	fixture := filepath.Join(root, "testdata", "fixtures", "typeform", "sample-export.json")
	var out, errb bytes.Buffer
	code := run([]string{"migrate", "--from=typeform", "--input", fixture, "--format=json"}, &out, &errb)
	if code == 0 {
		t.Fatal("apply mode must be refused until it is real")
	}
	if !strings.Contains(errb.String(), "dry-run") {
		t.Fatalf("refusal must name dry-run: %q", errb.String())
	}
}

// text format is a human summary; json stays machine-stable.
func TestMigrateTextFormat(t *testing.T) {
	root := repoRoot(t)
	fixture := filepath.Join(root, "testdata", "fixtures", "typeform", "sample-export.json")
	var out, errb bytes.Buffer
	code := run([]string{"migrate", "--from=typeform", "--dry-run", "--format=text", "--input", fixture}, &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "typeform") || !strings.Contains(out.String(), "4 question") {
		t.Fatalf("text summary wrong: %q", out.String())
	}
	_ = os.Getenv
}
