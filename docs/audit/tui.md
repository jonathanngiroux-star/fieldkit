# TUI audit (step 6)

Full audit of the terminal UI. Scripted key tests drive the real tview
app on a simulated tcell screen (80x24); model-level tests pin the
text views; the launcher's non-TTY path is timed. Bugs found were
fixed in this pass, not ticketed.

## Bugs found and fixed (this pass + the step-5 build)

1. **Fieldless wizard steps had no Run button** — the TUI form only
   added the step's Run/NextLabel button when the step had fields, so
   the import step was unrunnable from the keyboard (Enter fell
   through to navigation). Every step now gets its Run button.
   Found by `TestTUIWizardScriptedKeys` (import never ran).
2. **Donate addresses fell below the 80x24 fold** — the wizard page
   layout (body:form:output at 2:3:3) pushed the donate step's
   addresses off-screen at 80x24. Re-proportioned to 2:1:2; the
   addresses now render inside the fold. Found by
   `TestTUIWizardPagesRender` (pass 2: donate step on-screen without
   both addresses).

## Test-harness findings (not app bugs — recorded so they are not
re-chased)

- tview's first paint against a `SetScreen`-after-construction sim
  screen misassumes size 0; a resize pulse (80x25 → 80x24) forces a
  correct re-layout. Real terminals always deliver a size before the
  first paint, so production is unaffected.
- Injected runes append to input defaults; the driver backspaces the
  default away before typing (the author step's "My first survey"
  default otherwise corrupts the typed title).
- `QueueUpdate` does not redraw; page switches from outside the event
  loop need `QueueUpdateDraw`.

## Verified behavior

- **Keyboard coverage**: 1–4 switch sections (surveys / logic /
  migrate / donate) with content asserted on screen; `w` opens the
  wizard from the main screen; donate page shows both addresses
  byte-for-byte; the global donate footer is on screen in every
  section. (`TestTUIMainScreenKeyboard`)
- **First launch** on an empty data dir lands on wizard step 1 before
  the main screen is usable. (`TestTUIFirstLaunchOpensWizard`)
- **Scripted wizard walk**: Next (data) → Import (creates the fixture
  survey) → typed title + Create (author survey) → Submit (stores a
  response) — all through real key events on focused buttons, all
  cross-verified in SQLite. (`TestTUIWizardScriptedKeys`)
- **Every wizard page renders its title on 80x24**; the donate step
  shows both addresses on-screen. (`TestTUIWizardPagesRender`)
- **Back/Next via keys**; **Skip marks seen** (any exit marks seen;
  `w` reopens). (`TestTUIWizardBackNext`, `TestTUIWizardSkipMarksSeen`)
- **Resize mid-wizard** (80x24 → 60x18 → 80x24 with a keypress in the
  resize window): no crash. (`TestTUIResizeMidWizard`)
- **No blocking network on launch**: the non-TTY path refuses
  instantly (<2s, measured) with an interactive-terminal hint; the
  interactive path constructs only store + embed access. The TUI does
  no network I/O anywhere (grep-verified: no net/http in the TUI
  files; clipboard uses OSC-52 on /dev/tty then local tools).
- **Same fixture results as the CLI**: `TestTUIMigrateSummary` runs
  the same converter and asserts the same 4-mapped/3-unmapped counts
  the CI jq gate enforces on the CLI; `TestTUILogicDryRun` runs the
  same engine the `fieldkit logic` command runs.
- **Donate**: both addresses exact on the donate page, in the wizard's
  last step, and in the footer of every screen.

## Gates re-run fresh at commit time

gofmt clean · vet clean · `go test ./...` 6/6 packages · TUI tests
13/13 · plain build green · wails-tagged desktop build green.
