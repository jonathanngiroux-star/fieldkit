// Package plugin is Fieldkit's versioned plugin contract: a JSON
// manifest plus an entrypoint file. api_version 1 is the only supported
// ABI; anything else is rejected. The sample plugin in
// testdata/plugins/sample must pass the contract check in CI — a
// plugin contract nobody exercises is a contract nobody has.
package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// APIVersion is the only ABI this build speaks.
const APIVersion = 1

// Plugin kinds recognized by the core host.
const (
	KindQuestionType = "question_type"
	KindWebhook      = "webhook"
	KindScorer       = "scorer"
)

// Manifest is a plugin manifest (api_version 1).
type Manifest struct {
	APIVersion  int    `json:"api_version"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Kind        string `json:"kind"`
	Description string `json:"description,omitempty"`
	Entrypoint  string `json:"entrypoint"`
}

var knownKinds = map[string]bool{
	KindQuestionType: true, KindWebhook: true, KindScorer: true,
}

// ParseManifest decodes and structurally validates a manifest.
func ParseManifest(raw []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("manifest is not valid JSON: %w", err)
	}
	if m.APIVersion != APIVersion {
		return nil, fmt.Errorf("manifest api_version %d unsupported: this fieldkit speaks api_version %d only", m.APIVersion, APIVersion)
	}
	if strings.TrimSpace(m.Name) == "" {
		return nil, fmt.Errorf("manifest name required")
	}
	if !validName(m.Name) {
		return nil, fmt.Errorf("manifest name %q must be lowercase letters, digits, dashes", m.Name)
	}
	if !semverish(m.Version) {
		return nil, fmt.Errorf("manifest version %q must be MAJOR.MINOR.PATCH", m.Version)
	}
	if !knownKinds[m.Kind] {
		return nil, fmt.Errorf("manifest kind %q unknown (want one of question_type, webhook, scorer)", m.Kind)
	}
	if strings.TrimSpace(m.Entrypoint) == "" {
		return nil, fmt.Errorf("manifest entrypoint required")
	}
	if filepath.IsAbs(m.Entrypoint) || strings.Contains(m.Entrypoint, "..") {
		return nil, fmt.Errorf("entrypoint %q must be a relative path inside the plugin dir", m.Entrypoint)
	}
	return &m, nil
}

// Validate checks the contract against the plugin's directory: the
// entrypoint file must exist and be non-empty.
func (m *Manifest) Validate(dir string) error {
	if m == nil {
		return fmt.Errorf("nil manifest")
	}
	entry := filepath.Join(dir, m.Entrypoint)
	info, err := os.Stat(entry)
	if err != nil {
		return fmt.Errorf("entrypoint %s: %w", m.Entrypoint, err)
	}
	if info.IsDir() {
		return fmt.Errorf("entrypoint %s is a directory", m.Entrypoint)
	}
	if info.Size() == 0 {
		return fmt.Errorf("entrypoint %s is empty", m.Entrypoint)
	}
	return nil
}

func validName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
		default:
			return false
		}
	}
	return true
}

// semverish enforces MAJOR.MINOR.PATCH (no build metadata).
func semverish(v string) bool {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}
