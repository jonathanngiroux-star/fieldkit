package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoBillingSurface is the release gate for the donation-only rule: no
// payment processor, no pricing tiers, no "contact sales" anywhere in
// source or shipped docs. "subscription"/"invoice" may appear only inside
// an explicit negation ("no subscriptions", "without invoices", ...).
func TestNoBillingSurface(t *testing.T) {
	banned := []string{
		"stripe", "contact sales", "seat math", "per-seat",
		"cloud pro", "cloud starter", "enterprise tier",
	}
	skipDirs := map[string]bool{".git": true, "node_modules": true, ".hermes": true, "testdata": false}
	_ = skipDirs
	skipFiles := map[string]bool{
		"docs/pricing.md":              true, // the "no pricing" stub
		"cmd/fieldkit/billing_test.go": true, // this file's own vocabulary
		"cmd/fieldkit/main_test.go":    true, // donate-exact test's own strings
		"AGENTS.md":                    true, // agent-facing notes
		"GOVERNANCE.md":                true, // policy file names the banned concepts to ban them
		"docs/audit/backend.md":        true, // audit note documents the gate itself
		"docs/audit/gui.md":            true, // audit note names the banned tokens it scans for
	}

	root := repoRoot(t)
	var violations []string
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == ".hermes" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		// internal planning briefs never ship (gitignored); excluded
		base := filepath.Base(rel)
		if rel == "IDEA.md" || (len(base) >= 3 && base[0] >= '0' && base[0] <= '9' && base[1] >= '0' && base[1] <= '9' && base[2] == '-' && strings.HasSuffix(base, ".md")) {
			return nil
		}
		if skipFiles[rel] || skipFiles[filepath.Base(path)] {
			return nil
		}
		switch filepath.Ext(path) {
		case ".go", ".js", ".md", ".json", ".yml", ".yaml", ".html", ".css", ".ts":
		default:
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lower := strings.ToLower(string(b))
		for _, ban := range banned {
			if strings.Contains(lower, ban) {
				violations = append(violations, rel+": contains \""+ban+"\"")
			}
		}
		for _, w := range []string{"subscription", "invoice"} {
			if strings.Contains(lower, w) && !containsNegation(lower, w) {
				violations = append(violations, rel+": non-negated \""+w+"\"")
			}
		}
		return nil
	})

	if len(violations) > 0 {
		t.Fatalf("billing surface found:\n%s", strings.Join(violations, "\n"))
	}
}

// containsNegation reports whether every occurrence of w is preceded
// (within 40 chars) by a negation word.
func containsNegation(s, w string) bool {
	for _, idx := range indicesOf(s, w) {
		start := idx - 40
		if start < 0 {
			start = 0
		}
		window := s[start:idx]
		neg := false
		for _, n := range []string{"no ", "never ", "not ", "without ", "isn't", "isn’t", "there never will be"} {
			if strings.Contains(window, n) {
				neg = true
				break
			}
		}
		if !neg {
			return false
		}
	}
	return true
}

func indicesOf(s, sub string) []int {
	var out []int
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			out = append(out, i)
		}
	}
	return out
}

// TestFixtureHygiene refuses suspicious tokens in checked-in fixtures so
// no live credential ever lands in git.
func TestFixtureHygiene(t *testing.T) {
	root := repoRoot(t)
	fixtures := filepath.Join(root, "testdata")
	bad := []string{
		"dsn=", "@sentry", "ingest.sentry", "AKIA", "sk_live", "secretAccessKey",
		"BEGIN PRIVATE KEY", "ghp_", "xoxb-",
	}
	var violations []string
	filepath.WalkDir(fixtures, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lower := strings.ToLower(string(b))
		for _, tok := range bad {
			if strings.Contains(lower, strings.ToLower(tok)) {
				rel, _ := filepath.Rel(root, path)
				violations = append(violations, rel+": contains suspicious token "+tok)
			}
		}
		return nil
	})
	if len(violations) > 0 {
		t.Fatalf("fixture hygiene violations:\n%s", strings.Join(violations, "\n"))
	}
}
