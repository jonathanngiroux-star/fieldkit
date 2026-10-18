package serve

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jonathanngiroux-star/fieldkit/internal/store"
)

const branchFixture = `{
  "title": "Onboarding feedback",
  "questions": [
    {"id": "q1", "type": "single_choice", "title": "Did it work?", "required": true, "choices": ["yes","no"]},
    {"id": "q2", "type": "text", "title": "What failed?", "hidden": true},
    {"id": "q3", "type": "text", "title": "Anything else?"}
  ],
  "logic": [
    {"id": "br1", "when": {"question": "q1", "op": "eq", "value": "no"}, "then": {"action": "show", "question": "q2"}}
  ]
}`

func newServer(t *testing.T) (*httptest.Server, *store.Store, int64) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	var def map[string]any
	if err := json.Unmarshal([]byte(branchFixture), &def); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSurvey("fixture", def); err != nil {
		t.Fatal(err)
	}
	list, err := st.ListSurveys()
	if err != nil {
		t.Fatal(err)
	}
	h := New(st)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv, st, list[0].ID
}

func postJSON(t *testing.T, srv *httptest.Server, path, body string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Post(srv.URL+path, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

// Valid submit: 201, response id, stored in SQLite.
func TestSubmitStoresResponse(t *testing.T) {
	srv, st, id := newServer(t)
	resp, body := postJSON(t, srv, "/surveys/1/submit", `{"q1":"no","q2":"broken flow","q3":"ty"}`)
	if resp.StatusCode != 201 {
		t.Fatalf("status %d body %s", resp.StatusCode, body)
	}
	var out struct {
		ResponseID int64 `json:"response_id"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("body not json: %s", body)
	}
	if out.ResponseID == 0 {
		t.Fatalf("no response id: %s", body)
	}
	rows, err := st.Responses(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Answers["q2"] != "broken flow" {
		t.Fatalf("stored wrong: %+v", rows)
	}
}

// Required violation: 400 with a per-question error naming q1; nothing stored.
func TestSubmitValidationErrors(t *testing.T) {
	srv, st, id := newServer(t)
	resp, body := postJSON(t, srv, "/surveys/1/submit", `{"q3":"x"}`)
	if resp.StatusCode != 400 {
		t.Fatalf("status %d body %s", resp.StatusCode, body)
	}
	if !strings.Contains(body, `"q1"`) || !strings.Contains(body, "required") {
		t.Fatalf("error must name q1 required: %s", body)
	}
	rows, _ := st.Responses(id)
	if len(rows) != 0 {
		t.Fatalf("invalid submit must not store; got %d rows", len(rows))
	}
}

// Hidden question answered anyway: accepted, but hidden answers are
// dropped (they were never part of the visible contract).
func TestSubmitDropsHiddenAnswers(t *testing.T) {
	srv, st, id := newServer(t)
	resp, body := postJSON(t, srv, "/surveys/1/submit", `{"q1":"yes","q2":"sneaky","q3":"x"}`)
	if resp.StatusCode != 201 {
		t.Fatalf("status %d body %s", resp.StatusCode, body)
	}
	rows, err := st.Responses(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 stored row, got %d", len(rows))
	}
	if v, ok := rows[0].Answers["q2"]; ok {
		t.Fatalf("hidden answer must be dropped, stored %v", v)
	}
	if rows[0].Answers["q1"] != "yes" {
		t.Fatalf("visible answer must be stored: %+v", rows[0].Answers)
	}
}

// Unknown survey: 404, not a 500.
func TestSubmitUnknownSurvey(t *testing.T) {
	srv, _, _ := newServer(t)
	resp, body := postJSON(t, srv, "/surveys/999/submit", `{"q1":"yes"}`)
	if resp.StatusCode != 404 {
		t.Fatalf("status %d body %s", resp.StatusCode, body)
	}
}

// Malformed JSON: 400 with a parse message.
func TestSubmitMalformedJSON(t *testing.T) {
	srv, _, _ := newServer(t)
	resp, body := postJSON(t, srv, "/surveys/1/submit", `{not json`)
	if resp.StatusCode != 400 {
		t.Fatalf("status %d body %s", resp.StatusCode, body)
	}
	if !strings.Contains(strings.ToLower(body), "parse") && !strings.Contains(strings.ToLower(body), "invalid") {
		t.Fatalf("error should say parse: %s", body)
	}
}

// Oversized body: 413, not a silent 400.
func TestSubmitOversizedBody(t *testing.T) {
	srv, _, _ := newServer(t)
	huge := `{"q1":"` + strings.Repeat("x", 2<<20) + `"}`
	resp, _ := postJSON(t, srv, "/surveys/1/submit", huge)
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status %d, want 413", resp.StatusCode)
	}
}

// /donate serves an HTML page carrying both addresses exactly.
func TestDonatePage(t *testing.T) {
	srv, _, _ := newServer(t)
	resp, err := http.Get(srv.URL + "/donate")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	page := string(b)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if !strings.Contains(page, "0x85ee7E71f762d772599cbF1EC20E651B30657521") {
		t.Fatal("eth address missing from /donate")
	}
	if !strings.Contains(page, "bc1qxe2zx5tv3hdreaej6s2x4p7han85uey828rrhg") {
		t.Fatal("btc address missing from /donate")
	}
}

// Wrong content type: 415.
func TestSubmitWrongContentType(t *testing.T) {
	srv, _, _ := newServer(t)
	resp, err := http.Post(srv.URL+"/surveys/1/submit", "text/plain", bytes.NewReader([]byte(`{"q1":"yes"}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("status %d, want 415", resp.StatusCode)
	}
}
