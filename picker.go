//
// Date: 2026-06-09
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
	"github.com/charmbracelet/lipgloss"
)

// Palette. A small, cohesive set of colors for a clean dark-terminal look.
// These styles are shared with the fuzzyList component in the same package.
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#11111B")).
			Background(lipgloss.Color("#A78BFA")).
			Padding(0, 1)

	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
	countStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	nameStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB"))
	nameSelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	descStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	matchStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F2A900")).Bold(true)
	barStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
	footerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5563"))
)

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
	mode Mode
	ctx  RunContext

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

// newPickerModel builds the initial TUI state showing every action.
func newPickerModel(mode Mode, ctx RunContext, actions []Action) pickerModel {
	items := make([]listItem, len(actions))
	for i, a := range actions {
		items[i] = listItem{name: a.Name, desc: a.Description}
	}
	return pickerModel{
		mode:       mode,
		ctx:        ctx,
		stage:      stageActions,
		actions:    actions,
		actionList: newFuzzyList("Type to filter…", items),
	}
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

	cmd := m.actionList.editQuery(msg)
	return m, cmd
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
		idx := m.optionList.selectedIndex()
		if idx < 0 {
			return m, nil
		}
		m.value = m.current.Options[idx].resolvedValue()
		m.chosen = m.current
		return m, tea.Quit
	}

	cmd := m.optionList.editQuery(msg)
	return m, cmd
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
		b.WriteString(footerStyle.Render("↑/↓ move • enter select • esc back"))

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
		b.WriteString(titleStyle.Render(m.mode.Title))
		b.WriteString("\n\n")
		b.WriteString(m.actionList.view("no matching actions"))
		b.WriteString("\n")
		b.WriteString(footerStyle.Render("↑/↓ move • enter run • esc cancel"))
	}

	return b.String()
}

// optionItems turns a select action's options into list rows. The value is shown
// as a dim description only when it differs from the label, so a plain list of
// values stays uncluttered.
func optionItems(options []Option) []listItem {
	items := make([]listItem, len(options))
	for i, o := range options {
		desc := ""
		if o.Value != "" && o.Value != o.Label {
			desc = o.Value
		}
		items[i] = listItem{name: o.Label, desc: desc}
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

// runPicker loads the mode's actions, renders the fuzzy-finder TUI, runs the
// chosen action with the gathered context, and then closes its own herdr pane so
// focus returns to the pane it was launched from. selfPane is the pane id to
// close on exit.
func runPicker(mode Mode, ctx RunContext, selfPane string) {
	actions, err := loadActions(mode)
	if err != nil {
		// Leave the pane open so the user can read the config error.
		errExit(err)
	}

	if len(actions) == 0 {
		dir, _ := modeConfigDir(mode)
		fmt.Fprintf(os.Stderr, "herdr-plus: no actions found in %s\n", dir)
		closeSelf(selfPane)
		return
	}

	p := tea.NewProgram(newPickerModel(mode, ctx, actions), tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "herdr-plus:", err)
	}

	// Run the chosen action before tearing down the pane. The context carries the
	// working directory and session metadata; we fill in the resolved Value.
	if m, ok := result.(pickerModel); ok && m.chosen != nil {
		runCtx := ctx
		runCtx.Value = m.value
		if err := m.chosen.run(runCtx); err != nil {
			fmt.Fprintln(os.Stderr, "herdr-plus: action failed:", err)
		}
	}

	closeSelf(selfPane)
}

// closeSelf asks herdr to close the picker's pane. Failures are ignored: if we
// cannot reach the socket there is nothing useful to do from a pane that is
// about to go away anyway.
func closeSelf(paneID string) {
	if paneID == "" {
		return
	}
	client, err := newHerdrClient()
	if err != nil {
		return
	}
	_ = client.closePane(paneID)
}
