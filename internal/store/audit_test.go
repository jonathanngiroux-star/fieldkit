package store

import (
	"strings"
	"sync"
	"testing"
)

// AUDIT: the store must reject definitions the logic engine cannot run —
// a survey that enters the DB invalid turns every later submit into a
// 500. Validation belongs at the write boundary.
func TestCreateSurveyRejectsInvalidDefinition(t *testing.T) {
	s := mustOpen(t, t.TempDir())
	_, err := s.CreateSurvey("bad", map[string]any{
		"title":     "bad",
		"questions": []any{map[string]any{"id": "q1", "type": "hologram", "title": "x"}},
	})
	if err == nil {
		t.Fatal("invalid definition must be rejected at create")
	}
	if !strings.Contains(err.Error(), "hologram") {
		t.Fatalf("rejection must name the bad type: %v", err)
	}
	// and the DB must be unchanged
	list, _ := s.ListSurveys()
	if len(list) != 0 {
		t.Fatalf("rejected survey must not be stored, got %d", len(list))
	}
}

// AUDIT: SaveResponse must return the inserted id — callers reporting
// response ids by re-reading rows race and duplicate under concurrency.
func TestSaveResponseReturnsID(t *testing.T) {
	s := mustOpen(t, t.TempDir())
	if _, err := s.CreateSurvey("s", map[string]any{
		"title":     "s",
		"questions": []any{map[string]any{"id": "q1", "type": "text", "title": "x"}},
	}); err != nil {
		t.Fatal(err)
	}
	list, _ := s.ListSurveys()
	id1, err := s.SaveResponse(list[0].ID, map[string]any{"q1": "a"}, "t")
	if err != nil {
		t.Fatal(err)
	}
	id2, err := s.SaveResponse(list[0].ID, map[string]any{"q1": "b"}, "t")
	if err != nil {
		t.Fatal(err)
	}
	if id1 == 0 || id2 == 0 || id1 == id2 {
		t.Fatalf("ids must be distinct and nonzero: %d %d", id1, id2)
	}
	if id2 != id1+1 {
		t.Fatalf("expected sequential ids: %d then %d", id1, id2)
	}
}

// AUDIT: concurrent identical writes must not corrupt or lose rows.
func TestConcurrentSaveResponse(t *testing.T) {
	s := mustOpen(t, t.TempDir())
	if _, err := s.CreateSurvey("s", map[string]any{
		"title":     "s",
		"questions": []any{map[string]any{"id": "q1", "type": "text", "title": "x"}},
	}); err != nil {
		t.Fatal(err)
	}
	list, _ := s.ListSurveys()
	const N = 50
	ids := make([]int64, N)
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id, err := s.SaveResponse(list[0].ID, map[string]any{"q1": "x"}, "race")
			if err != nil {
				t.Errorf("save: %v", err)
				return
			}
			ids[i] = id
		}(i)
	}
	wg.Wait()
	seen := map[int64]bool{}
	for _, id := range ids {
		if id == 0 || seen[id] {
			t.Fatalf("duplicate or zero id: %d", id)
		}
		seen[id] = true
	}
	rows, err := s.Responses(list[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != N {
		t.Fatalf("want %d rows, got %d", N, len(rows))
	}
}

// AUDIT: CSV must never carry internal columns (agent/user-agent).
func TestCSVExcludesAgent(t *testing.T) {
	s := mustOpen(t, t.TempDir())
	if _, err := s.CreateSurvey("s", map[string]any{
		"title":     "s",
		"questions": []any{map[string]any{"id": "q1", "type": "text", "title": "x"}},
	}); err != nil {
		t.Fatal(err)
	}
	list, _ := s.ListSurveys()
	if _, err := s.SaveResponse(list[0].ID, map[string]any{"q1": "answer"}, "secret-agent-token-xyz"); err != nil {
		t.Fatal(err)
	}
	csv, err := s.ExportCSV(list[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(csv, "secret-agent-token-xyz") {
		t.Fatalf("csv leaks agent: %q", csv)
	}
	if strings.Contains(csv, "agent") {
		t.Fatalf("csv has an agent column: %q", csv)
	}
}
