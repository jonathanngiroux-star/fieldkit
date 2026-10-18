package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonathanngiroux-star/fieldkit/internal/logic"
	"github.com/jonathanngiroux-star/fieldkit/internal/migrate/typeform"
	"github.com/jonathanngiroux-star/fieldkit/internal/plugin"
	"github.com/jonathanngiroux-star/fieldkit/internal/store"
)

// fixtureSurvey is the embedded branch fixture — the wizard's demo data
// and the "already created" object the main view lands on.
//
//go:embed fixture.json
var fixtureSurvey []byte

// sampleTypeformExport is the bundled Typeform export for migrate demos.
//
//go:embed sample_typeform_export.json
var sampleTypeformExport []byte

// appModel is the GUI/TUI logic layer: every bound method is a thin
// wrapper over the store + engine packages. No logic lives in the
// frontend.
type appModel struct {
	st *store.Store
}

func newAppModel(st *store.Store) *appModel { return &appModel{st: st} }

// ---------- survey authoring / listing ----------

// surveySummaries lists surveys with response counts.
func (m *appModel) surveySummaries() ([]map[string]any, error) {
	list, err := m.st.ListSurveys()
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(list))
	for _, si := range list {
		rows, err := m.st.Responses(si.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id":      si.ID,
			"name":    si.Name,
			"created": si.CreatedAt.Format(time.RFC3339),
			"count":   len(rows),
		})
	}
	return out, nil
}

// createSurveyFromJSON validates and stores a definition; returns the id.
func (m *appModel) createSurveyFromJSON(raw []byte, name string) (int64, error) {
	var def map[string]any
	if err := json.Unmarshal(raw, &def); err != nil {
		return 0, fmt.Errorf("invalid survey JSON: %w", err)
	}
	if name == "" {
		if t, ok := def["title"].(string); ok && t != "" {
			name = t
		} else {
			return 0, fmt.Errorf("survey needs a title or a name")
		}
	}
	return m.st.CreateSurvey(name, def)
}

// surveyDefinition returns the stored definition JSON for editing.
func (m *appModel) surveyDefinition(id int64) (string, error) {
	b, err := m.st.SurveyJSON(id)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// deleteSurvey removes a survey and its responses.
func (m *appModel) deleteSurvey(id int64) error { return m.st.DeleteSurvey(id) }

// ---------- respondent runner ----------

// preview evaluates a stored survey against answers without storing:
// visible questions, validation errors. Same engine as submit/CI.
func (m *appModel) preview(id int64, answers map[string]any) (logic.Result, error) {
	def, err := m.st.SurveyJSON(id)
	if err != nil {
		return logic.Result{}, err
	}
	s, err := logic.Parse(def)
	if err != nil {
		return logic.Result{}, err
	}
	return logic.Evaluate(s, answers), nil
}

// previewJSON parses an answers JSON string and previews.
func (m *appModel) previewJSON(id int64, answersJSON string) (logic.Result, error) {
	var answers map[string]any
	if answersJSON != "" {
		if err := json.Unmarshal([]byte(answersJSON), &answers); err != nil {
			return logic.Result{}, fmt.Errorf("invalid answers JSON: %w", err)
		}
	}
	return m.preview(id, answers)
}

// submitResponse validates and stores one response (the GUI's
// "respondent page" submit action).
func (m *appModel) submitResponse(id int64, answers map[string]any, agent string) (int64, logic.Result, error) {
	res, err := m.preview(id, answers)
	if err != nil {
		return 0, res, err
	}
	if len(res.Errors) > 0 {
		return 0, res, fmt.Errorf("validation failed")
	}
	keep := map[string]any{}
	for _, qid := range res.Visible {
		if v, ok := answers[qid]; ok {
			keep[qid] = v
		}
	}
	rid, err := m.st.SaveResponse(id, keep, agent)
	return rid, res, err
}

// responses lists a survey's responses for the table view.
func (m *appModel) responses(id int64) ([]map[string]any, error) {
	rows, err := m.st.Responses(id)
	if err != nil {
		return nil, err
	}
	def, err := m.st.SurveyJSON(id)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Questions []struct {
			ID string `json:"id"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(def, &parsed); err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		answers := map[string]any{}
		for _, q := range parsed.Questions {
			if v, ok := r.Answers[q.ID]; ok {
				answers[q.ID] = v
			}
		}
		out = append(out, map[string]any{
			"response_id": r.ID,
			"submitted":   r.SubmittedAt.Format(time.RFC3339),
			"answers":     answers,
		})
	}
	return out, nil
}

// exportCSV returns the CSV text for download.
func (m *appModel) exportCSV(id int64) (string, error) { return m.st.ExportCSV(id) }

// ---------- migrate ----------

// migrateDryRun runs the Typeform converter on a file and returns the
// JSON report.
func (m *appModel) migrateDryRun(source, input string) (string, error) {
	if source != "typeform" {
		return "", fmt.Errorf("only typeform is supported (qualtrics is gated on typeform fidelity)")
	}
	raw, err := os.ReadFile(input)
	if err != nil {
		return "", err
	}
	rep, err := typeform.Convert(raw)
	if err != nil {
		return "", err
	}
	out, err := json.MarshalIndent(rep, "", "  ")
	return string(out), err
}

// sampleFixturePath materializes the embedded Typeform sample export for
// users without an export yet.
func sampleFixturePath() (string, error) {
	f, err := os.CreateTemp("", "fieldkit-typeform-sample-*.json")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(sampleTypeformExport); err != nil {
		return "", err
	}
	return f.Name(), nil
}

// ---------- plugin contract ----------

// pluginValidate validates a manifest file or directory and returns the
// ok line or the error text (the GUI shows errors, never a 500).
func (m *appModel) pluginValidate(target string) (string, bool) {
	dir := target
	manifest := filepath.Join(target, "manifest.json")
	if info, err := os.Stat(target); err == nil && !info.IsDir() {
		manifest = target
		dir = filepath.Dir(target)
	}
	raw, err := os.ReadFile(manifest)
	if err != nil {
		return "read manifest: " + err.Error(), false
	}
	pm, err := plugin.ParseManifest(raw)
	if err != nil {
		return "invalid manifest: " + err.Error(), false
	}
	if err := pm.Validate(dir); err != nil {
		return "contract failed: " + err.Error(), false
	}
	return fmt.Sprintf("ok %s %s (%s, api_version %d)", pm.Name, pm.Version, pm.Kind, pm.APIVersion), true
}

// ---------- wizard ----------

// wizardField is one wizard input.
type wizardField struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Type     string `json:"type"` // text | textarea
	Required bool   `json:"required"`
	Value    string `json:"value"`
}

// wizardStep is one first-run wizard page.
type wizardStep struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Body      string        `json:"body"`
	Fields    []wizardField `json:"fields"`
	NextLabel string        `json:"nextLabel"`
}

// wizardSteps returns the first-run sequence. Steps are specific to
// this product, in order; the last step is the donate step with both
// addresses. "Skip" only exists on the data-dir step (the rest of the
// wizard is the core workflow).
func wizardSteps() []wizardStep {
	return []wizardStep{
		{
			ID:        "data",
			Title:     "Where should Fieldkit store data?",
			Body:      "One SQLite file under a data directory you choose. This is the same data the CLI, the TUI, and this window use.",
			Fields:    []wizardField{{ID: "datadir", Label: "Data directory", Type: "text", Required: false, Value: defaultDataDir()}},
			NextLabel: "Next",
		},
		{
			ID:        "import",
			Title:     "Import the sample survey",
			Body:      "Creates the demo survey from the checked-in fixture (one required choice question, one branch, one hidden question) so the main view is never empty. You can also dry-run a Typeform export from the Migrate tab later.",
			NextLabel: "Import",
		},
		{
			ID:        "author",
			Title:     "Author your own survey",
			Body:      "Surveys are JSON documents (questions, branching, required, hidden fields). This step creates a starter survey you can edit in the Author tab.",
			Fields:    []wizardField{{ID: "title", Label: "Survey title", Type: "text", Required: true, Value: "My first survey"}},
			NextLabel: "Create",
		},
		{
			ID:    "respond",
			Title: "Answer it as a respondent",
			Body:  "The respondent runner walks the survey with live logic — hidden questions appear only when their branch fires. Submitting here stores a real response.",
			Fields: []wizardField{
				{ID: "q1", Label: "Did the onboarding flow work? (yes/no)", Type: "text", Required: true, Value: "yes"},
				{ID: "q3", Label: "Anything else?", Type: "text", Required: false, Value: ""},
			},
			NextLabel: "Submit",
		},
		{
			ID:        "export",
			Title:     "See the responses",
			Body:      "The response table shows every stored response, and CSV download exports them (formula-safe). The same data is available via `fieldkit export csv` from the terminal.",
			NextLabel: "Next",
		},
		{
			ID:        "donate",
			Title:     "Fieldkit is donation-only",
			Body:      "There is no paid tier, no subscription, no invoice, and no feature gated on money — there never will be. If Fieldkit replaces a paid survey-platform line item for you, donations are welcome:\n\nEthereum / USDC (ERC-20): " + donateEthereum + "\nBitcoin: " + donateBitcoin + "\n\nNo other wallets are associated with this project.",
			NextLabel: "Finish",
		},
	}
}

// wizardRunStep executes a step's real operation on the user's values.
func wizardRunStep(m *appModel, stepID string, values map[string]string) (string, error) {
	switch stepID {
	case "data":
		// No-op here: the data dir was chosen at launch; this step
		// records that the wizard was walked.
		return "data directory confirmed", nil
	case "import":
		id, err := m.createSurveyFromJSON(fixtureSurvey, "Onboarding feedback")
		if err != nil {
			return "", fmt.Errorf("import fixture: %w", err)
		}
		return fmt.Sprintf("imported survey %d \"Onboarding feedback\"", id), nil
	case "author":
		title := strings.TrimSpace(values["title"])
		if title == "" {
			return "", fmt.Errorf("survey title required")
		}
		raw := fmt.Sprintf(`{"title":%q,"questions":[{"id":"q1","type":"text","title":%q,"required":true}]}`, title, "What would you like to ask?")
		id, err := m.createSurveyFromJSON([]byte(raw), title)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("created survey %d %q", id, title), nil
	case "respond":
		list, err := m.st.ListSurveys()
		if err != nil {
			return "", err
		}
		if len(list) == 0 {
			return "", fmt.Errorf("no survey to answer yet — run the import step first")
		}
		target := list[0]
		for _, si := range list {
			if si.Name == "Onboarding feedback" {
				target = si
			}
		}
		answers := map[string]any{}
		if v := strings.TrimSpace(values["q1"]); v != "" {
			answers["q1"] = v
		}
		if v := strings.TrimSpace(values["q3"]); v != "" {
			answers["q3"] = v
		}
		rid, _, err := m.submitResponse(target.ID, answers, "fieldkit-wizard")
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("stored response %d", rid), nil
	case "export":
		list, err := m.st.ListSurveys()
		if err != nil {
			return "", err
		}
		if len(list) == 0 {
			return "no surveys yet — the CSV export will have a header only", nil
		}
		csv, err := m.exportCSV(list[0].ID)
		if err != nil {
			return "", err
		}
		lines := strings.Count(csv, "\n")
		return fmt.Sprintf("CSV ready: %d line(s) (header + responses)", lines), nil
	case "donate":
		if err := m.wizardDone(); err != nil {
			return "", err
		}
		return "done", nil
	default:
		return "", fmt.Errorf("unknown wizard step %q", stepID)
	}
}

// wizardSeen reports whether the first-run wizard has been completed.
func (m *appModel) wizardSeen() bool {
	return m.st.SettingGet("wizard_seen") == "1"
}

// wizardDone persists wizard completion.
func (m *appModel) wizardDone() error {
	return m.st.SettingSet("wizard_seen", "1")
}
