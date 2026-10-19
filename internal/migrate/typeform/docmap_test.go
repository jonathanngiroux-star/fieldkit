package typeform

import (
	"encoding/json"
	"testing"
)

// TestDocMappingTablePins pins every row of the docs/fidelity/typeform.md
// mapping table to actual converter behavior — the doc is a contract.
func TestDocMappingTablePins(t *testing.T) {
	cases := []struct {
		tfType   string
		wantType string // "" = must be unmapped/listed
	}{
		{"short_text", "text"},
		{"long_text", "text"},
		{"email", "text"},
		{"website", "text"},
		{"date", "text"},
		{"opinion_scale", "rating"},
		{"rating", "rating"},
		{"nps", "nps"},
		{"yes_no", "single_choice"},
		{"multiple_choice", "multi_choice"},
		{"dropdown", "single_choice"},
		{"number", "rating"},
		{"statement", ""},
		{"matrix", ""},
		{"hidden", ""},
		{"group", ""},
		{"payment", ""},
		{"file_url", ""},
		{"legal", ""},
	}
	for _, c := range cases {
		props := `{}`
		switch c.tfType {
		case "yes_no", "multiple_choice", "dropdown":
			props = `{"choices":[{"id":"c1","label":"a","ref":"c1"},{"id":"c2","label":"b","ref":"c2"}]}`
		case "opinion_scale", "rating", "number":
			props = `{"steps":5}`
		}
		// every export carries a mapped text companion so single-unmapped
		// cases still form a valid survey (the converter validates output)
		export := `{"form":{"id":"x","title":"t"},"fields":[{"id":"f0","ref":"r0","title":"Seed","type":"short_text","properties":{}},{"id":"f1","ref":"r1","title":"Q","type":"` + c.tfType + `","properties":` + props + `}]}`
		rep, err := Convert([]byte(export))
		if err != nil {
			t.Fatalf("%s: convert: %v", c.tfType, err)
		}
		var out struct {
			Survey struct {
				Questions []struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				} `json:"questions"`
			} `json:"survey"`
			Unmapped []struct {
				Key string `json:"key"`
			} `json:"unmapped"`
		}
		raw, _ := json.Marshal(rep)
		json.Unmarshal(raw, &out)
		if c.wantType == "" {
			// f1 must be listed as unmapped and produce NO question with ref r1
			found := false
			for _, u := range out.Unmapped {
				if u.Key == "f1" {
					found = true
				}
			}
			mapped := false
			for _, q := range out.Survey.Questions {
				if q.ID == "r1" {
					mapped = true
				}
			}
			if mapped {
				t.Errorf("%s: doc says unmapped, but converter mapped it", c.tfType)
			}
			if !found {
				t.Errorf("%s: doc says unmapped, but not listed in unmapped", c.tfType)
			}
			continue
		}
		if len(out.Survey.Questions) != 2 || out.Survey.Questions[1].Type != c.wantType {
			t.Errorf("%s: got %d question(s), last type=%q, want %q", c.tfType, len(out.Survey.Questions),
				func() string {
					if len(out.Survey.Questions) > 0 {
						return out.Survey.Questions[len(out.Survey.Questions)-1].Type
					}
					return ""
				}(), c.wantType)
		}
	}
}
