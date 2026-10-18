package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The versioned manifest contract: a valid manifest parses, an invalid
// one is rejected with a reason naming the field.
func TestManifestContract(t *testing.T) {
	good := `{
		"api_version": 1,
		"name": "sample",
		"version": "1.0.0",
		"kind": "question_type",
		"entrypoint": "sample.js"
	}`
	m, err := ParseManifest([]byte(good))
	if err != nil {
		t.Fatalf("good manifest rejected: %v", err)
	}
	if m.APIVersion != 1 || m.Name != "sample" || m.Kind != KindQuestionType {
		t.Fatalf("parsed wrong: %+v", m)
	}

	bad := `{"api_version": 2, "name": "x", "version": "1.0.0", "kind": "question_type", "entrypoint": "x.js"}`
	if _, err := ParseManifest([]byte(bad)); err == nil {
		t.Fatal("api_version 2 must be rejected (we only speak 1)")
	} else if want := "api_version"; !containsStr(err.Error(), want) {
		t.Fatalf("rejection must name api_version: %v", err)
	}

	missing := `{"api_version": 1, "name": "", "version": "1.0.0", "kind": "question_type", "entrypoint": "x.js"}`
	if _, err := ParseManifest([]byte(missing)); err == nil {
		t.Fatal("empty name must be rejected")
	}

	badKind := `{"api_version": 1, "name": "x", "version": "1.0.0", "kind": "nonsense", "entrypoint": "x.js"}`
	if _, err := ParseManifest([]byte(badKind)); err == nil {
		t.Fatal("unknown kind must be rejected")
	}

	badSemver := `{"api_version": 1, "name": "x", "version": "not-semver", "kind": "question_type", "entrypoint": "x.js"}`
	if _, err := ParseManifest([]byte(badSemver)); err == nil {
		t.Fatal("bad version must be rejected")
	}

	notJSON := `nope`
	if _, err := ParseManifest([]byte(notJSON)); err == nil {
		t.Fatal("non-JSON must be rejected")
	}
}

// The checked-in sample plugin must pass the contract check; CI fails
// if it does not. "The community will write integrations" is not an
// excuse for an unverified contract.
func TestSamplePluginPassesContract(t *testing.T) {
	dir := repoRoot(t)
	manifestPath := filepath.Join(dir, "testdata", "plugins", "sample", "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("sample manifest missing: %v", err)
	}
	m, err := ParseManifest(raw)
	if err != nil {
		t.Fatalf("sample manifest invalid: %v", err)
	}
	if err := m.Validate(filepath.Dir(manifestPath)); err != nil {
		t.Fatalf("sample plugin fails contract: %v", err)
	}
	entry := filepath.Join(filepath.Dir(manifestPath), m.Entrypoint)
	if _, err := os.Stat(entry); err != nil {
		t.Fatalf("sample entrypoint missing: %v", err)
	}
}

// A manifest whose entrypoint does not exist must fail the contract.
func TestContractRejectsMissingEntrypoint(t *testing.T) {
	m := Manifest{APIVersion: 1, Name: "x", Version: "1.0.0", Kind: KindQuestionType, Entrypoint: "ghost.js"}
	err := m.Validate(t.TempDir())
	if err == nil || !containsStr(err.Error(), "entrypoint") {
		t.Fatalf("must reject missing entrypoint naming it: %v", err)
	}
}

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

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && contains(s, sub))
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

var _ = json.Marshal
