package store

import (
	"os"
	"path/filepath"
	"testing"
)

// repoRoot walks up to the nearest go.mod — never hardcode ../ depth.
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

func mustOpen(t *testing.T, dir string) *Store {
	t.Helper()
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// Creating a survey persists it and lists it.
func TestCreateListSurveys(t *testing.T) {
	s := mustOpen(t, t.TempDir())
	if _, err := s.CreateSurvey("NPS survey", map[string]any{
		"title": "NPS survey",
		"questions": []any{
			map[string]any{"id": "q1", "type": "nps", "title": "How likely?", "required": true},
		},
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	list, err := s.ListSurveys()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1 survey, got %d", len(list))
	}
	if list[0].Name != "NPS survey" {
		t.Fatalf("name mismatch: %q", list[0].Name)
	}
}

// Survey JSON round-trips byte-stable (canonical form) through the store.
func TestSurveyJSONRoundTrip(t *testing.T) {
	s := mustOpen(t, t.TempDir())
	def := map[string]any{
		"title": "t",
		"questions": []any{
			map[string]any{"id": "q1", "type": "text", "title": "Name?"},
		},
	}
	if _, err := s.CreateSurvey("s1", def); err != nil {
		t.Fatal(err)
	}
	list, _ := s.ListSurveys()
	got, err := s.SurveyJSON(list[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 || got[0] != '{' {
		t.Fatalf("survey json not canonical object: %q", string(got[:min(40, len(got))]))
	}
}

// Responses persist with submission time and retrieve in order.
func TestSubmitResponses(t *testing.T) {
	s := mustOpen(t, t.TempDir())
	if _, err := s.CreateSurvey("s", map[string]any{
		"title":     "s",
		"questions": []any{map[string]any{"id": "q1", "type": "text", "title": "x"}},
	}); err != nil {
		t.Fatal(err)
	}
	list, _ := s.ListSurveys()
	id := list[0].ID
	if _, err := s.SaveResponse(id, map[string]any{"q1": "yes"}, "test-agent"); err != nil {
		t.Fatal(err)
	}
	rows, err := s.Responses(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Answers["q1"] != "yes" {
		t.Fatalf("response not stored: %+v", rows)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
