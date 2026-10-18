# Backend audit (step 2)

Full audit pass over the backend. Bugs found were fixed in this pass,
not ticketed. Each fix carries a named proof test.

## Bugs found and fixed

1. **Duplicate response ids under concurrency** — `submit` reported
   `response_id` by re-reading all rows and taking the max: two
   concurrent submits could report the same id, and the readback was
   O(n) per submit. `SaveResponse` now returns the inserted id
   (`LastInsertId`) and the handler reports that. Proof:
   `TestConcurrentSubmitsUniqueIDs` (25 parallel submits → 25 distinct
   reported ids, 25 stored rows) and `TestSaveResponseReturnsID`.
2. **Invalid surveys could enter the DB** — `CreateSurvey` stored any
   JSON, so a definition the logic engine would reject turned every
   later submit into a 500. Create now runs `logic.Parse` at the write
   boundary. Proof: `TestCreateSurveyRejectsInvalidDefinition` (bad
   type named in the error; DB stays empty).
3. **`surveys create` id-by-name lookup** — the CLI recovered the new
   survey's id by listing and matching on name: wrong id on name
   collision. `CreateSurvey` returns the id directly.
4. **Orphan hidden questions passed parse** — a hidden question no
   show rule could ever reveal was dead authoring state that silently
   never appeared. Parse now rejects it. Proof:
   `TestParseRejectsOrphanHidden`.
5. **Inverted numeric ranges passed parse** — `min > max` on rating/NPS
   questions made every answer invalid at runtime. Parse now rejects.
   Proof: `TestParseRejectsInvertedRange`.
6. **Operators beyond `eq` were dead code** — `gt/lt/gte/lte/empty/ne/in`
   had no tests; `empty` was needed by real surveys. Proof:
   `TestOperators`, `TestShowChainsTransitive` (fixpoint chains),
   `TestMultiChoiceValidation` (out-of-list multi-choice answers
   rejected).

## Pins (behavior verified as already correct, now locked by tests)

- **CSV never leaks the agent column** — `TestCSVExcludesAgent`
  (user-agent strings never reach exports).
- **Oversize body does not wedge the server** — `TestServerHealthyAfterOversize`
  (413 then a clean 201).
- **Survey definitions served as `application/json` only** —
  `TestSurveyViewIsJSONNotHTML`; question titles are untrusted author
  input and never render in an HTML context on this surface.
- **Error bodies are JSON** — `TestErrorBodiesAreJSON`.
- **Formula-injection defanging in CSV** — `'=SUM(...)` prefix (from
  step 1, re-verified).
- **Concurrent identical writes** — 50 parallel `SaveResponse` calls →
  50 distinct ids, 50 rows: `TestConcurrentSaveResponse` (race
  detector clean).

## Fuzz

- `FuzzParse` (logic): arbitrary bytes → parse+evaluate, no panics.
  15s in CI-shape runs, zero crashes.
- `FuzzConvert` (typeform): arbitrary bytes → convert, no panics.
  1.6M+ execs across runs, zero crashes.

## Migrate golden stability

The report contract (`source`, `dry_run`, 4 mapped questions, 3
unmapped, 1 ending, 1 logic rule) is asserted in-package
(`TestDryRunReportContract`, `TestReportJSONShape`) and gated in CI
with `jq` on the committed fixture — drift fails the build.

## Gates re-run fresh at commit time

gofmt clean · vet clean · `go test ./...` 6/6 green ·
`go test -race` on store/serve/logic green · fuzz targets clean ·
migrate/plugin/donate/logic CI gates green on a freshly built binary.
