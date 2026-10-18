package main

// tui_wizard.go: the TUI wizard renderer. One tview page per step, built
// from the shared fieldapp.WizardSteps() — zero logic here; Run collects the
// field values and calls fieldapp.WizardRunStep, exactly like the GUI. Every
// step is keyboard-reachable: Tab/Enter traverse the form, Enter in an
// input runs the step.

import (
	fieldapp "github.com/jonathanngiroux-star/fieldkit/internal/app"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// tuiWizard is the wizard overlay: one page per step over the main UI.
type tuiWizard struct {
	app     *tview.Application
	m       *fieldapp.Model
	pages   *tview.Pages
	steps   []fieldapp.WizardStep
	forms   []*tview.Form
	outputs []*tview.TextView
	onDone  func()
}

// newTUIWizard builds all wizard pages up front. onDone fires when the
// tour is finished or skipped.
func newTUIWizard(app *tview.Application, m *fieldapp.Model, onDone func()) *tuiWizard {
	w := &tuiWizard{
		app:    app,
		m:      m,
		pages:  tview.NewPages(),
		steps:  fieldapp.WizardSteps(),
		onDone: onDone,
	}
	for i := range w.steps {
		page, form, output := w.buildPage(i)
		w.forms = append(w.forms, form)
		w.outputs = append(w.outputs, output)
		w.pages.AddPage(w.steps[i].ID, page, true, i == 0)
	}
	return w
}

// buildPage renders one step: body, fields form (with Run + navigation
// buttons — tview forms give native Tab/Enter traversal), and output.
func (w *tuiWizard) buildPage(i int) (tview.Primitive, *tview.Form, *tview.TextView) {
	step := w.steps[i]

	body := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	body.SetText(step.Body)
	body.SetBorder(true).SetTitle(" " + step.Title + " ")

	output := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	output.SetBorder(true).SetTitle(" result ")

	form := tview.NewForm().
		SetCancelFunc(func() { w.finish(false) })
	for _, f := range step.Fields {
		form.AddInputField(f.Label, f.Value, 40, nil, nil)
		// Enter in the field runs the step directly.
		idx := form.GetFormItemCount() - 1
		form.GetFormItem(idx).(*tview.InputField).SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				w.run(i)
			}
		})
	}
	// Every step gets its Run button (label = NextLabel) — including
	// fieldless steps like import: the button both runs the operation
	// and shows its result; Next (below) advances without running.
	runLabel := step.NextLabel
	if runLabel == "" {
		runLabel = "Run"
	}
	form.AddButton(runLabel, func() {
		w.run(i)
		if i+1 < len(w.steps) {
			w.pages.SwitchToPage(w.steps[i+1].ID)
			w.app.SetFocus(w.forms[i+1])
		} else {
			w.finish(true)
		}
	})
	if i > 0 {
		form.AddButton("Prev", func() {
			w.pages.SwitchToPage(w.steps[i-1].ID)
			w.app.SetFocus(w.forms[i-1])
		})
	}
	nextLabel := "Next"
	if i == len(w.steps)-1 {
		nextLabel = "Finish"
	}
	form.AddButton(nextLabel, func() {
		if i+1 < len(w.steps) {
			w.pages.SwitchToPage(w.steps[i+1].ID)
			w.app.SetFocus(w.forms[i+1])
		} else {
			w.finish(true)
		}
	})
	form.AddButton("Skip tour", func() { w.finish(false) })
	form.SetBorder(true).SetTitle(" fields ")

	page := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(body, 0, 2, false).
		AddItem(form, 0, 1, true).
		AddItem(output, 0, 2, false)
	return page, form, output
}

// run collects the form's values and executes the step's real
// operation, rendering the result (or error) in the output area.
func (w *tuiWizard) run(i int) {
	step := w.steps[i]
	form := w.forms[i]
	values := map[string]string{}
	for fi, f := range step.Fields {
		values[f.ID] = form.GetFormItem(fi).(*tview.InputField).GetText()
	}
	out, err := fieldapp.WizardRunStep(w.m, step.ID, values)
	ot := w.outputs[i]
	if err != nil {
		ot.SetText("[red]" + err.Error())
	} else {
		ot.SetText("[white]" + out)
	}
}

// finish closes the wizard. ANY exit — finish, skip, or Esc — marks it
// seen: a wizard that cannot be declined is mandatory, and this one
// must not be. wizard_seen only gates the auto-open; the w hotkey and
// the nav entry always reopen it.
func (w *tuiWizard) finish(completed bool) {
	_ = w.m.WizardDone()
	w.onDone()
}
