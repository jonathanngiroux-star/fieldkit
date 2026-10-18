package serve

import (
	"net/http"
	"strings"
	"sync"
	"testing"
)

// AUDIT: concurrent submits must each report their OWN distinct response
// id (no max-readback race) and store exactly N rows.
func TestConcurrentSubmitsUniqueIDs(t *testing.T) {
	srv, st, id := newServer(t)
	const N = 25
	ids := make([]int64, N)
	codes := make([]int, N)
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, err := http.Post(srv.URL+"/surveys/1/submit", "application/json",
				strings.NewReader(`{"q1":"no","q3":"c"}`))
			if err != nil {
				t.Errorf("post: %v", err)
				return
			}
			defer resp.Body.Close()
			codes[i] = resp.StatusCode
			var out struct {
				ResponseID int64 `json:"response_id"`
			}
			if err := decodeJSON(resp.Body, &out); err != nil {
				t.Errorf("decode: %v", err)
				return
			}
			ids[i] = out.ResponseID
		}(i)
	}
	wg.Wait()
	seen := map[int64]bool{}
	for i, got := range ids {
		if codes[i] != http.StatusCreated {
			t.Fatalf("submit %d: status %d", i, codes[i])
		}
		if got == 0 || seen[got] {
			t.Fatalf("duplicate/zero response_id reported: %d", got)
		}
		seen[got] = true
	}
	rows, err := st.Responses(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != N {
		t.Fatalf("stored %d rows, want %d", len(rows), N)
	}
}

// AUDIT: after a 413 the server must keep serving — the rejected
// connection must not wedge the handler.
func TestServerHealthyAfterOversize(t *testing.T) {
	srv, _, _ := newServer(t)
	huge := `{"q1":"` + strings.Repeat("x", 2<<20) + `"}`
	resp, _ := postJSON(t, srv, "/surveys/1/submit", huge)
	resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("want 413, got %d", resp.StatusCode)
	}
	resp2, body := postJSON(t, srv, "/surveys/1/submit", `{"q1":"yes"}`)
	if resp2.StatusCode != http.StatusCreated {
		t.Fatalf("server unhealthy after 413: %d %s", resp2.StatusCode, body)
	}
}

// AUDIT (pin): survey definitions are served as application/json, never
// text/html — question titles are untrusted author input and must never
// render in an HTML context on this surface.
func TestSurveyViewIsJSONNotHTML(t *testing.T) {
	srv, _, _ := newServer(t)
	resp, err := http.Get(srv.URL + "/surveys/1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content-type %q, want application/json", ct)
	}
}

// AUDIT (pin): error bodies are JSON, not reflected HTML.
func TestErrorBodiesAreJSON(t *testing.T) {
	srv, _, _ := newServer(t)
	resp, body := postJSON(t, srv, "/surveys/999/submit", `{"q1":"yes"}`)
	if resp.StatusCode != 404 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if !strings.HasPrefix(strings.TrimSpace(body), "{") {
		t.Fatalf("error body must be JSON: %q", body)
	}
}
