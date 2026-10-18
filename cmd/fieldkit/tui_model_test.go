package main

import (
	fieldapp "github.com/jonathanngiroux-star/fieldkit/internal/app"

	"os"
	"strings"
	"testing"

	"github.com/jonathanngiroux-star/fieldkit/internal/store"
)

func openTUIModel(t *testing.T) (*fieldapp.Model, *store.Store) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return fieldapp.New(st), st
}

// The TUI's main view data: survey list with response counts.
func TestTUISurveyListView(t *testing.T) {
	m, _ := openTUIModel(t)
	if _, err := fieldapp.WizardRunStep(m, "import", map[string]string{}); err != nil {
		t.Fatal(err)
	}
	lines, err := m.TUISurveyList()
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) < 2 {
		t.Fatalf("want header + row, got %v", lines)
	}
	if !strings.Contains(lines[0], "id") || !strings.Contains(lines[0], "name") {
		t.Fatalf("header wrong: %q", lines[0])
	}
	if !strings.Contains(lines[1], "Onboarding feedback") {
		t.Fatalf("row missing imported survey: %q", lines[1])
	}
	if !strings.Contains(lines[1], "0 responses") {
		t.Fatalf("count missing: %q", lines[1])
	}
}

// The TUI's logic dry-run view: same engine, same fixture, text shape.
func TestTUILogicDryRun(t *testing.T) {
	m, st := openTUIModel(t)
	if _, err := fieldapp.WizardRunStep(m, "import", map[string]string{}); err != nil {
		t.Fatal(err)
	}
	list, _ := st.ListSurveys()
	out, err := m.TUILogicDryRun(list[0].ID, `{"q1":"no","q2":"broken","q3":"ok"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "visible: q1, q2, q3") {
		t.Fatalf("visible list wrong: %q", out)
	}
	if !strings.Contains(out, "valid") {
		t.Fatalf("verdict missing: %q", out)
	}
	// invalid: required q1 missing
	out, err = m.TUILogicDryRun(list[0].ID, `{"q3":"x"}`)
	if err == nil || !strings.Contains(out, "q1") {
		t.Fatalf("required error must name q1: %q err=%v", out, err)
	}
}

// The TUI's migrate summary: text form of the same report.
func TestTUIMigrateSummary(t *testing.T) {
	m, _ := openTUIModel(t)
	path := t.TempDir() + "/tf.json"
	if err := writeFile(path, fieldapp.SampleTypeformExport); err != nil {
		t.Fatal(err)
	}
	out, err := m.TUIMigrateSummary(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "4 mapped") || !strings.Contains(out, "3 unmapped") {
		t.Fatalf("summary counts wrong: %q", out)
	}
	if !strings.Contains(out, "matrix") {
		t.Fatalf("unmapped detail missing: %q", out)
	}
}

// The TUI donate view text: both addresses, exact.
func TestTUIDonateView(t *testing.T) {
	m, _ := openTUIModel(t)
	out := m.TUIDonate()
	if !strings.Contains(out, "0x85ee7E71f762d772599cbF1EC20E651B30657521") ||
		!strings.Contains(out, "bc1qxe2zx5tv3hdreaej6s2x4p7han85uey828rrhg") {
		t.Fatalf("donate view missing addresses: %q", out)
	}
}

// The TUI wizard is the shared fieldapp.WizardSteps/fieldapp.WizardRunStep — zero TUI-only
// logic. This pins that the same steps run in both renderers.
func TestTUIWizardSharesSteps(t *testing.T) {
	m, _ := openTUIModel(t)
	// import through the shared runner (as the TUI wizard does)
	out, err := fieldapp.WizardRunStep(m, "import", map[string]string{})
	if err != nil || !strings.Contains(out, "imported") {
		t.Fatalf("TUI wizard import: %q %v", out, err)
	}
	seen := m.WizardSeen()
	if seen {
		t.Fatal("import alone must not mark the wizard seen")
	}
}

// writeFile is a tiny helper (tests share it).
func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
