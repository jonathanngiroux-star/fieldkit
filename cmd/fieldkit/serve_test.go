package main

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonathanngiroux-star/fieldkit/internal/store"
)

// serve + export end to end: create survey via CLI, POST a response
// against the started server, export CSV from the CLI, all against the
// same data dir.
func TestServeAndExportCSV(t *testing.T) {
	root := repoRoot(t)
	_ = root
	data := t.TempDir()
	surveyPath := filepath.Join(data, "survey.json")
	os.WriteFile(surveyPath, []byte(`{"title":"E2E","questions":[{"id":"q1","type":"text","title":"Name","required":true}]}`), 0o644)

	var out, errb bytes.Buffer
	if code := run([]string{"surveys", "create", "--file", surveyPath, "--data-dir", data}, &out, &errb); code != 0 {
		t.Fatalf("create: %d %s", code, errb.String())
	}
	out.Reset()
	errb.Reset()
	if code := run([]string{"surveys", "list", "--data-dir", data}, &out, &errb); code != 0 {
		t.Fatalf("list: %d", code)
	}
	if !strings.Contains(out.String(), "1\t") {
		t.Fatalf("survey 1 missing from list: %q", out.String())
	}

	// Start the server on an ephemeral port, in-process.
	var sout, serr bytes.Buffer
	st, err := store.Open(data)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	stop, port, err := startServe(st, &sout, &serr)
	if err != nil {
		t.Fatalf("start server: %v (stderr: %s)", err, serr.String())
	}
	defer stop()

	resp, err := http.Post(
		"http://127.0.0.1:"+port+"/surveys/1/submit",
		"application/json",
		strings.NewReader(`{"q1":"from CLI e2e"}`),
	)
	if err != nil {
		t.Fatalf("submit: %v (server stderr: %s)", err, serr.String())
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("submit status %d body %s (server: %s)", resp.StatusCode, body, serr.String())
	}

	// CSV export from the CLI over the same data dir.
	out.Reset()
	errb.Reset()
	if code := run([]string{"export", "csv", "--survey", "1", "--data-dir", data}, &out, &errb); code != 0 {
		t.Fatalf("export: %d %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "from CLI e2e") {
		t.Fatalf("csv missing response: %q", out.String())
	}
	if !strings.HasPrefix(out.String(), "response_id,submitted_at,q1") {
		t.Fatalf("csv header: %q", out.String())
	}
}
