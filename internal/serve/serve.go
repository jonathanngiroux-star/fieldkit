// Package serve is Fieldkit's HTTP surface: a public submit endpoint
// (no account, no auth — surveys are addressed by id), survey metadata
// for respondents, and the /donate page. One handler, no framework.
package serve

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/jonathanngiroux-star/fieldkit/internal/logic"
	"github.com/jonathanngiroux-star/fieldkit/internal/store"
)

// maxBodyBytes is the submit-payload cap: 1 MiB.
const maxBodyBytes = 1 << 20

// Donation addresses — binding. Changes require the GOVERNANCE.md
// process (unanimous maintainer consent + 30-day public notice).
const (
	donateEthereum = "0x85ee7E71f762d772599cbF1EC20E651B30657521"
	donateBitcoin  = "bc1qxe2zx5tv3hdreaej6s2x4p7han85uey828rrhg"
)

// Handler serves all Fieldkit HTTP routes.
type Handler struct {
	store *store.Store
	mux   *http.ServeMux
}

// New builds the handler over an open store.
func New(st *store.Store) *Handler {
	h := &Handler{store: st, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /surveys/{id}/submit", h.submit)
	h.mux.HandleFunc("GET /surveys/{id}", h.surveyView)
	h.mux.HandleFunc("GET /donate", h.donate)
	h.mux.HandleFunc("GET /health", h.health)
	return h
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

// submit is the public response endpoint. Validation errors are 400 with
// a per-question list; oversized bodies are 413; wrong content type 415;
// unknown survey 404. Hidden questions' answers are dropped — they were
// never part of the visible contract at submit time.
func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusNotFound, "survey not found")
		return
	}
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		writeErr(w, http.StatusUnsupportedMediaType, "content-type must be application/json")
		return
	}
	body := http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var answers map[string]any
	dec := json.NewDecoder(body)
	if err := dec.Decode(&answers); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeErr(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		writeErr(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	def, err := h.store.SurveyJSON(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "survey not found")
		return
	}
	surv, err := logic.Parse(def)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "stored survey is invalid: "+err.Error())
		return
	}
	res := logic.Evaluate(surv, answers)
	if len(res.Errors) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"errors": res.Errors})
		return
	}
	// Drop answers to questions that are not visible.
	keep := map[string]any{}
	for _, qid := range res.Visible {
		if v, ok := answers[qid]; ok {
			keep[qid] = v
		}
	}
	agent := r.Header.Get("User-Agent")
	if len(agent) > 200 {
		agent = agent[:200]
	}
	if err := h.store.SaveResponse(id, keep, agent); err != nil {
		writeErr(w, http.StatusInternalServerError, "store response: "+err.Error())
		return
	}
	var maxID int64
	rows, err := h.store.Responses(id)
	if err == nil && len(rows) > 0 {
		maxID = rows[len(rows)-1].ID
	}
	writeJSON(w, http.StatusCreated, map[string]any{"response_id": maxID})
}

// surveyView returns the respondent-visible survey: questions with
// titles, choices, required, hidden (no stored logic internals beyond
// what a respondent's client needs to run the survey).
func (h *Handler) surveyView(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusNotFound, "survey not found")
		return
	}
	def, err := h.store.SurveyJSON(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "survey not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(def)
}

var donatePage = template.Must(template.New("donate").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Fieldkit — Donate</title>
<style>
body{font-family:system-ui,sans-serif;margin:0;display:flex;justify-content:center;background:#f6f7f9}
.card{max-width:36rem;margin:4rem 1rem;padding:2rem;background:#fff;border-radius:8px}
code{display:block;word-break:break-all;padding:.6rem;background:#eef0f4;border-radius:6px;font-size:.95rem}
.foot{margin-top:2rem;color:#555;font-size:.9rem}
</style>
</head>
<body>
<main class="card">
<h1>Donate to Fieldkit</h1>
<p>Fieldkit is free to self-host. There is no paid tier, no subscription,
no invoice, and no feature gated on money — there never will be. If it
replaces a paid survey-platform line item for you, donations are welcome.</p>
<h2>Ethereum / USDC (ERC-20)</h2>
<code>{{.Eth}}</code>
<h2>Bitcoin</h2>
<code>{{.Btc}}</code>
<p class="foot">No other wallet addresses are associated with this project.
Wallet or license changes follow GOVERNANCE.md (unanimous maintainer consent
+ 30 days of public notice). <code>fieldkit donate</code> prints both addresses.</p>
</main>
</body>
</html>
`))

func (h *Handler) donate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	donatePage.Execute(w, map[string]string{"Eth": donateEthereum, "Btc": donateBitcoin})
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
