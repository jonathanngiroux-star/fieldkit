package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("no go.mod upward")
	return ""
}

// logic command: run the branch fixture, expect visible q2 when q1=no,
// and a validation error naming q1 when the required answer is missing.
func TestLogicCommandFixture(t *testing.T) {
	root := repoRoot(t)
	survey := filepath.Join(root, "testdata", "logic", "branch.json")

	answers := t.TempDir()
	ansPath := filepath.Join(answers, "a.json")
	if err := os.WriteFile(ansPath, []byte(`{"q1":"no","q2":"the widget broke","q3":"n/a"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, errb bytes.Buffer
	code := run([]string{"logic", "--survey", survey, "--answers", ansPath, "--format=json"}, &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, errb.String())
	}
	var res struct {
		VisibleQuestions []string `json:"visible_questions"`
		Errors           []struct {
			Question string `json:"question"`
			Message  string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatalf("output not json: %v\n%s", err, out.String())
	}
	if len(res.VisibleQuestions) != 3 {
		t.Fatalf("visible = %v, want 3", res.VisibleQuestions)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("errors = %+v", res.Errors)
	}

	// Missing required answer: nonzero exit + error naming q1 + errors JSON on stdout.
	empty := filepath.Join(answers, "e.json")
	os.WriteFile(empty, []byte(`{"q3":"x"}`), 0o644)
	out.Reset()
	errb.Reset()
	code = run([]string{"logic", "--survey", survey, "--answers", empty, "--format=json"}, &out, &errb)
	if code == 0 {
		t.Fatal("required violation must exit nonzero")
	}
	if !strings.Contains(out.String(), `"q1"`) {
		t.Fatalf("error must name q1: %s", out.String())
	}
}

// logic command with both --answers and a positional answers file must be
// a usage error naming the ambiguity.
func TestLogicCommandAmbiguousInput(t *testing.T) {
	root := repoRoot(t)
	survey := filepath.Join(root, "testdata", "logic", "branch.json")
	var out, errb bytes.Buffer
	code := run([]string{"logic", "--survey", survey, "--answers", "a.json", "b.json", "--format=json"}, &out, &errb)
	if code == 0 {
		t.Fatal("ambiguous input must be an error")
	}
	if !strings.Contains(errb.String(), "not both") {
		t.Fatalf("error must name the ambiguity: %q", errb.String())
	}
}
