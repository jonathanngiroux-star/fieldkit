package store

import (
	"strings"
	"testing"
)

// CSV export: header from question ids in survey order, one row per
// response, formula-injection-safe values, RFC3339 timestamps.
func TestCSVExport(t *testing.T) {
	s := mustOpen(t, t.TempDir())
	_, err := s.CreateSurvey("s", map[string]any{
		"title": "s",
		"questions": []any{
			map[string]any{"id": "q1", "type": "text", "title": "Name"},
			map[string]any{"id": "q2", "type": "rating", "title": "Rate"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	list, _ := s.ListSurveys()
	id := list[0].ID
	if _, err := s.SaveResponse(id, map[string]any{"q1": "=SUM(A1:A2)", "q2": 4.0}, "agent"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveResponse(id, map[string]any{"q1": `say "hi", ok`, "q2": 5.0}, "agent"); err != nil {
		t.Fatal(err)
	}
	csv, err := s.ExportCSV(id)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(csv, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 lines (header + 2 rows), got %d:\n%s", len(lines), csv)
	}
	if lines[0] != "response_id,submitted_at,q1,q2" {
		t.Fatalf("header = %q", lines[0])
	}
	if !strings.Contains(lines[1], `'=SUM(A1:A2)`) || strings.Contains(lines[1], ",=SUM") {
		t.Fatalf("formula must be defanged: %q", lines[1])
	}
	if !strings.Contains(lines[1], "202") {
		t.Fatalf("timestamp missing: %q", lines[1])
	}
	if !strings.Contains(lines[2], `"say ""hi"", ok"`) {
		t.Fatalf("quoting broken: %q", lines[2])
	}
}
