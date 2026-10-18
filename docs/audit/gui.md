# GUI audit (step 4)

Full audit of the Wails desktop GUI via the in-app test bridge
(`cmd/fieldkit/gui_driver.py` + `gui_suite_audit.py`, run on a fresh
data dir). Bugs found were fixed in this pass, not ticketed.

## Suites

- `gui_suite_full.py` — first-run wizard walk (all 6 steps with real
  operations), every tab, donate byte-for-byte, wizard reopen + keyboard
  path, sqlite cross-verification of every persistence claim.
- `gui_suite_audit.py` — step-4 checklist: empty/error/loading states,
  Back/Next, author error display, branch both directions, migrate
  unmapped specificity, plugin bad-manifest message, secrets scan,
  CLI/GUI migrate agreement.

## Bugs found and fixed (during step 3 build + this audit pass)

1. **`wizard.js` referenced `$$` outside its scope** — `app.js`'s
   selector helpers are IIFE-private; `collect()` threw
   `ReferenceError: Can't find variable: $$`, so every wizard step
   silently no-advanced. Fixed: local `$$` in wizard.js. Found by the
   bridge suite (the reference error appeared in `#wizard-result`).
2. **Stale main view after the wizard closed** — the survey list only
   refreshed on tab activation, so finishing the wizard landed on an
   empty main view with 2 surveys in the DB. Fixed:
   `window.fieldkitRefresh` hook called by the wizard's `close()`.
3. **Respondent form field count double-counted** (audit-suite
   selector bug, not an app bug): `[data-qid]` matched both the wrapper
   div and the input. Suite fixed to `input[data-qid],
   select[data-qid]`.

## Verified behavior (all through the real DOM + bindings)

- First launch opens the wizard before the main view; finish lands on a
  non-empty main view (2 wizard-created surveys, cross-checked in
  SQLite).
- Author → respondent → response row: a survey created through the
  Author tab is answerable in the respondent runner, and the stored
  response appears in the Responses table with its id and chosen answer.
- Branch logic both directions in the runner: q1=x hides q2 (preview
  lists no q2); q1=y reveals it.
- Invalid author JSON shows the error in the status line and stores
  nothing (sqlite cross-check: 0 surveys).
- Migrate dry-run lists the exact unmapped keys from the fixture (f6
  hidden, f7 matrix), and the CLI on the same fixture reports the same
  counts (4 mapped / 3 unmapped).
- Plugin: sample passes (`ok sample 1.0.0`); a manifest with
  `api_version: 9` shows an error naming `api_version` — a message, not
  a crash or a 500.
- Back enabled from step 2; Back returns to step 1.
- Empty state: fresh dir shows the empty message and hides the table;
  loading indicators never stick.
- No secrets in the window text (AKIA, sk_live, ghp_, private keys,
  xoxb-, secretAccessKey all absent).
- Donate: both addresses byte-for-byte on the donate view, the wizard's
  last step, and the footer.
- Keyboard: `w` reopens the wizard, Escape skips it.
- `go build -tags "wails production webkit2_41"` still produces the
  desktop binary; the default cgo-free build still builds and prints
  desktop instructions.

## Gates re-run fresh at commit time

gofmt clean · vet clean · `go test ./...` 6/6 · tagged desktop build
green · both GUI suites green on fresh data dirs (full: 11 PASS +
wait-gates; audit: 23 PASS + wait-gates).
