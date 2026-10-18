package logic

import (
	"os"
	"path/filepath"
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

func loadFixture(t *testing.T) *Survey {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "testdata", "logic", "branch.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	s, err := Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return s
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// The branch fixture: q2 is hidden until q1 answers "no".
func TestBranchHidesQuestion(t *testing.T) {
	s := loadFixture(t)
	res := Evaluate(s, map[string]any{"q1": "yes"})
	if !eq(res.Visible, []string{"q1", "q3"}) {
		t.Fatalf("q1=yes visible = %v, want [q1 q3]", res.Visible)
	}
	res = Evaluate(s, map[string]any{"q1": "no"})
	if !eq(res.Visible, []string{"q1", "q2", "q3"}) {
		t.Fatalf("q1=no visible = %v, want [q1 q2 q3]", res.Visible)
	}
}

// A hidden question that becomes visible must not have been required
// while hidden, and a required visible question must have an answer.
func TestRequiredVisibleOnly(t *testing.T) {
	s := loadFixture(t)
	// q1 required, missing -> exactly one validation error naming q1.
	res := Evaluate(s, map[string]any{})
	if len(res.Errors) != 1 || res.Errors[0].Question != "q1" {
		t.Fatalf("errors = %+v, want one for q1", res.Errors)
	}
	// q1 answered, q2 stays hidden and unanswered: no error.
	res = Evaluate(s, map[string]any{"q1": "yes"})
	if len(res.Errors) != 0 {
		t.Fatalf("hidden unanswered must not error: %+v", res.Errors)
	}
}

// Bad op is rejected at parse time, not silently ignored.
func TestParseRejectsUnknownOp(t *testing.T) {
	_, err := Parse([]byte(`{
		"title":"t",
		"questions":[{"id":"q1","type":"single_choice","title":"x","choices":["a"]}],
		"logic":[{"id":"b1","when":{"question":"q1","op":"wildcard","value":"a"},"then":{"action":"show","question":"q1"}}]
	}`))
	if err == nil {
		t.Fatal("unknown op must be a parse error")
	}
}

// A logic rule pointing at a question that does not exist is a parse
// error — missing targets are caught before the survey can run.
func TestParseRejectsMissingTarget(t *testing.T) {
	_, err := Parse([]byte(`{
		"title":"t",
		"questions":[{"id":"q1","type":"single_choice","title":"x","choices":["a"]}],
		"logic":[{"id":"b1","when":{"question":"q1","op":"eq","value":"a"},"then":{"action":"show","question":"ghost"}}]
	}`))
	if err == nil {
		t.Fatal("missing target must be a parse error")
	}
}

// A branch loop (q1 shows q2, q2 shows q1) is rejected at parse time.
func TestParseRejectsLoop(t *testing.T) {
	_, err := Parse([]byte(`{
		"title":"t",
		"questions":[
			{"id":"q1","type":"single_choice","title":"x","choices":["a","b"]},
			{"id":"q2","type":"single_choice","title":"y","choices":["a","b"]}],
		"logic":[
			{"id":"b1","when":{"question":"q1","op":"eq","value":"a"},"then":{"action":"show","question":"q2"}},
			{"id":"b2","when":{"question":"q2","op":"eq","value":"a"},"then":{"action":"show","question":"q1"}}]
	}`))
	if err == nil {
		t.Fatal("cycle must be a parse error")
	}
}
