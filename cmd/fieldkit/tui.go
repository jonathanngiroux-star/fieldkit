package main

// tui.go: the terminal UI. tview/tcell is pure Go (no cgo), so the TUI
// ships in the default cgo-free binary. All logic lives on appModel;
// this file only renders. Keyboard-first: q quits, w opens the wizard,
// Tab/Enter traverse forms, 1-4 switch sections, e/b copy the donate
// addresses on the donate page.

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// runTUI builds and runs the terminal UI. Requires a TTY (checked by
// the launcher, not here).
func runTUI(m *appModel) int {
	app := tview.NewApplication()

	output := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	output.SetBorder(true).SetTitle(" output ")

	// ---------- surveys page ----------
	surveyList := tview.NewList().ShowSecondaryText(false)
	refillSurveys := func() {
		surveyList.Clear()
		lines, err := m.tuiSurveyList()
		if err != nil {
			surveyList.AddItem("[red]"+err.Error(), "", 0, nil)
			return
		}
		for _, ln := range lines {
			surveyList.AddItem(ln, "", 0, nil)
		}
	}
	refillSurveys()

	// ---------- logic page ----------
	logicForm := tview.NewForm()
	logicForm.AddInputField("survey id", "1", 6, nil, nil)
	logicForm.AddInputField("answers JSON", `{"q1":"no","q2":"x","q3":"y"}`, 48, nil, nil)
	logicForm.AddButton("Dry-run", func() {
		id := parseID(logicForm.GetFormItem(0).(*tview.InputField).GetText())
		out, err := m.tuiLogicDryRun(id, logicForm.GetFormItem(1).(*tview.InputField).GetText())
		if err != nil {
			output.SetText("[red]" + out)
			return
		}
		output.SetText("[white]" + out)
	})
	logicForm.SetBorder(true).SetTitle(" logic dry-run ")

	// ---------- migrate page ----------
	migrateForm := tview.NewForm()
	migrateForm.AddInputField("typeform export path", "", 48, nil, nil)
	migrateForm.AddButton("Use bundled sample", func() {
		path, err := sampleFixturePath()
		if err != nil {
			output.SetText("[red]" + err.Error())
			return
		}
		migrateForm.GetFormItem(0).(*tview.InputField).SetText(path)
	})
	migrateForm.AddButton("Dry-run", func() {
		out, err := m.tuiMigrateSummary(migrateForm.GetFormItem(0).(*tview.InputField).GetText())
		if err != nil {
			output.SetText("[red]" + err.Error())
			return
		}
		output.SetText("[white]" + out)
	})
	migrateForm.SetBorder(true).SetTitle(" migrate dry-run ")

	// ---------- donate page ----------
	donateText := tview.NewTextView().SetDynamicColors(true)
	donateText.SetText(m.tuiDonate())
	donateStatus := tview.NewTextView().SetDynamicColors(true)
	setDonateHint := func() {
		donateStatus.SetText("[gray]e = copy ETH/USDC · b = copy Bitcoin · q = quit[-:]")
	}
	setDonateHint()
	copyDonate := func(addr string) {
		if err := copyAddress(addr); err != nil {
			donateStatus.SetText("[red]" + err.Error())
			return
		}
		donateStatus.SetText("[green]copied:[white] " + addr)
	}
	donatePage := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(donateText, 0, 1, false).
		AddItem(donateStatus, 1, 0, false)

	// ---------- pages + nav ----------
	pages := tview.NewPages()
	pages.AddPage("surveys", surveyList, true, true)
	pages.AddPage("logic", tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(logicForm, 7, 0, true).
		AddItem(output, 0, 1, false), true, false)
	pages.AddPage("migrate", tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(migrateForm, 8, 0, true).
		AddItem(output, 0, 1, false), true, false)
	pages.AddPage("donate", donatePage, true, false)

	activePage := "surveys"

	// donate footer — visible on every page, per the donation rule
	donateFooter := tview.NewTextView().SetDynamicColors(true)
	donateFooter.SetText("[gray]free forever · donate: ETH/USDC " + donateEthereum + " · BTC " + donateBitcoin + "[-:]")

	nav := tview.NewList().ShowSecondaryText(false)
	sections := []struct {
		name   string
		page   string
		refill func()
	}{
		{"surveys", "surveys", refillSurveys},
		{"logic", "logic", func() {}},
		{"migrate", "migrate", func() {}},
		{"donate", "donate", func() {}},
	}
	for _, sec := range sections {
		sec := sec
		nav.AddItem(sec.name, "", 0, func() {
			pages.SwitchToPage(sec.page)
			activePage = sec.page
			sec.refill()
		})
	}

	root := tview.NewFlex().
		AddItem(nav, 16, 0, true).
		AddItem(pages, 0, 1, false)
	rootRow := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(root, 0, 1, true).
		AddItem(donateFooter, 1, 0, false)

	// The wizard overlays the whole UI; finishing restores the main root.
	openWizard := func() {
		wiz := newTUIWizard(app, m, func() {
			app.SetRoot(rootRow, true)
			refillSurveys()
		})
		app.SetRoot(wiz.pages, true)
	}
	nav.AddItem("wizard (w)", "", 0, openWizard)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'q', 'Q':
				app.Stop()
				return nil
			case 'w', 'W':
				openWizard()
				return nil
			case 'e', 'E':
				if activePage == "donate" {
					copyDonate(donateEthereum)
					return nil
				}
			case 'b', 'B':
				if activePage == "donate" {
					copyDonate(donateBitcoin)
					return nil
				}
			case '1', '2', '3', '4':
				idx := int(event.Rune() - '1')
				if idx < len(sections) {
					pages.SwitchToPage(sections[idx].page)
					activePage = sections[idx].page
					sections[idx].refill()
					return nil
				}
			}
		}
		return event
	})

	// first run: the wizard opens before the main screen
	if !m.wizardSeen() {
		go func() {
			// let the app paint first, then overlay
			app.QueueUpdateDraw(func() { openWizard() })
		}()
	}

	if err := app.Run(); err != nil {
		return 1
	}
	return 0
}

// parseID parses a numeric survey id; 0 on junk (the dry-run will say
// survey not found).
func parseID(s string) int64 {
	var id int64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		id = id*10 + int64(r-'0')
	}
	return id
}
