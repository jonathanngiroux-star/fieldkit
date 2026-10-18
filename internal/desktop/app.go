//go:build wails

package desktop

// desktop_wails.go: `fieldkit desktop` launches the Wails v2 desktop
// GUI. This file only compiles with `-tags wails` (webkit2gtk-4.1 on
// Linux, WebView2 on Windows). The GUI is a thin renderer over
// appModel — one logic layer, thin renderers (CLI, TUI, GUI); the
// frontend is vanilla JS embedded in the binary: no npm tree, no
// bundler.

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wr "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/jonathanngiroux-star/fieldkit/internal/app"
	"github.com/jonathanngiroux-star/fieldkit/internal/logic"
	"github.com/jonathanngiroux-star/fieldkit/internal/plugin"
	"github.com/jonathanngiroux-star/fieldkit/internal/store"
)

//go:embed all:frontend/dist
var desktopAssets embed.FS

// DesktopApp is the struct bound to the frontend; every method is a
// thin wrapper over appModel (which wraps store/logic/plugin/migrate).
type DesktopApp struct {
	ctx context.Context
	m   *app.Model
}

// ---------- surveys ----------

// Surveys lists surveys with response counts.
func (a *DesktopApp) Surveys() ([]map[string]any, error) { return a.m.SurveySummaries() }

// CreateSurvey validates + stores a definition; returns the new id.
func (a *DesktopApp) CreateSurvey(defJSON, name string) (map[string]any, error) {
	id, err := a.m.CreateSurveyFromJSON([]byte(defJSON), name)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": id}, nil
}

// SurveyDefinition returns the stored definition for editing.
func (a *DesktopApp) SurveyDefinition(id int64) (string, error) { return a.m.SurveyDefinition(id) }

// DeleteSurvey removes a survey and its responses.
func (a *DesktopApp) DeleteSurvey(id int64) error { return a.m.DeleteSurvey(id) }

// ---------- respondent runner ----------

// Preview evaluates answers without storing: visible ids + errors.
func (a *DesktopApp) Preview(id int64, answersJSON string) (logic.Result, error) {
	return a.m.PreviewJSON(id, answersJSON)
}

// SubmitResponse validates + stores one response; returns its id.
func (a *DesktopApp) SubmitResponse(id int64, answersJSON string) (map[string]any, error) {
	var answers map[string]any
	if strings.TrimSpace(answersJSON) != "" {
		if err := json.Unmarshal([]byte(answersJSON), &answers); err != nil {
			return nil, fmt.Errorf("invalid answers JSON: %w", err)
		}
	}
	rid, res, err := a.m.SubmitResponse(id, answers, "fieldkit-desktop")
	if err != nil && len(res.Errors) == 0 {
		return nil, err
	}
	return map[string]any{"response_id": rid, "result": res}, nil
}

// Responses lists stored responses for the table.
func (a *DesktopApp) Responses(id int64) ([]map[string]any, error) { return a.m.Responses(id) }

// ExportCSV returns CSV text for download.
func (a *DesktopApp) ExportCSV(id int64) (string, error) { return a.m.ExportCSV(id) }

// ---------- migrate ----------

// MigrateDryRun runs the Typeform converter on a file.
func (a *DesktopApp) MigrateDryRun(source, input string) (string, error) {
	return a.m.MigrateDryRun(source, input)
}

// SampleFixturePath materializes the bundled Typeform export for demos.
func (a *DesktopApp) SampleFixturePath() (string, error) { return app.SampleFixturePath() }

// ---------- plugin ----------

// PluginValidate validates a manifest (file or dir); the GUI shows the
// error string, never a 500.
func (a *DesktopApp) PluginValidate(target string) (map[string]any, error) {
	msg, ok := a.m.PluginValidate(target)
	return map[string]any{"ok": ok, "message": msg}, nil
}

// PluginValidateJSON validates a pasted manifest ("upload"): the same
// api_version-1 contract over the JSON text; entryDir (optional) is
// where the entrypoint file is checked. Errors name the field — the
// frontend shows them, never a 500.
func (a *DesktopApp) PluginValidateJSON(manifestJSON, entryDir string) (map[string]any, error) {
	m, err := plugin.ParseManifest([]byte(manifestJSON))
	if err != nil {
		return map[string]any{"ok": false, "message": "invalid manifest: " + err.Error()}, nil
	}
	if entryDir != "" {
		if err := m.Validate(entryDir); err != nil {
			return map[string]any{"ok": false, "message": "contract failed: " + err.Error()}, nil
		}
	}
	return map[string]any{
		"ok":      true,
		"message": fmt.Sprintf("ok %s %s (%s, api_version %d)", m.Name, m.Version, m.Kind, m.APIVersion),
	}, nil
}

// ---------- wizard ----------

// WizardSteps returns the first-run sequence.
func (a *DesktopApp) WizardSteps() []app.WizardStep { return app.WizardSteps() }

// WizardRunStep runs a step's real operation on the user's values.
func (a *DesktopApp) WizardRunStep(stepID string, values map[string]string) (string, error) {
	return app.WizardRunStep(a.m, stepID, values)
}

// WizardSeen reports first-run state; WizardDone marks it complete.
func (a *DesktopApp) WizardSeen() bool  { return a.m.WizardSeen() }
func (a *DesktopApp) WizardDone() error { return a.m.WizardDone() }

// ---------- misc ----------

// Version returns the binary app.Version.
func (a *DesktopApp) Version() string { return app.Version }

// DonateText returns the donate text with both addresses.
func (a *DesktopApp) DonateText() string {
	return "Fieldkit is free to self-host. Donations:\nEthereum / USDC (ERC-20): " + app.DonateEthereum + "\nBitcoin: " + app.DonateBitcoin + "\nNo feature is gated on donations. Details: docs/donate.md"
}

// CopyAddress copies text to the clipboard (donate copy buttons).
func (a *DesktopApp) CopyAddress(text string) (string, error) {
	if err := wr.ClipboardSetText(a.ctx, text); err != nil {
		return "", err
	}
	return "copied", nil
}

// PickJSONPath opens a native file dialog for a JSON file.
func (a *DesktopApp) PickJSONPath(title string) (string, error) {
	return wr.OpenFileDialog(a.ctx, wr.OpenDialogOptions{
		Title: title,
		Filters: []wr.FileFilter{
			{DisplayName: "JSON (*.json)", Pattern: "*.json"},
			{DisplayName: "All files (*.*)", Pattern: "*.*"},
		},
	})
}

// SaveCSVPath opens a native save dialog for the CSV export.
func (a *DesktopApp) SaveCSVPath(suggested string) (string, error) {
	return wr.SaveFileDialog(a.ctx, wr.SaveDialogOptions{
		Title:           "Save responses CSV",
		DefaultFilename: suggested,
		Filters: []wr.FileFilter{
			{DisplayName: "CSV (*.csv)", Pattern: "*.csv"},
		},
	})
}

// SaveFile writes text to a chosen path (the CSV download action).
func (a *DesktopApp) SaveFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// runDesktop builds and shows the Wails window.
func runDesktop(m *app.Model) int {
	app := &DesktopApp{m: m}
	err := wails.Run(&options.App{
		Title:     "fieldkit",
		Width:     1024,
		Height:    700,
		MinWidth:  720,
		MinHeight: 480,
		AssetServer: &assetserver.Options{
			Assets: desktopAssets,
		},
		OnStartup: func(ctx context.Context) {
			app.ctx = ctx
			app.startTestBridge()
		},
		Bind: []interface{}{app},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "desktop: %v\n", err)
		return 1
	}
	return 0
}

// Run opens the store and runs the Wails app. dataDir empty falls back
// to FIELDKIT_DATA_DIR then the app package's default.
func Run(dataDir string) int {
	if dataDir == "" {
		dataDir = os.Getenv("FIELDKIT_DATA_DIR")
	}
	if dataDir == "" {
		dataDir = app.DefaultDataDir()
	}
	st, err := store.Open(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "desktop: %v\n", err)
		return 1
	}
	defer st.Close()
	return runDesktop(app.New(st))
}
