package main

import (
	"strings"
	"testing"

	"github.com/jonathanngiroux-star/fieldkit/internal/store"
)

// The wizard's steps are the product's actual onboarding: data dir ->
// fixture import (or the migrate dry-run) -> author a survey -> submit a
// response -> export -> donate. The last step shows both addresses.
func TestWizardStepsContract(t *testing.T) {
	steps := wizardSteps()
	if len(steps) < 5 {
		t.Fatalf("wizard too short: %d steps", len(steps))
	}
	ids := map[string]bool{}
	for _, s := range steps {
		ids[s.ID] = true
	}
	for _, want := range []string{"data", "import", "author", "respond", "export", "donate"} {
		if !ids[want] {
			t.Fatalf("wizard missing step %q (have %v)", want, ids)
		}
	}
	// donate step carries both addresses byte-for-byte
	var donateStep *wizardStep
	for i := range steps {
		if steps[i].ID == "donate" {
			donateStep = &steps[i]
		}
	}
	if donateStep == nil {
		t.Fatal("no donate step")
	}
	if !strings.Contains(donateStep.Body, "0x85ee7E71f762d772599cbF1EC20E651B30657521") ||
		!strings.Contains(donateStep.Body, "bc1qxe2zx5tv3hdreaej6s2x4p7han85uey828rrhg") {
		t.Fatalf("donate step missing addresses: %q", donateStep.Body)
	}
}

// wizardRunStep: the fixture-import step must create a real survey in a
// real store — the wizard runs real operations, not tooltips.
func TestWizardImportStepCreatesSurvey(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	m := newAppModel(st)
	out, err := wizardRunStep(m, "import", map[string]string{})
	if err != nil {
		t.Fatalf("import step: %v", err)
	}
	list, err := st.ListSurveys()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("import must create exactly 1 survey, got %d (%s)", len(list), out)
	}
	if list[0].Name != "Onboarding feedback" {
		t.Fatalf("imported name: %q", list[0].Name)
	}
}

// wizardRunStep: the author step must create the survey the user named.
func TestWizardAuthorStepCreatesSurvey(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	m := newAppModel(st)
	out, err := wizardRunStep(m, "author", map[string]string{"title": "My first survey"})
	if err != nil {
		t.Fatalf("author step: %v", err)
	}
	list, _ := st.ListSurveys()
	if len(list) != 1 || list[0].Name != "My first survey" {
		t.Fatalf("author step wrong: %+v (%s)", list, out)
	}
}

// wizardRunStep: the respond step submits a real response to the
// imported survey.
func TestWizardRespondStepSubmits(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	m := newAppModel(st)
	if _, err := wizardRunStep(m, "import", map[string]string{}); err != nil {
		t.Fatal(err)
	}
	_, err = wizardRunStep(m, "respond", map[string]string{"q1": "yes", "q3": "wizard"})
	if err != nil {
		t.Fatalf("respond step: %v", err)
	}
	list, _ := st.ListSurveys()
	rows, err := st.Responses(list[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("respond step must store 1 response, got %d", len(rows))
	}
}

// wizardRunStep: unknown step id is an error naming it.
func TestWizardUnknownStep(t *testing.T) {
	st, _ := store.Open(t.TempDir())
	defer st.Close()
	m := newAppModel(st)
	if _, err := wizardRunStep(m, "teleport", map[string]string{}); err == nil {
		t.Fatal("unknown step must error")
	}
}

// First-run state: unseen -> wizard must open; after WizardDone it must
// not auto-open (but stays reopenable).
func TestWizardSeenState(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	m := newAppModel(st)
	if m.wizardSeen() {
		t.Fatal("fresh data dir must be unseen")
	}
	if err := m.wizardDone(); err != nil {
		t.Fatal(err)
	}
	if !m.wizardSeen() {
		t.Fatal("wizardDone must persist seen state")
	}
}
