package main

// tui_screen_test.go: scripted keyboard test of the TUI wizard through
// a simulated tcell screen — real tview app, real key events, no
// terminal. Buttons are focused directly (tview forms have many focus
// stops; guessing Tab counts is flaky), then pressed with Enter — the
// handler chain (button -> run -> fieldapp.WizardRunStep -> store) is fully real.

import (
	fieldapp "github.com/jonathanngiroux-star/fieldkit/internal/app"

	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/jonathanngiroux-star/fieldkit/internal/store"
)

// runWizardOnScreen mounts the wizard on a sim screen with the app's
// draw loop running, drives the script, returns the final screen text.
//
// Ordering is load-bearing: tview requires SetScreen BEFORE Run().
// Calling SetScreen after Run() leaves the event loop attached to no
// usable screen — QueueUpdateDraw then blocks forever (observed as a
// 10m CI hang, passing locally by timing luck). A bounded readiness
// wait (non-blank content) makes any future regression fail fast
// instead of hanging the package.
func runWizardOnScreen(t *testing.T, m *fieldapp.Model, script func(app *tview.Application, w *tuiWizard, screen tcell.SimulationScreen)) string {
	t.Helper()
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("sim screen: %v", err)
	}
	t.Cleanup(func() { screen.Fini() })
	screen.SetSize(80, 24)

	app := tview.NewApplication()
	wiz := newTUIWizard(app, m, func() {})
	app.SetRoot(wiz.pages, true)

	// SetScreen BEFORE Run — the API contract.
	app.SetScreen(screen)
	go func() { _ = app.Run() }()
	t.Cleanup(func() {
		app.QueueUpdate(func() { app.Stop() })
		time.Sleep(100 * time.Millisecond)
	})

	// readiness: wait (bounded) until the loop has drawn the wizard's
	// first step — content appears on the sim screen.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		screen.Sync()
		if c, _, _, _ := screen.GetContent(2, 1); c != ' ' && c != 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if time.Now().After(deadline) {
		t.Fatal("tview draw loop never became ready on the sim screen")
	}

	script(app, wiz, screen)

	screen.Sync()
	var b strings.Builder
	for y := 0; y < 24; y++ {
		for x := 0; x < 80; x++ {
			c, _, _, _ := screen.GetContent(x, y)
			b.WriteRune(c)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// pressButton focuses a form's button by label and presses Enter —
// real key event on the real focused button.
func pressButton(app *tview.Application, w *tuiWizard, screen tcell.SimulationScreen, step int, label string) {
	app.QueueUpdate(func() {
		for i := 0; i < w.forms[step].GetButtonCount(); i++ {
			btn := w.forms[step].GetButton(i)
			if strings.EqualFold(btn.GetLabel(), label) {
				app.SetFocus(btn)
				return
			}
		}
	})
	time.Sleep(80 * time.Millisecond)
	screen.InjectKey(tcell.KeyEnter, 0, tcell.ModNone)
	time.Sleep(120 * time.Millisecond)
}

// typeInto focuses a step's input field, selects-all/clears it, and
// types text (injected runes append, so the default must be removed
// first).
func typeInto(app *tview.Application, w *tuiWizard, screen tcell.SimulationScreen, step, field int, text string) {
	app.QueueUpdate(func() {
		f := w.forms[step].GetFormItem(field).(*tview.InputField)
		app.SetFocus(f)
	})
	time.Sleep(80 * time.Millisecond)
	// Backspace out any default text, then type. (Ctrl-A selection in
	// the sim screen leaves a partial tail; backspaces are mechanical
	// and visible in the assertion output when wrong.)
	for i := 0; i < 40; i++ {
		screen.InjectKey(tcell.KeyBackspace, 0, tcell.ModNone)
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(30 * time.Millisecond)
	for _, r := range text {
		screen.InjectKey(tcell.KeyRune, r, tcell.ModNone)
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(80 * time.Millisecond)
}

// The full wizard walk via keys: Next (data), Import, Create with a
// typed title, Submit — every step's real operation verified in the
// store afterwards.
func TestTUIWizardScriptedKeys(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	m := fieldapp.New(st)

	runWizardOnScreen(t, m, func(app *tview.Application, w *tuiWizard, screen tcell.SimulationScreen) {
		// step 0 data: Next
		pressButton(app, w, screen, 0, "Next")
		// step 1 import: Import (real op)
		pressButton(app, w, screen, 1, "Import")
		// step 2 author: type a title, then Create
		typeInto(app, w, screen, 2, 0, "TUI wizard survey")
		pressButton(app, w, screen, 2, "Create")
		// step 3 respond: Submit (fields prefilled: q1=yes)
		pressButton(app, w, screen, 3, "Submit")
		time.Sleep(400 * time.Millisecond)
	})

	list, err := st.ListSurveys()
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, s := range list {
		names[s.Name] = true
	}
	if !names["Onboarding feedback"] {
		t.Fatalf("import step did not run via keys; surveys=%v", names)
	}
	if !names["TUI wizard survey"] {
		t.Fatalf("author step did not run via keys; surveys=%v", names)
	}
	var total int
	for _, s := range list {
		rows, err := st.Responses(s.ID)
		if err != nil {
			t.Fatal(err)
		}
		total += len(rows)
	}
	if total < 1 {
		t.Fatal("respond step did not run: no response stored")
	}
}

// Every wizard page renders on an 80x24 screen without crashing, and
// the donate step's addresses are on-screen when it is the visible
// page.
func TestTUIWizardPagesRender(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	m := fieldapp.New(st)

	// Pass 1: step through every page; each must leave its title on
	// screen (a page that fails to draw shows blank/previous content).
	for i := range fieldapp.WizardSteps() {
		idx := i
		text := runWizardOnScreen(t, m, func(app *tview.Application, w *tuiWizard, screen tcell.SimulationScreen) {
			app.QueueUpdateDraw(func() {
				w.pages.SwitchToPage(w.steps[idx].ID)
			})
			time.Sleep(300 * time.Millisecond)
		})
		want := wsteps()[idx].Title
		if !strings.Contains(text, want) {
			t.Fatalf("step %d (%q) did not render its title on 80x24:\n%s", idx, want, text)
		}
	}

	// Pass 2: the donate step shows both addresses byte-for-byte.
	text := runWizardOnScreen(t, m, func(app *tview.Application, w *tuiWizard, screen tcell.SimulationScreen) {
		last := len(w.steps) - 1
		app.QueueUpdateDraw(func() {
			w.pages.SwitchToPage(w.steps[last].ID)
		})
		time.Sleep(300 * time.Millisecond)
	})
	if !strings.Contains(text, fieldapp.DonateEthereum) || !strings.Contains(text, fieldapp.DonateBitcoin) {
		t.Fatalf("donate step missing addresses:\n%s", text)
	}
}

// wsteps is fieldapp.WizardSteps via the shared constructor (kept separate so
// the test reads clearly).
func wsteps() []fieldapp.WizardStep { return fieldapp.WizardSteps() }

// Back/Next navigation via keys: Next then Prev returns to step 1's
// title on screen.
func TestTUIWizardBackNext(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	m := fieldapp.New(st)

	text := runWizardOnScreen(t, m, func(app *tview.Application, w *tuiWizard, screen tcell.SimulationScreen) {
		pressButton(app, w, screen, 0, "Next")
		pressButton(app, w, screen, 1, "Prev")
		time.Sleep(200 * time.Millisecond)
	})
	if !strings.Contains(text, "Where should Fieldkit store data?") {
		t.Fatalf("Prev did not return to step 1:\n%.400s", text)
	}
}

// Skip tour: any exit marks seen; the store records it.
func TestTUIWizardSkipMarksSeen(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	m := fieldapp.New(st)

	runWizardOnScreen(t, m, func(app *tview.Application, w *tuiWizard, screen tcell.SimulationScreen) {
		pressButton(app, w, screen, 0, "Skip tour")
		time.Sleep(200 * time.Millisecond)
	})
	if !m.WizardSeen() {
		t.Fatal("skip must mark the wizard seen")
	}
}
