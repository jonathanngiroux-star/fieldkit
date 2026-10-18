package app

// tui_views.go: text views for the TUI. Same logic layer as the GUI
// (Model); these are pure formatting functions over it — a view can
// be tested without a screen.

import (
	"encoding/json"
	"fmt"
	"strings"
)

// tuiSurveyList renders the surveys table as text lines: header first.
func (m *Model) TUISurveyList() ([]string, error) {
	infos, err := m.SurveySummaries()
	if err != nil {
		return nil, err
	}
	out := []string{fmt.Sprintf("%-4s %-32s %s", "id", "name", "responses")}
	for _, s := range infos {
		out = append(out, fmt.Sprintf("%-4d %-32s %d responses", s["id"], s["name"], s["count"]))
	}
	return out, nil
}

// tuiLogicDryRun evaluates answers against a stored survey and renders
// the result as text (same engine as submit/CI).
func (m *Model) TUILogicDryRun(surveyID int64, answersJSON string) (string, error) {
	var answers map[string]any
	if strings.TrimSpace(answersJSON) != "" {
		if err := json.Unmarshal([]byte(answersJSON), &answers); err != nil {
			return "", fmt.Errorf("invalid answers JSON: %w", err)
		}
	}
	res, err := m.preview(surveyID, answers)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "visible: %s\n", strings.Join(res.Visible, ", "))
	if res.Next != "" {
		fmt.Fprintf(&b, "jumps to: %s\n", res.Next)
	}
	if len(res.Errors) == 0 {
		b.WriteString("valid: no errors\n")
	} else {
		for _, e := range res.Errors {
			fmt.Fprintf(&b, "error %s: %s\n", e.Question, e.Message)
		}
		return b.String(), fmt.Errorf("validation failed")
	}
	return b.String(), nil
}

// tuiMigrateSummary renders the dry-run report counts as text.
func (m *Model) TUIMigrateSummary(input string) (string, error) {
	repJSON, err := m.MigrateDryRun("typeform", input)
	if err != nil {
		return "", err
	}
	var rep struct {
		Survey struct {
			Questions []struct {
				ID string `json:"id"`
			} `json:"questions"`
			Logic []struct {
				ID string `json:"id"`
			} `json:"logic"`
		} `json:"survey"`
		Unmapped []struct {
			Key  string `json:"key"`
			Type string `json:"type"`
			Why  string `json:"why"`
		} `json:"unmapped"`
		Endings []struct {
			Ref string `json:"ref"`
		} `json:"endings"`
	}
	if err := json.Unmarshal([]byte(repJSON), &rep); err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d mapped, %d unmapped, %d logic rules, %d endings\n",
		len(rep.Survey.Questions), len(rep.Unmapped), len(rep.Survey.Logic), len(rep.Endings))
	for _, u := range rep.Unmapped {
		fmt.Fprintf(&b, "  unmapped %s (%s): %s\n", u.Key, u.Type, u.Why)
	}
	return b.String(), nil
}

// tuiDonate renders the donate text (both addresses, exact).
func (m *Model) TUIDonate() string {
	return "Fieldkit is free to self-host. There is no paid tier, no\n" +
		"subscription, no invoice, and no feature gated on money — there\n" +
		"never will be. Donations welcome:\n\n" +
		"Ethereum / USDC (ERC-20): " + DonateEthereum + "\n" +
		"Bitcoin:                  " + DonateBitcoin + "\n\n" +
		"No other wallets are associated with this project.\n" +
		"e = copy ETH/USDC · b = copy Bitcoin"
}
