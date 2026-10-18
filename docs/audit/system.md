# System audit (step 7) — v0.1 release gate

Whole-system audit: backend + Wails GUI + TUI, cross-surface agreement,
billing sweep, cold start, and the release checks. Bugs found were
fixed in this pass, not ticketed.

## Cross-surface agreement (the step-7 contract)

`gui_suite_system.py` (bridge-driven, fresh datadir) proves:

- **Migrate JSON matches across CLI and GUI**: the same committed
  Typeform fixture run through the GUI's migrate viewer and the CLI's
  `fieldkit migrate` produces identical reports — same source, 4
  mapped questions, 3 unmapped, 1 ending. The TUI runs the same
  converter (its summary counts are pinned by `TestTUIMigrateSummary`
  against the same fixture the CI jq gate uses).
- **Same survey submits from every surface into one store**: a response
  submitted through the GUI's respondent runner appears in the CLI's
  `fieldkit export csv`; a response POSTed to the HTTP endpoint appears
  in the same export. One SQLite file, three front doors.
- **Addresses identical**: CLI `fieldkit donate`, the GUI donate view,
  the TUI donate page, README, GOVERNANCE.md, and docs/donate.md all
  carry both addresses byte-for-byte.

## Bugs found and fixed in this pass

1. **Wails binding namespace broke after the restructure** — moving
   the bound `DesktopApp` from the CLI's `main` package into
   `internal/desktop` changed the generated binding namespace from
   `window.go.main` to `window.go.desktop`; every frontend call
   silently no-oped (no wizard auto-open, no version line, no binding
   response). The frontend now resolves the bindings dynamically
   (`fkBindings()` probes `window.go.*` for `DesktopApp`) and the boot
   path waits for injection. Found by driving the window: the DOM
   rendered but no binding call produced an effect; fixed and re-proven
   by all three suites (11 + 23 + 13 checks).
2. **`wails build` could not produce a binary from this repo** — the
   Wails CLI requires the main package at the project root, which lived
   in `cmd/fieldkit`. Added a wails-tagged root `main.go` shimming
   `internal/desktop.Run`, split the desktop package into wails/default
   halves so the cgo-free build stays clean, and added `wails.json`.
   `wails build -tags "wails webkit2_41"` now produces
   `build/bin/fieldkit-desktop` (verified locally; gated in CI).
3. **Test bridge raced the Wails runtime injection** — the JS
   interpreter early-returned when `window.go` was not yet injected and
   registered its `test:cmd` listener only if the runtime existed at
   `DOMContentLoaded`. Both now poll for readiness (bounded), and the
   bridge announces `bridge-ready` — the driver's ping-wait covers it.

## Verified in this pass (fresh runs)

- Backend suite 6/6 packages green, race clean on store/serve/logic.
- TUI suite 13/13 green (sim-screen keyboard walks, resize, first-run).
- GUI suites 3/3 green: full 11 PASS, audit 23 PASS, system 13 PASS.
- `wails build` produces `build/bin/fieldkit-desktop`; the default
  cgo-free build still compiles and its `fieldkit desktop` prints
  build instructions.
- **Billing vocabulary sweep**: the only occurrences of
  stripe/subscription/contact-sales-class words are negation voice
  ("no subscription", "no paid tier") — the `TestNoBillingSurface` walk
  gate passes on the full tree (all shipped files).
- **GOVERNANCE.md** lists both addresses and the 4/4 + 30-day rule.
- **Plugin contract still green** after GUI/TUI landed: sample plugin
  validates via CLI and via the GUI plugin tab.
- **Cold start**: fresh tree → `go build` + create survey + list +
  migrate dry-run in ~2s (warm cache) from a temp archive; prebuilt
  binary cold start (create + list) measured at **10 ms**. Cap is
  15 minutes; the margin is not close.
- **Docker**: `docker build` gated in CI (golang:1.27 build stage →
  scratch runtime, VOLUME /data, ENTRYPOINT fieldkit). Not runnable on
  this host (no docker daemon) — the gate runs on CI instead. This is
  the one system check verified by CI config rather than a local run,
  and it is recorded here as such.

## Release gate verdict

v0.1 draft notes: `docs/release-v0.1.md`. All step-7 definition-of-done
items verified except `docker run`, which is deferred to CI on push
(no local daemon).
