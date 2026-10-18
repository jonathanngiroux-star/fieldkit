package store

import (
	"database/sql"
	"testing"
)

// Settings: tiny key/value persistence for wizard state and future
// preferences. Get on a missing key returns "" (not an error).
func TestSettingsRoundTrip(t *testing.T) {
	s := mustOpen(t, t.TempDir())
	if got := s.SettingGet("wizard_seen"); got != "" {
		t.Fatalf("missing key must return empty, got %q", got)
	}
	if err := s.SettingSet("wizard_seen", "1"); err != nil {
		t.Fatal(err)
	}
	if got := s.SettingGet("wizard_seen"); got != "1" {
		t.Fatalf("want 1, got %q", got)
	}
	if err := s.SettingSet("wizard_seen", "2"); err != nil {
		t.Fatal(err)
	}
	if got := s.SettingGet("wizard_seen"); got != "2" {
		t.Fatalf("upsert must overwrite, got %q", got)
	}
	_ = sql.ErrNoRows
}
