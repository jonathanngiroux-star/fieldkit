// Package typeform converts a Typeform export into a Fieldkit survey
// definition and a dry-run migration report. The report is the product:
// machine-readable, honest about what does not map, stable in shape for
// CI.
package typeform

import (
	"encoding/json"
	"fmt"

	"github.com/jonathanngiroux-star/fieldkit/internal/logic"
)

// Report is the dry-run migration report (the machine-stable contract).
type Report struct {
	Source   string       `json:"source"`
	DryRun   bool         `json:"dry_run"`
	Survey   logic.Survey `json:"survey"`
	Unmapped []Unmapped   `json:"unmapped"`
	Endings  []Ending     `json:"endings"`
	Risks    []Risk       `json:"risks,omitempty"`
}

// Unmapped is a field Typeform has that Fieldkit does not (yet) map —
// listed honestly, never silently dropped.
type Unmapped struct {
	Key  string `json:"key"`
	Ref  string `json:"ref,omitempty"`
	Type string `json:"type"`
	Why  string `json:"why"`
}

// Ending is a Typeform ending screen.
type Ending struct {
	Ref  string `json:"ref"`
	Type string `json:"type"`
}

// Risk is a migration risk the operator must judge (not silently decide).
type Risk struct {
	Key    string `json:"key"`
	Detail string `json:"detail"`
}

// tfExport mirrors the subset of Typeform's export schema we read.
type tfExport struct {
	Form struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"form"`
	Fields []struct {
		ID         string `json:"id"`
		Ref        string `json:"ref"`
		Title      string `json:"title"`
		Type       string `json:"type"`
		Properties struct {
			Required bool `json:"required"`
			Steps    *int `json:"steps"`
			Choices  []struct {
				ID    string `json:"id"`
				Label string `json:"label"`
				Ref   string `json:"ref"`
			} `json:"choices"`
			Rows    []string `json:"rows"`
			Columns []string `json:"columns"`
		} `json:"properties"`
		Validations struct {
			Required bool `json:"required"`
		} `json:"validations"`
	} `json:"fields"`
	Logic []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		From       string `json:"from"`
		Conditions struct {
			Op   string `json:"op"`
			Vars []struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"vars"`
		} `json:"conditions"`
		Actions []struct {
			Type string `json:"type"`
			To   string `json:"to"`
		} `json:"actions"`
	} `json:"logic"`
	Endings []struct {
		ID   string `json:"id"`
		Ref  string `json:"ref"`
		Type string `json:"type"`
	} `json:"endings"`
}

// typeMap: Typeform field type -> Fieldkit question type ("" = unmapped).
var typeMap = map[string]string{
	"short_text":      logic.TypeText,
	"long_text":       logic.TypeText,
	"opinion_scale":   logic.TypeRating,
	"rating":          logic.TypeRating,
	"nps":             logic.TypeNPS,
	"yes_no":          logic.TypeSingleChoice,
	"multiple_choice": logic.TypeMultiChoice,
	"dropdown":        logic.TypeSingleChoice,
	"email":           logic.TypeText,
	"website":         logic.TypeText,
	"number":          logic.TypeRating,
	"date":            logic.TypeText,
}

// Convert parses a Typeform export and builds the dry-run report.
func Convert(raw []byte) (*Report, error) {
	var tf tfExport
	if err := json.Unmarshal(raw, &tf); err != nil {
		return nil, fmt.Errorf("parse typeform export: %w", err)
	}
	if tf.Form.ID == "" && tf.Form.Title == "" && len(tf.Fields) == 0 {
		return nil, fmt.Errorf("not a typeform export: no form, fields")
	}
	rep := &Report{Source: "typeform", DryRun: true}
	s := logic.Survey{Title: tf.Form.Title}
	if s.Title == "" {
		s.Title = tf.Form.ID
	}
	refs := map[string]bool{}
	for _, f := range tf.Fields {
		if f.Ref != "" {
			refs[f.Ref] = true
		}
	}
	for _, f := range tf.Fields {
		switch f.Type {
		case "statement", "matrix", "group", "payment", "file_url", "legal":
			why := "no fieldkit equivalent yet"
			if f.Type == "matrix" {
				why = "matrix becomes a plugin question type; core maps single-dimension scales only"
			}
			rep.Unmapped = append(rep.Unmapped, Unmapped{Key: f.ID, Ref: f.Ref, Type: f.Type, Why: why})
			continue
		case "hidden":
			// Hidden fields are metadata, not respondent questions. Keep
			// them visible in the report so nothing is silently dropped,
			// but they are not mapped into the survey questions.
			rep.Unmapped = append(rep.Unmapped, Unmapped{
				Key: f.ID, Ref: f.Ref, Type: f.Type,
				Why: "hidden field captured as pass-through metadata; not a respondent question",
			})
			continue
		}
		qType, ok := typeMap[f.Type]
		if !ok {
			rep.Unmapped = append(rep.Unmapped, Unmapped{Key: f.ID, Ref: f.Ref, Type: f.Type, Why: "unknown typeform field type"})
			continue
		}
		id := f.Ref
		if id == "" {
			id = f.ID
		}
		q := logic.Question{ID: id, Type: qType, Title: f.Title, Required: f.Validations.Required || f.Properties.Required}
		switch qType {
		case logic.TypeRating:
			if f.Properties.Steps != nil {
				max := float64(*f.Properties.Steps)
				q.Max = &max
			}
		case logic.TypeSingleChoice, logic.TypeMultiChoice:
			for _, c := range f.Properties.Choices {
				q.Choices = append(q.Choices, c.Label)
			}
			if len(q.Choices) < 2 && f.Type == "yes_no" {
				q.Choices = []string{"yes", "no"}
			}
		}
		s.Questions = append(s.Questions, q)
	}
	// logic: map field-level jump rules only; unsupported rule types are
	// listed as unmapped.
	for _, r := range tf.Logic {
		if r.Type != "field" {
			rep.Unmapped = append(rep.Unmapped, Unmapped{Key: r.ID, Type: "logic:" + r.Type, Why: "only field-level jump rules map"})
			continue
		}
		var condVal string
		for _, v := range r.Conditions.Vars {
			if v.Type == "constant" {
				condVal = v.Value
			}
		}
		for _, a := range r.Actions {
			if a.Type != "jump" {
				rep.Unmapped = append(rep.Unmapped, Unmapped{Key: r.ID, Type: "logic_action:" + a.Type, Why: "only jump actions map"})
				continue
			}
			s.Logic = append(s.Logic, logic.Rule{
				ID:   r.ID,
				When: logic.Condition{Question: r.From, Op: logic.OpEq, Value: condVal},
				Then: logic.Action{Action: logic.ActionJump, Question: a.To},
			})
		}
	}
	for _, e := range tf.Endings {
		rep.Endings = append(rep.Endings, Ending{Ref: e.Ref, Type: e.Type})
	}
	// Validate the mapped survey with the same engine CI uses — a
	// conversion that produces an invalid fieldkit survey is a bug, not
	// an "unmapped" note.
	rawSurvey, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	if _, err := logic.Parse(rawSurvey); err != nil {
		return nil, fmt.Errorf("converted survey is invalid: %w", err)
	}
	rep.Survey = s
	return rep, nil
}
