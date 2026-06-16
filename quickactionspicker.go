//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// quickActionsTitle is the heading shown across the top of the action picker.
const quickActionsTitle = "⚡ Quick Actions"

// pickerHeaderLines is how many screen lines precede the embedded fuzzyList in
// every picker stage: the title bar and the blank line under it. A mouse click's
// screen row minus this offset is the list-local line passed to clickRow.
const pickerHeaderLines = 2

// stage is which screen the picker is currently showing.
type stage int

const (
	// stageActions is the top-level list of actions.
	stageActions stage = iota
	// stageSelect is the option list shown for a "select" action.
	stageSelect
	// stageForm is the text field shown for a "form" action.
	stageForm
)

// pickerModel holds the state of the fuzzy-finder TUI. It is a small state
// machine: pick an action, then for select/form actions gather the extra input
// before the program quits and the chosen action runs.
type pickerModel struct {
	ctx RunContext

	stage stage

	actions    []Action
	actionList fuzzyList

	// current is the action awaiting extra input (set in the select/form stages).
	current    *Action
	optionList fuzzyList
	formInput  textinput.Model

	width  int
	height int

	// Results, read back after the program exits.
	chosen   *Action // the action to run, nil if the user cancelled
	value    string  // resolved option value or form input for the chosen action
	quitting bool
}

// newPickerModel builds the initial TUI state showing every action. Actions are
// partitioned by origin so project-local actions sort and render ahead of the
// global ones; m.actions is stored in that same project-then-global order so each
// list row's ref indexes straight back into it.
func newPickerModel(ctx RunContext, actions []Action) pickerModel {
	var project, global []Action
	for _, a := range actions {
		if a.origin == originProject {
			project = append(project, a)
		} else {
			global = append(global, a)
		}
	}

	ordered := make([]Action, 0, len(actions))
	ordered = append(ordered, project...)
	ordered = append(ordered, global...)

	return pickerModel{
		ctx:        ctx,
		stage:      stageActions,
		actions:    ordered,
		actionList: newFuzzyList("Type to filter…", actionListItems(project, global)),
	}
}

// actionListItems turns the project and global actions into picker rows. When
// both groups are present it brackets each with a non-selectable "Project" /
// "Global" heading so the origin of every action is clear; when only one group
// exists it emits a plain, ungrouped list so a repo without project actions looks
// exactly as it did before. Each selectable row's ref is its index into the
// concatenated project++global slice, matching the order newPickerModel stores in
// m.actions.
func actionListItems(project, global []Action) []listItem {
	grouped := len(project) > 0 && len(global) > 0
	items := make([]listItem, 0, len(project)+len(global)+2)

	if grouped {
		items = append(items, listItem{name: "Project"})
	}
	for i, a := range project {
		items = append(items, listItem{name: a.Name, desc: a.Description, selectable: true, ref: i})
	}

	if grouped {
		items = append(items, listItem{name: "Global"})
	}
	for j, a := range global {
		items = append(items, listItem{name: a.Name, desc: a.Description, selectable: true, ref: len(project) + j})
	}

	return items
}

// Init implements tea.Model and starts the cursor blinking.
func (m pickerModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update routes key presses to the handler for the current stage and forwards
// everything else (window sizes, the blink tick) to the active input.
func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.MouseMsg:
		return m.updateMouse(msg)

	case tea.KeyMsg:
		switch m.stage {
		case stageActions:
			return m.updateActions(msg)
		case stageSelect:
			return m.updateSelect(msg)
		case stageForm:
			return m.updateForm(msg)
		}
	}

	return m.forwardToInput(msg)
}

// updateMouse turns mouse input into list navigation: the wheel moves the
// highlight, and the left button selects the row under the pointer — running it
// on release, the natural completion of a click. The text-only form stage has no
// list, so it ignores the mouse. This works only because herdr forwards mouse
// events to the focused pane's program once that program enables mouse reporting
// (which runQuickActionsUI does via tea.WithMouseCellMotion); otherwise herdr
// would keep the clicks for its own pane focus/selection.
func (m pickerModel) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch m.stage {
	case stageActions:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.actionList.moveUp()
		case tea.MouseButtonWheelDown:
			m.actionList.moveDown()
		case tea.MouseButtonLeft:
			if m.actionList.clickRow(msg.Y-pickerHeaderLines) && msg.Action == tea.MouseActionRelease {
				return m.activateAction()
			}
		}
	case stageSelect:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.optionList.moveUp()
		case tea.MouseButtonWheelDown:
			m.optionList.moveDown()
		case tea.MouseButtonLeft:
			if m.optionList.clickRow(msg.Y-pickerHeaderLines) && msg.Action == tea.MouseActionRelease {
				return m.activateOption()
			}
		}
	}
	return m, nil
}

// updateActions handles keys while choosing an action. Selecting a plain command
// quits so it can run; selecting a select/form action advances to the matching
// input stage.
func (m pickerModel) updateActions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.quitting = true
		return m, tea.Quit
	case "up", "ctrl+p":
		m.actionList.moveUp()
		return m, nil
	case "down", "ctrl+n":
		m.actionList.moveDown()
		return m, nil
	case "enter":
		return m.activateAction()
	}

	cmd := m.actionList.editQuery(msg)
	return m, cmd
}

// activateAction runs the highlighted action: a plain command quits so it can
// run, while a select/form action advances to its input stage. It is shared by
// the enter key and a left-click; activating with nothing selectable is a no-op.
func (m pickerModel) activateAction() (tea.Model, tea.Cmd) {
	idx := m.actionList.selectedIndex()
	if idx < 0 {
		return m, nil
	}
	a := m.actions[idx]
	switch a.effectiveType() {
	case TypeSelect:
		m.current = &a
		m.optionList = newFuzzyList("Pick an option…", optionItems(a.Options))
		m.stage = stageSelect
		return m, textinput.Blink
	case TypeForm:
		m.current = &a
		m.formInput = newFormInput(a.Form)
		m.stage = stageForm
		return m, textinput.Blink
	default: // TypeCommand
		m.chosen = &a
		return m, tea.Quit
	}
}

// updateSelect handles keys while choosing an option for a select action. esc
// returns to the action list; enter records the chosen value and quits.
func (m pickerModel) updateSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		m.current = nil
		m.stage = stageActions
		return m, textinput.Blink
	case "up", "ctrl+p":
		m.optionList.moveUp()
		return m, nil
	case "down", "ctrl+n":
		m.optionList.moveDown()
		return m, nil
	case "enter":
		return m.activateOption()
	}

	cmd := m.optionList.editQuery(msg)
	return m, cmd
}

// activateOption records the highlighted option's value as the chosen action's
// value and quits so the action runs. Shared by the enter key and a left-click;
// activating with nothing selectable is a no-op.
func (m pickerModel) activateOption() (tea.Model, tea.Cmd) {
	idx := m.optionList.selectedIndex()
	if idx < 0 {
		return m, nil
	}
	m.value = m.current.Options[idx].resolvedValue()
	m.chosen = m.current
	return m, tea.Quit
}

// updateForm handles keys while entering text for a form action. esc returns to
// the action list; enter records the entered string and quits.
func (m pickerModel) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		m.current = nil
		m.stage = stageActions
		return m, textinput.Blink
	case "enter":
		m.value = strings.TrimSpace(m.formInput.Value())
		m.chosen = m.current
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.formInput, cmd = m.formInput.Update(msg)
	return m, cmd
}

// forwardToInput passes a non-key message to whichever input is active so the
// cursor keeps blinking and text keeps flowing in every stage.
func (m pickerModel) forwardToInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.stage {
	case stageActions:
		cmd = m.actionList.editQuery(msg)
	case stageSelect:
		cmd = m.optionList.editQuery(msg)
	case stageForm:
		m.formInput, cmd = m.formInput.Update(msg)
	}
	return m, cmd
}

// View renders the screen for the current stage.
func (m pickerModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	switch m.stage {
	case stageSelect:
		b.WriteString(titleStyle.Render(m.current.Name))
		b.WriteString("\n\n")
		b.WriteString(m.optionList.view("no matching options"))
		b.WriteString("\n")
		b.WriteString(footerStyle.Render("↑/↓ move • click/enter select • esc back"))

	case stageForm:
		b.WriteString(titleStyle.Render(m.current.Name))
		b.WriteString("\n\n")
		prompt := m.current.Form.Prompt
		if prompt == "" {
			prompt = "Enter a value"
		}
		b.WriteString(descStyle.Render(prompt))
		b.WriteString("\n")
		b.WriteString(promptStyle.Render("❯ "))
		b.WriteString(m.formInput.View())
		b.WriteString("\n\n")
		b.WriteString(footerStyle.Render("enter submit • esc back"))

	default: // stageActions
		b.WriteString(titleStyle.Render(quickActionsTitle))
		b.WriteString("\n\n")
		b.WriteString(m.actionList.view("no matching actions"))
		b.WriteString("\n")
		b.WriteString(footerStyle.Render("↑/↓ move • click/enter run • esc cancel"))
	}

	return b.String()
}

// optionItems turns a select action's options into list rows. A selectable row
// shows its label plus the option's optional description (the value is never
// shown, so encoding data into the value — e.g. "host url" — does not clutter
// the list). A separator option becomes a non-selectable heading/spacer row.
// ref carries each row's index into the original options slice so the picker can
// map the selected row back to its option.
func optionItems(options []Option) []listItem {
	items := make([]listItem, 0, len(options))
	for i, o := range options {
		if o.isSeparator() {
			items = append(items, listItem{name: o.Heading})
			continue
		}
		items = append(items, listItem{name: o.Label, desc: o.Description, selectable: true, ref: i})
	}
	return items
}

// newFormInput builds the text field for a form action, applying its optional
// placeholder.
func newFormInput(form FormConfig) textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = form.Placeholder
	if ti.Placeholder == "" {
		ti.Placeholder = "Type a value…"
	}
	ti.Focus()
	return ti
}

// runQuickActionsUI renders the fuzzy-finder TUI inside the zoomed pane herdr
// opens for the `quick-actions-picker` entrypoint. It recovers the launch context
// from the HERDR_PLUS_CTX environment variable the launching action set, loads the
// actions for that working directory, and runs the chosen action with that
// context. On exit herdr tears the pane down, returning focus to where you were.
func runQuickActionsUI() {
	ctx, err := decodeRunContext(os.Getenv("HERDR_PLUS_CTX"))
	if err != nil {
		errExit("could not decode run context:", err)
	}

	actions, err := loadPickerActions(ctx.WorkDir)
	if err != nil {
		// Leave the pane open so the user can read the config error.
		errExit(err)
	}

	if len(actions) == 0 {
		dir, _ := quickActionsConfigDir()
		errExit(fmt.Sprintf("no actions found in %s", dir))
	}

	// WithMouseCellMotion enables click/release/wheel events so a row can be run
	// with the mouse.
	p := tea.NewProgram(newPickerModel(ctx, actions), tea.WithAltScreen(), tea.WithMouseCellMotion())
	result, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "herdr-plus:", err)
	}

	// Run the chosen action before exiting (which tears the pane down). The context
	// carries the working directory and session metadata; we fill in the resolved
	// Value.
	if m, ok := result.(pickerModel); ok && m.chosen != nil {
		runCtx := ctx
		runCtx.Value = m.value
		if err := m.chosen.run(runCtx); err != nil {
			fmt.Fprintln(os.Stderr, "herdr-plus: action failed:", err)
		}
	}
}
