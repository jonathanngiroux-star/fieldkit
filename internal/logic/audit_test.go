package logic

import (
	"encoding/json"
	"testing"
)

// AUDIT: a hidden question no rule can ever reveal is an authoring bug
// the parser must catch — the question is dead state in the survey.
func TestParseRejectsOrphanHidden(t *testing.T) {
	raw := `{
		"title":"t",
		"questions":[
			{"id":"q1","type":"text","title":"x"},
			{"id":"q2","type":"text","title":"orphan","hidden":true}
		],
		"logic":[]
	}`
	if _, err := Parse([]byte(raw)); err == nil {
		t.Fatal("orphan hidden question must be a parse error")
	}
}

// AUDIT: operators beyond eq must work — the typeform converter and
// hand-authored surveys both need them.
func TestOperators(t *testing.T) {
	raw := `{
		"title":"ops",
		"questions":[
			{"id":"score","type":"rating","title":"score","max":10},
			{"id":"comment","type":"text","title":"comment","hidden":true}
		],
		"logic":[
			{"id":"r1","when":{"question":"score","op":"gt","value":7},"then":{"action":"show","question":"comment"}}
		]
	}`
	s, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	res := Evaluate(s, map[string]any{"score": 8.0})
	if len(res.Visible) != 2 {
		t.Fatalf("gt 8 must reveal comment: %v", res.Visible)
	}
	res = Evaluate(s, map[string]any{"score": 3.0})
	if len(res.Visible) != 1 {
		t.Fatalf("3 must not reveal comment: %v", res.Visible)
	}

	// empty op
	raw2 := `{
		"title":"ops2",
		"questions":[
			{"id":"a","type":"text","title":"a"},
			{"id":"b","type":"text","title":"b","hidden":true}
		],
		"logic":[
			{"id":"r1","when":{"question":"a","op":"empty","value":null},"then":{"action":"show","question":"b"}}
		]
	}`
	s2, err := Parse([]byte(raw2))
	if err != nil {
		t.Fatal(err)
	}
	if res := Evaluate(s2, map[string]any{}); len(res.Visible) != 2 {
		t.Fatalf("empty must reveal b: %v", res.Visible)
	}
	if res := Evaluate(s2, map[string]any{"a": "filled"}); len(res.Visible) != 1 {
		t.Fatalf("filled must not reveal b: %v", res.Visible)
	}
}

// AUDIT: chain of shows must be transitive (q1 shows q2, q2 shows q3 —
// answering q1 reveals both).
func TestShowChainsTransitive(t *testing.T) {
	raw := `{
		"title":"chain",
		"questions":[
			{"id":"a","type":"single_choice","title":"a","choices":["x","y"]},
			{"id":"b","type":"text","title":"b","hidden":true},
			{"id":"c","type":"text","title":"c","hidden":true}
		],
		"logic":[
			{"id":"r1","when":{"question":"a","op":"eq","value":"x"},"then":{"action":"show","question":"b"}},
			{"id":"r2","when":{"question":"b","op":"eq","value":"go"},"then":{"action":"show","question":"c"}}
		]
	}`
	s, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	// a=x, b unanswered: c must NOT be visible (b's condition not met)
	res := Evaluate(s, map[string]any{"a": "x"})
	if len(res.Visible) != 2 {
		t.Fatalf("a=x alone must reveal only b: %v", res.Visible)
	}
	// a=x, b=go: both revealed
	res = Evaluate(s, map[string]any{"a": "x", "b": "go"})
	if len(res.Visible) != 3 {
		t.Fatalf("chain must reveal c: %v", res.Visible)
	}
}

// AUDIT: multi_choice validation rejects values outside the choice list.
func TestMultiChoiceValidation(t *testing.T) {
	raw := `{
		"title":"m",
		"questions":[
			{"id":"m1","type":"multi_choice","title":"pick","required":true,"choices":["a","b"]}
		]
	}`
	s, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	res := Evaluate(s, map[string]any{"m1": []any{"a", "zzz"}})
	if len(res.Errors) != 1 || res.Errors[0].Question != "m1" {
		t.Fatalf("out-of-list choice must error: %+v", res.Errors)
	}
	res = Evaluate(s, map[string]any{"m1": []any{"a", "b"}})
	if len(res.Errors) != 0 {
		t.Fatalf("valid choices must pass: %+v", res.Errors)
	}
}

// AUDIT: Parse must never panic on arbitrary bytes (fuzz pin).
func FuzzParse(f *testing.F) {
	f.Add([]byte(`{"title":"t","questions":[{"id":"q","type":"text","title":"x"}]}`))
	f.Add([]byte(`{"title":"t","questions":[],"logic":null}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`{`))
	f.Fuzz(func(t *testing.T, data []byte) {
		s, err := Parse(data)
		if err != nil {
			return
		}
		if s == nil {
			t.Fatal("nil survey, nil error")
		}
		// evaluation over arbitrary answers must also never panic
		_ = Evaluate(s, map[string]any{"anything": "value", "num": 1.5})
	})
}

// AUDIT: Parse rejects min > max on numeric questions.
func TestParseRejectsInvertedRange(t *testing.T) {
	raw := `{"title":"t","questions":[{"id":"r1","type":"rating","title":"x","min":10,"max":1}]}`
	if _, err := Parse([]byte(raw)); err == nil {
		t.Fatal("inverted range must be rejected")
	}
	_ = json.Marshal
}
