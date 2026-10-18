// Package logic is Fieldkit's survey logic engine: question types,
// branching, required, hidden fields, and validation. It is a pure
// function of (survey definition, answers) — no I/O — so CI can run it
// against fixture files and the same code runs behind the submit
// endpoint and the offline `fieldkit logic` command.
package logic

import (
	"encoding/json"
	"fmt"
)

// Question types supported by the core engine.
const (
	TypeText         = "text"
	TypeSingleChoice = "single_choice"
	TypeMultiChoice  = "multi_choice"
	TypeRating       = "rating"
	TypeNPS          = "nps"
)

// Branch rule actions.
const (
	ActionShow = "show"
	ActionJump = "jump"
)

// Condition operators.
const (
	OpEq    = "eq"
	OpNe    = "ne"
	OpIn    = "in"
	OpGt    = "gt"
	OpLt    = "lt"
	OpGte   = "gte"
	OpLte   = "lte"
	OpEmpty = "empty"
)

// Question is one survey question.
type Question struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Title    string   `json:"title"`
	Required bool     `json:"required"`
	Hidden   bool     `json:"hidden"`
	Choices  []string `json:"choices,omitempty"`
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
}

// Condition is the "when" of a rule: question, operator, value.
type Condition struct {
	Question string `json:"question"`
	Op       string `json:"op"`
	Value    any    `json:"value"`
}

// Action is the "then" of a rule: show a hidden question, or jump to a
// question id.
type Action struct {
	Action   string `json:"action"`
	Question string `json:"question"`
}

// Rule is one branch rule.
type Rule struct {
	ID   string    `json:"id"`
	When Condition `json:"when"`
	Then Action    `json:"then"`
}

// Survey is a parsed, validated survey definition.
type Survey struct {
	Title     string     `json:"title"`
	Questions []Question `json:"questions"`
	Logic     []Rule     `json:"logic"`

	byID map[string]Question
}

// FieldError is a validation failure tied to one question.
type FieldError struct {
	Question string `json:"question"`
	Message  string `json:"message"`
}

// Result is the outcome of evaluating a survey against answers. It is
// the machine-stable CI contract: visible question IDs (not objects),
// per-question errors, and the jump target when one fires.
type Result struct {
	Visible []string     `json:"visible_questions"`
	Errors  []FieldError `json:"errors"`
	Next    string       `json:"next,omitempty"`
}

// VisibleQuestions returns the full question objects in survey order —
// for UIs that need titles/choices. CI and the CLI consume IDs.
func (r Result) VisibleQuestions(s *Survey) []Question {
	if s == nil {
		return nil
	}
	var out []Question
	for _, id := range r.Visible {
		for _, q := range s.Questions {
			if q.ID == id {
				out = append(out, q)
				break
			}
		}
	}
	return out
}

var knownTypes = map[string]bool{
	TypeText: true, TypeSingleChoice: true, TypeMultiChoice: true,
	TypeRating: true, TypeNPS: true,
}

var knownOps = map[string]bool{
	OpEq: true, OpNe: true, OpIn: true, OpGt: true, OpLt: true,
	OpGte: true, OpLte: true, OpEmpty: true,
}

var knownActions = map[string]bool{ActionShow: true, ActionJump: true}

// Parse decodes and validates a survey definition. Structural failures —
// unknown question types, choice questions without choices, unknown
// operators or actions, rules referencing questions that do not exist,
// backward jumps, and cyclic show-rules — are errors here, before any
// evaluation runs.
func Parse(raw []byte) (*Survey, error) {
	var s Survey
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("decode survey: %w", err)
	}
	if s.Title == "" {
		return nil, fmt.Errorf("survey title required")
	}
	if len(s.Questions) == 0 {
		return nil, fmt.Errorf("survey needs at least one question")
	}
	s.byID = make(map[string]Question, len(s.Questions))
	for _, q := range s.Questions {
		if q.ID == "" {
			return nil, fmt.Errorf("question id required")
		}
		if _, dup := s.byID[q.ID]; dup {
			return nil, fmt.Errorf("duplicate question id %q", q.ID)
		}
		if !knownTypes[q.Type] {
			return nil, fmt.Errorf("question %q: unknown type %q", q.ID, q.Type)
		}
		if q.Title == "" {
			return nil, fmt.Errorf("question %q: title required", q.ID)
		}
		switch q.Type {
		case TypeSingleChoice, TypeMultiChoice:
			if len(q.Choices) < 2 {
				return nil, fmt.Errorf("question %q: choice questions need at least 2 choices", q.ID)
			}
		case TypeNPS:
			// fixed 0..10 scale
		case TypeRating:
			// optional min/max; defaults applied at evaluation
		}
		s.byID[q.ID] = q
	}
	pos := make(map[string]int, len(s.Questions))
	for i, q := range s.Questions {
		pos[q.ID] = i
	}
	// show-rule edges for cycle detection
	edges := map[string][]string{}
	for _, r := range s.Logic {
		if r.ID == "" {
			return nil, fmt.Errorf("rule id required")
		}
		if !knownOps[r.When.Op] {
			return nil, fmt.Errorf("rule %q: unknown op %q", r.ID, r.When.Op)
		}
		if !knownActions[r.Then.Action] {
			return nil, fmt.Errorf("rule %q: unknown action %q", r.ID, r.Then.Action)
		}
		if _, ok := s.byID[r.When.Question]; !ok {
			return nil, fmt.Errorf("rule %q: when.question %q does not exist", r.ID, r.When.Question)
		}
		if _, ok := s.byID[r.Then.Question]; !ok {
			return nil, fmt.Errorf("rule %q: then.question %q does not exist", r.ID, r.Then.Question)
		}
		switch r.Then.Action {
		case ActionShow:
			edges[r.When.Question] = append(edges[r.When.Question], r.Then.Question)
		case ActionJump:
			if pos[r.Then.Question] <= pos[r.When.Question] {
				return nil, fmt.Errorf("rule %q: jump target %q must come after %q", r.ID, r.Then.Question, r.When.Question)
			}
		}
	}
	if cycle := findCycle(edges); cycle != "" {
		return nil, fmt.Errorf("show-rule cycle through question %q", cycle)
	}
	return &s, nil
}

// findCycle returns a question id on a cycle in the show-rule graph, or
// "" when acyclic.
func findCycle(edges map[string][]string) string {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := map[string]int{}
	var visit func(node string) string
	visit = func(node string) string {
		color[node] = gray
		for _, next := range edges[node] {
			switch color[next] {
			case gray:
				return next
			case white:
				if hit := visit(next); hit != "" {
					return hit
				}
			}
		}
		color[node] = black
		return ""
	}
	for node := range edges {
		if color[node] == white {
			if hit := visit(node); hit != "" {
				return hit
			}
		}
	}
	return ""
}

// Evaluate runs the logic engine: computes visible questions (hidden
// ones revealed by satisfied show rules, transitively), then validates
// answers for visible questions only. Hidden questions never block
// submission and their answers are ignored.
func Evaluate(s *Survey, answers map[string]any) Result {
	if s == nil {
		return Result{Errors: []FieldError{{Message: "nil survey"}}}
	}
	if answers == nil {
		answers = map[string]any{}
	}
	// Reveal fixpoint: apply show rules until no change. Parse rejects
	// cycles, so this terminates within len(rules) passes.
	visible := map[string]bool{}
	for _, q := range s.Questions {
		if !q.Hidden {
			visible[q.ID] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for _, r := range s.Logic {
			if r.Then.Action != ActionShow {
				continue
			}
			if visible[r.When.Question] && condMet(r.When, answers[r.When.Question]) && !visible[r.Then.Question] {
				visible[r.Then.Question] = true
				changed = true
			}
		}
	}
	var res Result
	for _, q := range s.Questions {
		if visible[q.ID] {
			res.Visible = append(res.Visible, q.ID)
		}
	}
	// First satisfied jump rule wins; targets are forward-only by parse.
	for _, r := range s.Logic {
		if r.Then.Action == ActionJump && visible[r.When.Question] && condMet(r.When, answers[r.When.Question]) {
			res.Next = r.Then.Question
			break
		}
	}
	for _, q := range s.Questions {
		if !visible[q.ID] {
			continue
		}
		if msg := validate(q, answers[q.ID]); msg != "" {
			res.Errors = append(res.Errors, FieldError{Question: q.ID, Message: msg})
		}
	}
	if res.Errors == nil {
		res.Errors = []FieldError{}
	}
	return res
}

// condMet evaluates a condition against one answer value.
func condMet(c Condition, v any) bool {
	switch c.Op {
	case OpEmpty:
		return v == nil || v == "" || isEmptyList(v)
	case OpEq:
		return eqAny(v, c.Value)
	case OpNe:
		return !eqAny(v, c.Value)
	case OpIn:
		list, ok := c.Value.([]any)
		if !ok {
			return false
		}
		for _, item := range list {
			if eqAny(v, item) {
				return true
			}
		}
		return false
	case OpGt, OpLt, OpGte, OpLte:
		a, aok := toFloat(v)
		b, bok := toFloat(c.Value)
		if !aok || !bok {
			return false
		}
		switch c.Op {
		case OpGt:
			return a > b
		case OpLt:
			return a < b
		case OpGte:
			return a >= b
		case OpLte:
			return a <= b
		}
	}
	return false
}

// validate checks one visible question's answer; "" means valid.
func validate(q Question, v any) string {
	empty := v == nil || v == "" || isEmptyList(v)
	if q.Required && empty {
		return "required"
	}
	if empty {
		return "" // optional and unanswered: fine
	}
	switch q.Type {
	case TypeText:
		if _, ok := v.(string); !ok {
			return "answer must be a string"
		}
	case TypeSingleChoice:
		s, ok := v.(string)
		if !ok {
			return "answer must be one of the choices"
		}
		if !inList(q.Choices, s) {
			return "answer must be one of the choices"
		}
	case TypeMultiChoice:
		list, ok := v.([]any)
		if !ok {
			return "answer must be a list of choices"
		}
		for _, item := range list {
			s, ok := item.(string)
			if !ok || !inList(q.Choices, s) {
				return "answer must be a list of choices"
			}
		}
	case TypeRating, TypeNPS:
		n, ok := toFloat(v)
		if !ok {
			return "answer must be a number"
		}
		lo, hi := 1.0, 5.0
		if q.Type == TypeNPS {
			lo, hi = 0, 10
		}
		if q.Min != nil {
			lo = *q.Min
		}
		if q.Max != nil {
			hi = *q.Max
		}
		if n < lo || n > hi {
			return fmt.Sprintf("answer must be between %v and %v", lo, hi)
		}
	}
	return ""
}

func inList(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

func isEmptyList(v any) bool {
	l, ok := v.([]any)
	return ok && len(l) == 0
}

// eqAny compares decoded-JSON values: numbers cross int/float, strings
// exact.
func eqAny(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if as, ok := a.(string); ok {
		bs, ok2 := b.(string)
		return ok2 && as == bs
	}
	af, aok := toFloat(a)
	bf, bok := toFloat(b)
	return aok && bok && af == bf
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}
