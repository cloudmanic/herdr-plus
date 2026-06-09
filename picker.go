//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

// Palette. A small, cohesive set of colors for a clean dark-terminal look.
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

// scored is a command paired with the indexes of the characters in its name
// that matched the current fuzzy query, used for highlighting.
type scored struct {
	cmd     Command
	matched []int
}

// pickerModel holds the state of the fuzzy-finder TUI.
type pickerModel struct {
	input    textinput.Model // the query box
	all      []Command       // every available command
	filtered []scored        // commands matching the current query, in rank order
	cursor   int             // index into filtered of the highlighted row
	width    int
	height   int
	chosen   *Command // set when the user selects a command
	quitting bool
}

// newPickerModel builds the initial TUI state with every command visible.
func newPickerModel() pickerModel {
	ti := textinput.New()
	ti.Placeholder = "Type to filter…"
	ti.Prompt = ""
	ti.Focus()

	m := pickerModel{
		input: ti,
		all:   commands(),
	}
	m.filter()
	return m
}

// filter recomputes the filtered list from the current query. An empty query
// shows every command in its natural order. Matching runs against the command
// name and description together so a URL fragment also narrows the list, while
// only matches that land inside the name are highlighted.
func (m *pickerModel) filter() {
	q := strings.TrimSpace(m.input.Value())
	m.filtered = m.filtered[:0]

	if q == "" {
		for _, c := range m.all {
			m.filtered = append(m.filtered, scored{cmd: c})
		}
	} else {
		haystacks := make([]string, len(m.all))
		nameLens := make([]int, len(m.all))
		for i, c := range m.all {
			haystacks[i] = c.Name + "  " + c.Description
			nameLens[i] = len(c.Name)
		}
		for _, mt := range fuzzy.Find(q, haystacks) {
			var inName []int
			for _, idx := range mt.MatchedIndexes {
				if idx < nameLens[mt.Index] {
					inName = append(inName, idx)
				}
			}
			m.filtered = append(m.filtered, scored{cmd: m.all[mt.Index], matched: inName})
		}
	}

	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// Init implements tea.Model and starts the cursor blinking.
func (m pickerModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles key presses and window-size changes. Navigation keys move the
// cursor, enter selects, esc/ctrl+c cancels, and any other key edits the query.
func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 {
				c := m.filtered[m.cursor].cmd
				m.chosen = &c
			}
			return m, tea.Quit
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		}
	}

	// Anything else edits the query, after which we re-filter the list.
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.filter()
	return m, cmd
}

// View renders the title bar, query line, match count, result list, and footer
// hints.
func (m pickerModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title bar.
	b.WriteString(titleStyle.Render("⚡ Quick Actions"))
	b.WriteString("\n\n")

	// Query line and match count.
	b.WriteString(promptStyle.Render("❯ "))
	b.WriteString(m.input.View())
	b.WriteString("   ")
	b.WriteString(countStyle.Render(fmt.Sprintf("%d/%d", len(m.filtered), len(m.all))))
	b.WriteString("\n\n")

	// Results.
	if len(m.filtered) == 0 {
		b.WriteString(descStyle.Render("  no matching commands"))
		b.WriteString("\n")
	}
	for i, s := range m.filtered {
		selected := i == m.cursor
		if selected {
			b.WriteString(barStyle.Render("▌ "))
		} else {
			b.WriteString("  ")
		}
		b.WriteString(highlightName(s.cmd.Name, s.matched, selected))
		b.WriteString("  ")
		b.WriteString(descStyle.Render(s.cmd.Description))
		b.WriteString("\n")
	}

	// Footer.
	b.WriteString("\n")
	b.WriteString(footerStyle.Render("↑/↓ move • enter run • esc cancel"))

	return b.String()
}

// highlightName renders a command name with the fuzzy-matched characters
// emphasized. matched holds byte indexes into the name string (the names are
// ASCII, so byte and rune indexes coincide).
func highlightName(name string, matched []int, selected bool) string {
	base := nameStyle
	if selected {
		base = nameSelStyle
	}
	if len(matched) == 0 {
		return base.Render(name)
	}

	set := make(map[int]bool, len(matched))
	for _, idx := range matched {
		set[idx] = true
	}

	var b strings.Builder
	for i, r := range name {
		if set[i] {
			b.WriteString(matchStyle.Render(string(r)))
		} else {
			b.WriteString(base.Render(string(r)))
		}
	}
	return b.String()
}

// runPicker renders the fuzzy-finder TUI, runs the selected command, and then
// closes its own herdr pane so focus returns to the pane it was launched from.
// args[0], when present, is the pane id to close; it falls back to
// HERDR_PANE_ID.
func runPicker(args []string) {
	selfPane := ""
	if len(args) > 0 {
		selfPane = args[0]
	}
	if selfPane == "" {
		selfPane = os.Getenv("HERDR_PANE_ID")
	}

	p := tea.NewProgram(newPickerModel(), tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "herdr-quick-actions:", err)
	}

	// Run the chosen command before tearing down the pane. "open <url>" returns
	// as soon as the URL is handed to the browser, so this does not block.
	if m, ok := result.(pickerModel); ok && m.chosen != nil {
		runCommand(m.chosen.Run)
	}

	closeSelf(selfPane)
}

// runCommand executes a quick action's shell command. The command is run
// through the shell so future actions can use arguments, pipes, and the like.
func runCommand(cmdline string) {
	cmd := exec.Command("sh", "-c", cmdline)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
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
