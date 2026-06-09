//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// docsURL is where the empty-state points for "more documentation". Full docs
// will live elsewhere later; for now the repo is the home of everything.
const docsURL = "https://github.com/cloudmanic/herdr-plus"

// Control-mode styles. These build on the shared palette declared in picker.go
// (titleStyle, nameStyle, descStyle, footerStyle, …); here we add the few extra
// pieces the full-screen projects browser needs.
var (
	// headerBarStyle is the full-width purple title bar across the top.
	headerBarStyle = titleStyle

	// detailBoxStyle frames the bottom bar that previews the highlighted project.
	detailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#A78BFA")).
			Padding(0, 1)

	// dirIconStyle / pathStyle render the "📁 <working dir>" line of the detail bar.
	dirIconStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))
	pathStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB"))
	tabNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#C4B5FD"))
	dotStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5563"))

	// Empty-state styles: a centered onboarding card.
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#A78BFA")).
			Padding(1, 3)
	cardTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
	bodyStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB"))
	pathHintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F2A900")).Bold(true)
	codeStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	linkStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#67E8F9")).Underline(true)
)

// controlModel is the full-screen projects browser shown in the "Herdr Plus"
// control workspace. It is a thin shell around the shared fuzzyList: arrow or
// type to find a project, enter to open it, esc to back out. When there are no
// projects it renders an onboarding card instead of the list.
type controlModel struct {
	projects    []Project
	list        fuzzyList
	projectsDir string

	width  int
	height int

	// chosen is the project to open, read back after the program exits; nil when
	// the user cancelled.
	chosen   *Project
	quitting bool
}

// newControlModel builds the initial TUI state over the loaded projects.
// projectsDir is shown in the empty-state so the user knows where to add files.
func newControlModel(projects []Project, projectsDir string) controlModel {
	items := make([]listItem, len(projects))
	for i, p := range projects {
		items[i] = listItem{name: p.Name, desc: p.Description, selectable: true, ref: i}
	}
	return controlModel{
		projects:    projects,
		list:        newFuzzyList("Type to filter projects…", items),
		projectsDir: projectsDir,
	}
}

// Init implements tea.Model and starts the cursor blinking.
func (m controlModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update routes key presses; everything else (window sizes, the blink tick) is
// forwarded to the query box so the cursor keeps blinking and text keeps flowing.
func (m controlModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// With no projects, the screen is just an onboarding card: any exit key
		// closes it.
		if len(m.projects) == 0 {
			switch msg.String() {
			case "ctrl+c", "esc", "q", "enter":
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up", "ctrl+p":
			m.list.moveUp()
			return m, nil
		case "down", "ctrl+n":
			m.list.moveDown()
			return m, nil
		case "enter":
			idx := m.list.selectedIndex()
			if idx < 0 {
				return m, nil
			}
			p := m.projects[idx]
			m.chosen = &p
			return m, tea.Quit
		}

		cmd := m.list.editQuery(msg)
		return m, cmd
	}

	// Non-key messages (e.g. the blink tick) keep the input alive.
	cmd := m.list.editQuery(msg)
	return m, cmd
}

// View renders the screen for the current state: the onboarding card when there
// are no projects, otherwise the header / list / detail-bar / footer layout.
func (m controlModel) View() string {
	if m.quitting {
		return ""
	}

	// Fall back to a sane size until the first WindowSizeMsg arrives.
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	if len(m.projects) == 0 {
		return m.emptyView(w, h)
	}
	return m.browserView(w, h)
}

// browserView lays out the populated projects browser: a full-width title bar up
// top, the fuzzy list below it, and the highlighted project's detail bar pinned
// just above the footer at the bottom of the screen.
func (m controlModel) browserView(w, h int) string {
	header := headerBarStyle.Width(w).Render(ModeControl.Title)

	body := m.list.view("no matching projects")

	detail := m.detailBar(w)
	footer := footerStyle.Render("  ↑/↓ move · type to filter · enter open · esc quit")

	top := header + "\n\n" + body
	bottom := detail + "\n" + footer

	// Pin the detail bar + footer to the bottom by padding the space between.
	gap := h - lipgloss.Height(top) - lipgloss.Height(bottom)
	if gap < 1 {
		gap = 1
	}
	return top + strings.Repeat("\n", gap) + bottom
}

// detailBar renders the bordered preview of the currently highlighted project:
// its working directory and the ordered list of tab names. It updates live as
// the cursor moves.
func (m controlModel) detailBar(w int) string {
	// lipgloss Width counts content+padding; the 1-col border is added outside,
	// so Width(w-2) makes the box span (almost) the full screen width.
	box := detailBoxStyle.Width(w - 2)

	idx := m.list.selectedIndex()
	if idx < 0 {
		return box.Render(descStyle.Render("no matching project"))
	}
	p := m.projects[idx]

	// inner is the usable text width inside the box; keep a couple columns of
	// slack under the true content width so a long line never soft-wraps and
	// breaks the fixed two-line box.
	inner := w - 7
	if inner < 10 {
		inner = 10
	}

	dirLine := dirIconStyle.Render("📁 ") + pathStyle.Render(truncate(p.expandedWorkingDir(), inner-3))

	labels := p.tabLabels()
	styled := make([]string, len(labels))
	for i, n := range labels {
		styled[i] = tabNameStyle.Render(n)
	}
	tabsLine := strings.Join(styled, dotStyle.Render(" · "))
	tabsLine = truncateStyled(tabsLine, labels, inner)

	return box.Render(dirLine + "\n" + tabsLine)
}

// emptyView renders the onboarding card shown the first time, before any project
// files exist: what projects are, where to put them, a copy-paste example, and a
// docs link. It is centered in the full screen.
func (m controlModel) emptyView(w, h int) string {
	var b strings.Builder

	b.WriteString(cardTitleStyle.Render("Welcome to Herdr Plus · Projects"))
	b.WriteString("\n\n")
	b.WriteString(bodyStyle.Render("A project is a saved herdr workspace: a working directory and an"))
	b.WriteString("\n")
	b.WriteString(bodyStyle.Render("ordered set of tabs, each with a command to run on startup. Pick one"))
	b.WriteString("\n")
	b.WriteString(bodyStyle.Render("here and herdr-plus spins up the whole workspace for you."))
	b.WriteString("\n\n")
	b.WriteString(bodyStyle.Render("Create your first project — drop a .toml file in:"))
	b.WriteString("\n")
	b.WriteString(pathHintStyle.Render("  " + m.projectsDir))
	b.WriteString("\n\n")
	b.WriteString(descStyle.Render("Example (" + exampleFileName + "):"))
	b.WriteString("\n")
	b.WriteString(codeStyle.Render(indent(exampleProjectSnippet(), "  ")))
	b.WriteString("\n\n")
	b.WriteString(descStyle.Render("Docs: "))
	b.WriteString(linkStyle.Render(docsURL))

	card := cardStyle.Render(b.String())

	footer := footerStyle.Render("esc to close")
	content := lipgloss.JoinVertical(lipgloss.Center, card, "", footer)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}

// exampleFileName / exampleProjectTOML expose the bundled sample project so the
// empty-state and the repo share one source of truth (it is embedded by the
// //go:embed directive in config.go).
const exampleFileName = "options-cafe.toml"

func exampleProjectTOML() string {
	b, err := embeddedExamples.ReadFile("examples/projects/example.toml")
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(b), "\n")
}

// exampleProjectSnippet is the example trimmed for the empty-state card: full-line
// comments are dropped and runs of blank lines collapsed, so the card stays
// compact enough for shorter terminals while still derived from the one embedded
// source of truth. Inline comments (after a value) are kept — they teach.
func exampleProjectSnippet() string {
	var out []string
	blank := false
	for _, line := range strings.Split(exampleProjectTOML(), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if trimmed == "" {
			if blank || len(out) == 0 {
				continue // collapse consecutive blanks and skip a leading blank
			}
			blank = true
			out = append(out, "")
			continue
		}
		blank = false
		out = append(out, line)
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

// indent prefixes every line of s with prefix, used to inset the example block.
func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}

// truncate shortens a plain (unstyled) string to max display columns, ending it
// with an ellipsis when it had to cut. Used for the working-dir path.
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r))+1 > max {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

// truncateStyled keeps the styled tab-names line from overflowing the detail box.
// Styling makes display width hard to measure directly, so it falls back to the
// plain names: if they fit, the styled line is returned untouched; if not, it
// re-renders only as many names as fit, plus a "+N" tail.
func truncateStyled(styled string, names []string, max int) string {
	plain := strings.Join(names, " · ")
	if lipgloss.Width(plain) <= max {
		return styled
	}

	var shown []string
	width := 0
	for _, n := range names {
		add := lipgloss.Width(n)
		if len(shown) > 0 {
			add += 3 // " · "
		}
		if width+add > max-6 { // leave room for the "+N more" tail
			break
		}
		width += add
		shown = append(shown, tabNameStyle.Render(n))
	}
	tail := dotStyle.Render(" +" + strconv.Itoa(len(names)-len(shown)) + " more")
	return strings.Join(shown, dotStyle.Render(" · ")) + tail
}
