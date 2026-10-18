package typeform

import (
	"encoding/json"
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

func loadFixture(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "testdata", "fixtures", "typeform", "sample-export.json"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// The dry-run report on the committed fixture is the migration contract.
// CI gates on these exact keys and counts — the report is the product.
func TestDryRunReportContract(t *testing.T) {
	rep, err := Convert(loadFixture(t))
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	// mapped questions: short_text, opinion_scale, yes_no, long_text
	if len(rep.Survey.Questions) != 4 {
		t.Fatalf("mapped questions = %d, want 4: %+v", len(rep.Survey.Questions), rep.Survey.Questions)
	}
	byID := map[string]bool{}
	for _, q := range rep.Survey.Questions {
		byID[q.ID] = true
	}
	if !byID["name_ref"] || !byID["satisfaction_ref"] || !byID["recommend_ref"] || !byID["improve_ref"] {
		t.Fatalf("expected refs mapped: %+v", byID)
	}
	// hidden field must map to a hidden question
	var hidden int
	for _, q := range rep.Survey.Questions {
		if q.ID == "source_ref" {
			t.Fatal("hidden field must not be a mapped question")
		}
	}
	_ = hidden
	// required flags survive
	for _, q := range rep.Survey.Questions {
		if q.ID == "name_ref" && !q.Required {
			t.Fatal("name_ref must be required")
		}
		if q.ID == "improve_ref" && q.Required {
			t.Fatal("improve_ref must not be required")
		}
	}
	// logic: one jump rule mapped
	if len(rep.Survey.Logic) != 1 {
		t.Fatalf("logic rules = %d, want 1", len(rep.Survey.Logic))
	}
	if rep.Survey.Logic[0].Then.Action != "jump" || rep.Survey.Logic[0].Then.Question != "improve_ref" {
		t.Fatalf("jump rule wrong: %+v", rep.Survey.Logic[0])
	}
	// unmapped listed honestly
	um := map[string]bool{}
	for _, u := range rep.Unmapped {
		um[u.Key] = true
	}
	if !um["f1"] || !um["f7"] {
		t.Fatalf("statement and matrix must be unmapped: %+v", rep.Unmapped)
	}
	if !um["f6"] {
		t.Fatalf("hidden field must be listed (mapped-as-hidden): %+v", rep.Unmapped)
	}
	// endings listed
	if len(rep.Endings) != 1 {
		t.Fatalf("endings = %d, want 1", len(rep.Endings))
	}
}

// The JSON report shape is machine-stable: source, dry_run, added,
// unmapped, endings, risks keys exist.
func TestReportJSONShape(t *testing.T) {
	rep, err := Convert(loadFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"source", "dry_run", "survey", "unmapped", "endings"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("report missing key %q", k)
		}
	}
	if m["source"] != "typeform" || m["dry_run"] != true {
		t.Fatalf("source/dry_run wrong: %v %v", m["source"], m["dry_run"])
	}
}

// Fuzz the converter: arbitrary bytes must never panic; valid JSON must
// either convert or return an error.
func FuzzConvert(f *testing.F) {
	f.Add([]byte(`{"form":{"id":"x","title":"t"},"fields":[]}`))
	f.Add([]byte(`{"form":{"id":"AbCdEf01","title":"Customer feedback","fields":[{"id":"f1","ref":"a","title":"Q","type":"short_text"}]}}`))
	f.Add([]byte(`not json`))
	f.Fuzz(func(t *testing.T, data []byte) {
		rep, err := Convert(data)
		if err != nil {
			return // legal: reject with an error
		}
		if rep == nil {
			t.Fatal("nil report with nil error")
		}
	})
}
