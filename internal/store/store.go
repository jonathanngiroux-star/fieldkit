// Package store is Fieldkit's SQLite persistence: surveys and responses.
// One file, one connection (SQLite is the serialization point), pure-Go
// driver — no cgo.
package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/jonathanngiroux-star/fieldkit/internal/logic"
)

// Store wraps the SQLite database file under a data directory.
type Store struct {
	db   *sql.DB
	path string
}

// SurveyInfo is a survey row without the full definition.
type SurveyInfo struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

// ResponseRow is one submitted response.
type ResponseRow struct {
	ID          int64
	SurveyID    int64
	Answers     map[string]any
	SubmittedAt time.Time
	Agent       string
}

// Open creates dir if needed, opens/creates the SQLite database, and
// applies the schema. The caller owns the returned Store.
func Open(dir string) (*Store, error) {
	if dir == "" {
		return nil, fmt.Errorf("data dir required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	path := filepath.Join(dir, "fieldkit.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// Single connection: SQLite serializes writers anyway; one conn makes
	// check-then-insert sequences atomic at the transaction level.
	db.SetMaxOpenConns(1)
	if err := migrateSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("schema: %w", err)
	}
	return &Store{db: db, path: path}, nil
}

func migrateSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS surveys (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			name          TEXT NOT NULL,
			definition    TEXT NOT NULL,
			created_nanos INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS responses (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			survey_id       INTEGER NOT NULL REFERENCES surveys(id),
			answers         TEXT NOT NULL,
			submitted_nanos INTEGER NOT NULL,
			agent           TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_responses_survey
			ON responses(survey_id, submitted_nanos)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("apply %q: %w", q[:30], err)
		}
	}
	return nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// Path is the on-disk database file (for diagnostics).
func (s *Store) Path() string { return s.path }

// CreateSurvey stores a survey definition under a human name. The
// definition is validated with the same logic engine that runs at
// submit time — a survey that cannot run never enters the DB. Returns
// the new survey id.
func (s *Store) CreateSurvey(name string, def map[string]any) (int64, error) {
	if name == "" {
		return 0, fmt.Errorf("survey name required")
	}
	if def == nil {
		return 0, fmt.Errorf("survey definition required")
	}
	raw, err := json.Marshal(def)
	if err != nil {
		return 0, fmt.Errorf("marshal definition: %w", err)
	}
	if _, err := logic.Parse(raw); err != nil {
		return 0, fmt.Errorf("invalid survey definition: %w", err)
	}
	res, err := s.db.Exec(
		`INSERT INTO surveys (name, definition, created_nanos) VALUES (?, ?, ?)`,
		name, string(raw), time.Now().UnixNano(),
	)
	if err != nil {
		return 0, fmt.Errorf("insert survey: %w", err)
	}
	return res.LastInsertId()
}

// ListSurveys returns all surveys, oldest first.
func (s *Store) ListSurveys() ([]SurveyInfo, error) {
	rows, err := s.db.Query(`SELECT id, name, created_nanos FROM surveys ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list surveys: %w", err)
	}
	defer rows.Close()
	var out []SurveyInfo
	for rows.Next() {
		var si SurveyInfo
		var nanos int64
		if err := rows.Scan(&si.ID, &si.Name, &nanos); err != nil {
			return nil, fmt.Errorf("scan survey: %w", err)
		}
		si.CreatedAt = time.Unix(0, nanos)
		out = append(out, si)
	}
	return out, rows.Err()
}

// SurveyJSON returns the canonical stored definition bytes.
func (s *Store) SurveyJSON(id int64) ([]byte, error) {
	var raw string
	err := s.db.QueryRow(`SELECT definition FROM surveys WHERE id = ?`, id).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("survey %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("survey json: %w", err)
	}
	return []byte(raw), nil
}

// DeleteSurvey removes a survey and its responses.
func (s *Store) DeleteSurvey(id int64) error {
	if _, err := s.db.Exec(`DELETE FROM responses WHERE survey_id = ?`, id); err != nil {
		return fmt.Errorf("delete responses: %w", err)
	}
	res, err := s.db.Exec(`DELETE FROM surveys WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete survey: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("survey %d not found", id)
	}
	return nil
}

// SaveResponse stores one response and returns the inserted response id.
// Answers are canonicalized to JSON.
func (s *Store) SaveResponse(surveyID int64, answers map[string]any, agent string) (int64, error) {
	if answers == nil {
		answers = map[string]any{}
	}
	raw, err := json.Marshal(answers)
	if err != nil {
		return 0, fmt.Errorf("marshal answers: %w", err)
	}
	var exists int
	err = s.db.QueryRow(`SELECT 1 FROM surveys WHERE id = ?`, surveyID).Scan(&exists)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("survey %d not found", surveyID)
	}
	if err != nil {
		return 0, fmt.Errorf("check survey: %w", err)
	}
	res, err := s.db.Exec(
		`INSERT INTO responses (survey_id, answers, submitted_nanos, agent)
		 VALUES (?, ?, ?, ?)`,
		surveyID, string(raw), time.Now().UnixNano(), agent,
	)
	if err != nil {
		return 0, fmt.Errorf("insert response: %w", err)
	}
	return res.LastInsertId()
}

// Responses returns a survey's responses, oldest first.
func (s *Store) Responses(surveyID int64) ([]ResponseRow, error) {
	rows, err := s.db.Query(
		`SELECT id, survey_id, answers, submitted_nanos, agent
		 FROM responses WHERE survey_id = ? ORDER BY submitted_nanos ASC, id ASC`,
		surveyID,
	)
	if err != nil {
		return nil, fmt.Errorf("responses: %w", err)
	}
	defer rows.Close()
	var out []ResponseRow
	for rows.Next() {
		var r ResponseRow
		var raw, agent string
		var nanos int64
		if err := rows.Scan(&r.ID, &r.SurveyID, &raw, &nanos, &agent); err != nil {
			return nil, fmt.Errorf("scan response: %w", err)
		}
		if err := json.Unmarshal([]byte(raw), &r.Answers); err != nil {
			return nil, fmt.Errorf("decode answers for response %d: %w", r.ID, err)
		}
		r.SubmittedAt = time.Unix(0, nanos)
		r.Agent = agent
		out = append(out, r)
	}
	return out, rows.Err()
}

// SurveyNames returns survey names sorted (diagnostics/TUI).
func SurveyNames(infos []SurveyInfo) []string {
	names := make([]string, len(infos))
	for i, si := range infos {
		names[i] = si.Name
	}
	sort.Strings(names)
	return names
}

// ExportCSV writes all responses for a survey as CSV: a header row
// (response_id, submitted_at, then question ids in survey order), one
// row per response. Values are CSV-quoted, and formula-injection
// prefixes (=, +, -, @) are defanged with a leading apostrophe so a
// spreadsheet cannot execute untrusted response content.
func (s *Store) ExportCSV(surveyID int64) (string, error) {
	def, err := s.SurveyJSON(surveyID)
	if err != nil {
		return "", err
	}
	var order []string
	var parsed struct {
		Questions []struct {
			ID string `json:"id"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(def, &parsed); err != nil {
		return "", fmt.Errorf("parse stored survey: %w", err)
	}
	for _, q := range parsed.Questions {
		order = append(order, q.ID)
	}
	rows, err := s.Responses(surveyID)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("response_id,submitted_at")
	for _, id := range order {
		b.WriteString(",")
		b.WriteString(csvCell(id))
	}
	b.WriteString("\n")
	for _, r := range rows {
		b.WriteString(strconv.FormatInt(r.ID, 10))
		b.WriteString(",")
		b.WriteString(r.SubmittedAt.UTC().Format(time.RFC3339Nano))
		for _, id := range order {
			b.WriteString(",")
			b.WriteString(csvCell(answerString(r.Answers[id])))
		}
		b.WriteString("\n")
	}
	return b.String(), nil
}

// answerString renders one answer value as CSV text: lists join with
// " | ", other scalars via fmt.
func answerString(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case []any:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = answerString(item)
		}
		return strings.Join(parts, " | ")
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// csvCell applies RFC 4180 quoting only when needed (comma, quote,
// newline) and defangs formula prefixes (=, +, -, @, tab) so spreadsheet
// apps cannot execute response content.
func csvCell(s string) string {
	if len(s) > 0 && strings.ContainsRune("=+-@	", rune(s[0])) {
		s = "'" + s
	}
	if strings.ContainsAny(s, ",\"\n\r") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
