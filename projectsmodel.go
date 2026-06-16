//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// projectsTitle is the heading shown across the top of the projects browser.
const projectsTitle = "Herdr Plus · Projects"

// docsURL is where the empty-state points for "more documentation". Full docs
// live elsewhere later; for now the repo is the home of everything.
const docsURL = "https://github.com/cloudmanic/herdr-plus"

// projectsHeaderLines is how many screen lines precede the embedded fuzzyList in
// the projects browser: the full-width title bar and the blank line under it. A
// mouse click's screen row minus this offset is the list-local line for clickRow.
const projectsHeaderLines = 2

// Projects-browser styles. These build on the shared palette in styles.go
// (titleStyle, nameStyle, descStyle, footerStyle, …); here we add the few extra
// pieces the full-screen browser needs.
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

// projectsModel is the full-screen projects browser. It is a thin shell around
// the shared fuzzyList: arrow or type to find a project, enter to open it, esc to
// back out. When there are no projects it renders an onboarding card instead of
// the list.
type projectsModel struct {
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

// ungroupedHeading labels the catch-all bucket for projects that declare no
// group. It is only ever shown when at least one other project does declare one;
// when no project sets a group the browser has no headings at all.
const ungroupedHeading = "Ungrouped"

// newProjectsModel builds the initial TUI state over the loaded projects.
// projectsDir is shown in the empty-state so the user knows where to add files.
// Projects are arranged into group headings (see orderProjectsByGroup) and the
// resulting display order is stored on the model so each list row's ref indexes
// straight back into it.
func newProjectsModel(projects []Project, projectsDir string) projectsModel {
	ordered, items := orderProjectsByGroup(projects)
	return projectsModel{
		projects:    ordered,
		list:        newFuzzyList("Type to filter projects…", items),
		projectsDir: projectsDir,
	}
}

// orderProjectsByGroup arranges projects for the browser and returns them in
// display order alongside the matching list rows. Grouping engages only when at
// least one project declares a group: named groups come first in
// case-insensitive alphabetical order, each introduced by a non-selectable
// heading row, followed by any group-less projects under an "Ungrouped" heading.
// Within every group the input order (name-sorted by loadProjects) is preserved,
// so a client's projects stay alphabetized under their heading. Each selectable
// row's ref indexes into the returned slice, so the caller stores that slice and
// looks a project up by ref directly. When no project declares a group, the input
// is returned unchanged with a plain, heading-less list. Filtering is unaffected:
// the fuzzyList drops heading rows while a query is active, collapsing back to one
// ranked list.
func orderProjectsByGroup(projects []Project) ([]Project, []listItem) {
	// Does anything opt into grouping? If not, emit a plain list whose refs index
	// straight into the unchanged input.
	grouped := false
	for _, p := range projects {
		if strings.TrimSpace(p.Group) != "" {
			grouped = true
			break
		}
	}
	if !grouped {
		items := make([]listItem, len(projects))
		for i, p := range projects {
			items[i] = listItem{name: p.Name, desc: p.Description, selectable: true, ref: i}
		}
		return projects, items
	}

	// Partition into named groups (first-seen order recorded for sorting) plus the
	// group-less remainder, preserving each project's incoming name order.
	byGroup := map[string][]Project{}
	var groupNames []string
	var ungrouped []Project
	for _, p := range projects {
		g := strings.TrimSpace(p.Group)
		if g == "" {
			ungrouped = append(ungrouped, p)
			continue
		}
		if _, seen := byGroup[g]; !seen {
			groupNames = append(groupNames, g)
		}
		byGroup[g] = append(byGroup[g], p)
	}

	// Sort group headings case-insensitively, falling back to the raw label so two
	// groups differing only in case still order deterministically.
	sort.SliceStable(groupNames, func(i, j int) bool {
		li, lj := strings.ToLower(groupNames[i]), strings.ToLower(groupNames[j])
		if li == lj {
			return groupNames[i] < groupNames[j]
		}
		return li < lj
	})

	ordered := make([]Project, 0, len(projects))
	items := make([]listItem, 0, len(projects)+len(groupNames)+1)

	// Emit each named group's heading followed by its projects; ref tracks the
	// running index into ordered so every row points back at its project.
	for _, name := range groupNames {
		items = append(items, listItem{name: name})
		for _, p := range byGroup[name] {
			items = append(items, listItem{name: p.Name, desc: p.Description, selectable: true, ref: len(ordered)})
			ordered = append(ordered, p)
		}
	}

	// Group-less projects trail under the catch-all heading.
	if len(ungrouped) > 0 {
		items = append(items, listItem{name: ungroupedHeading})
		for _, p := range ungrouped {
			items = append(items, listItem{name: p.Name, desc: p.Description, selectable: true, ref: len(ordered)})
			ordered = append(ordered, p)
		}
	}

	return ordered, items
}

// Init implements tea.Model and starts the cursor blinking.
func (m projectsModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update routes key presses; everything else (window sizes, the blink tick) is
// forwarded to the query box so the cursor keeps blinking and text keeps flowing.
func (m projectsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			return m.activateProject()
		}

		cmd := m.list.editQuery(msg)
		return m, cmd

	case tea.MouseMsg:
		// The onboarding card (no projects) has nothing to click.
		if len(m.projects) == 0 {
			return m, nil
		}
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.list.moveUp()
		case tea.MouseButtonWheelDown:
			m.list.moveDown()
		case tea.MouseButtonLeft:
			// Move the highlight to the clicked row, opening it on release — the
			// natural completion of a click.
			if m.list.clickRow(msg.Y-projectsHeaderLines) && msg.Action == tea.MouseActionRelease {
				return m.activateProject()
			}
		}
		return m, nil
	}

	// Non-key messages (e.g. the blink tick) keep the input alive.
	cmd := m.list.editQuery(msg)
	return m, cmd
}

// activateProject records the highlighted project as the chosen one and signals
// a quit so its workspace gets built. Shared by the enter key and a left-click;
// activating with nothing selectable is a no-op.
func (m projectsModel) activateProject() (tea.Model, tea.Cmd) {
	idx := m.list.selectedIndex()
	if idx < 0 {
		return m, nil
	}
	p := m.projects[idx]
	m.chosen = &p
	return m, tea.Quit
}

// View renders the screen for the current state: the onboarding card when there
// are no projects, otherwise the header / list / detail-bar / footer layout.
func (m projectsModel) View() string {
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
func (m projectsModel) browserView(w, h int) string {
	header := headerBarStyle.Width(w).Render(projectsTitle)

	body := m.list.view("no matching projects")

	detail := m.detailBar(w)
	footer := footerStyle.Render("  ↑/↓ move · type to filter · click/enter open · esc quit")

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
func (m projectsModel) detailBar(w int) string {
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
func (m projectsModel) emptyView(w, h int) string {
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
