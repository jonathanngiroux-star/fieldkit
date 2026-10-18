# Typeform migration fidelity

The dry-run report is the product. This document is the honest record of
what maps and what does not — no fake 100% coverage.

## What is covered

`fieldkit migrate --from=typeform --dry-run --format=json --input export.json`

reads a Typeform workspace export (the JSON "create form" payload shape:
`form`, `fields`, `logic`, `endings`) and produces:

- `survey` — a Fieldkit survey definition the same logic engine CI runs
  validates at conversion time
- `unmapped` — every Typeform field/rule that does not map, with a reason
- `endings` — ending screens, listed (not yet rendered)
- `source`, `dry_run` — fixed keys for machine consumers

## Field type mapping

| Typeform type | Fieldkit type | Notes |
|---|---|---|
| `short_text`, `long_text`, `email`, `website`, `date` | `text` | plain text answers |
| `opinion_scale`, `rating` | `rating` | `properties.steps` becomes `max` |
| `nps` | `nps` | 0–10 |
| `yes_no` | `single_choice` | choices `yes`/`no` |
| `multiple_choice` | `multi_choice` | choice labels carried over |
| `dropdown` | `single_choice` | choice labels carried over |
| `number` | `rating` | numeric answer in a scale band |
| `statement` | — unmapped | no fieldkit equivalent yet |
| `matrix` | — unmapped | the documented plugin-contract use case |
| `hidden` | — listed | pass-through metadata, not a respondent question |
| `group`, `payment`, `file_url`, `legal` | — unmapped | no equivalent yet |

## Logic mapping

Field-level `jump` rules with an `equal` condition map to Fieldkit `jump`
rules (`when {question, op: eq, value}` → `then {action: jump}`). Other
Typeform rule types (`hidden` field assignment logic, calculations) are
listed as unmapped rather than approximated.

## Verified against

- The committed fixture: `testdata/fixtures/typeform/sample-export.json`
  (7 fields, 1 jump rule, 1 ending).
- CI gate: the JSON report on that fixture must keep
  `source`, `dry_run`, 4 mapped questions, 3 unmapped items, 1 ending.
- Fuzz: the converter must never panic on arbitrary input
  (`FuzzConvert`, 15s per CI run).

## Not covered yet (listed, never silently dropped)

- Ending screens as rendered content
- `matrix` questions — the sample plugin demonstrates the question_type
  plugin contract for exactly this case
- Recall links / response data import (dry-run covers the form only)
- Qualtrics — gated on this table being green first
