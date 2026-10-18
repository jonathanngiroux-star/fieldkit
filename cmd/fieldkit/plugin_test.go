package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fieldkit plugin validate <dir-or-manifest>: the checked-in sample
// passes; a bad manifest fails with exit 1 naming the problem.
func TestPluginCommandValidate(t *testing.T) {
	root := repoRoot(t)
	sample := filepath.Join(root, "testdata", "plugins", "sample")

	var out, errb bytes.Buffer
	code := run([]string{"plugin", "validate", sample}, &out, &errb)
	if code != 0 {
		t.Fatalf("sample must pass: %d %s %s", code, out.String(), errb.String())
	}
	if !strings.Contains(out.String(), "ok") {
		t.Fatalf("expected ok: %q", out.String())
	}

	// Bad manifest in a temp dir.
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"api_version":9,"name":"bad","version":"1.0.0","kind":"question_type","entrypoint":"x.js"}`), 0o644)
	out.Reset()
	errb.Reset()
	code = run([]string{"plugin", "validate", dir}, &out, &errb)
	if code == 0 {
		t.Fatal("bad manifest must fail")
	}
	if !strings.Contains(errb.String(), "api_version") {
		t.Fatalf("error must name api_version: %q", errb.String())
	}
}
