//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// projectsTitle is the heading shown across the top of the projects browser.
const projectsTitle = "Herdr Plus · Projects"

// newWorkspaceTitle replaces it when the browser is shown over a brand-new,
// empty workspace, so it is obvious the choice is about what that workspace
// becomes rather than an extra workspace appearing from nowhere.
const newWorkspaceTitle = "Herdr Plus · New workspace"

// docsURL is where the empty-state points for "more documentation". Full docs
// live elsewhere later; for now the repo is the home of everything.
const docsURL = "https://github.com/cloudmanic/herdr-plus"

// projectsHeaderLines is how many screen lines precede the embedded fuzzyList in
// the projects browser: the full-width title bar and the blank line under it. A
// mouse click's screen row minus this offset is the list-local line for clickRow.
const projectsHeaderLines = 2

// projectsChromeLines is every fixed line around the list's body in
// browserView: the title bar and its blank line (projectsHeaderLines), the
// query/prompt block (listPromptLines), the minimum one-line gap above the
// detail bar, the four-line detail box, and the footer. The list's body budget
// is the pane height minus this, so the browser always fits the pane instead of
// overflowing it when there are more projects than rows.
const projectsChromeLines = projectsHeaderLines + listPromptLines + 1 + 4 + 1

// listViewport applies the pane size to the embedded list: the body-line budget
// left after the fixed chrome, and the full pane width as the row cap.
func (m *projectsModel) listViewport() {
	budget := m.height - projectsChromeLines
	if budget < 1 {
		budget = 1
	}
	m.list.setViewport(budget, m.width)
}

type projectsMode int

const (
	modeList projectsMode = iota
	modeBranch
	modePath
)

// Projects-browser styles. These build on the shared palette in styles.go
// (titleStyle, nameStyle, descStyle, footerStyle, …); here we add the few extra
// pieces the full-screen browser needs.
var (
	// headerBarStyle is the full-width purple title bar across the top.
	headerBarStyle = titleStyle

	// detailBoxStyle frames the bottom bar that previews the highlighted project.
	detailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("5")).
			Padding(0, 1)

	// dirIconStyle / pathStyle render the "📁 <working dir>" line of the detail bar.
	dirIconStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	pathStyle    = lipgloss.NewStyle()
	tabNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
	dotStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	// Empty-state styles: a centered onboarding card.
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("5")).
			Padding(1, 3)
	cardTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)
	bodyStyle      = lipgloss.NewStyle()
	pathHintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	codeStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	linkStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Underline(true)
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
	chosen *Project

	mode         projectsMode
	branchInput  textinput.Model
	branchPrefix string
	worktree     bool
	branch       string
	quitting     bool

	// Path-prompt state for projects with working_dir = "{prompt}": the input,
	// the last validation error, and whether the worktree branch prompt should
	// follow once a directory is accepted (the ctrl+g flow).
	pathInput       textinput.Model
	pathErr         string
	branchAfterPath bool

	// newWorkspace marks the browser as the one shown over a brand-new, empty
	// workspace (see asNewWorkspacePicker). It changes only the wording — the
	// list, the keys, and the choice it produces are identical.
	newWorkspace bool
}

// asNewWorkspacePicker returns the model presented as the new-workspace picker:
// its own title, and a footer that spells out what esc leaves behind — the empty
// workspace herdr just made. Cancelling has to read as a real option there, since
// it is how you still get a plain workspace with the feature enabled.
func (m projectsModel) asNewWorkspacePicker() projectsModel {
	m.newWorkspace = true
	return m
}

// headerTitle is the heading for the current presentation.
func (m projectsModel) headerTitle() string {
	if m.newWorkspace {
		return newWorkspaceTitle
	}
	return projectsTitle
}

// cancelHint describes what esc does in the current presentation.
func (m projectsModel) cancelHint() string {
	if m.newWorkspace {
		return "esc keep empty workspace"
	}
	return "esc quit"
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
func newProjectsModel(projects []Project, projectsDir, branchPrefix string) projectsModel {
	ordered, items := orderProjectsByGroup(projects)
	branchInput := textinput.New()
	branchInput.Prompt = ""
	branchInput.Placeholder = "empty → generated name"
	pathInput := textinput.New()
	pathInput.Prompt = ""
	pathInput.Placeholder = "~/path/to/project"
	return projectsModel{
		projects:     ordered,
		list:         newFuzzyList("Type to filter projects…", items),
		projectsDir:  projectsDir,
		branchInput:  branchInput,
		branchPrefix: branchPrefix,
		pathInput:    pathInput,
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
	if m.mode == modeBranch {
		return m.updateBranch(msg)
	}
	if m.mode == modePath {
		return m.updatePath(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.listViewport()
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
		case "ctrl+g":
			return m.promptWorktreeBranch()
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

// updateBranch handles input while the worktree branch prompt is showing (the
// modeBranch state entered by ctrl+g). Enter confirms: it marks the choice as a
// worktree and resolves the typed name through the configured prefix, then quits
// so runProjectsUI opens it. Esc backs out to the list, clearing the pending
// choice. Every other key — and the cursor-blink tick — flows to the text input.
func (m projectsModel) updateBranch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.listViewport()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.chosen == nil {
				m.mode = modeList
				return m, nil
			}
			m.worktree = true
			m.branch = resolveWorktreeBranch(m.branchInput.Value(), m.branchPrefix)
			return m, tea.Quit
		case "esc":
			m.mode = modeList
			m.chosen = nil
			m.worktree = false
			m.branch = ""
			return m, nil
		}

		var cmd tea.Cmd
		m.branchInput, cmd = m.branchInput.Update(msg)
		return m, cmd

	case tea.MouseMsg:
		return m, nil
	}

	var cmd tea.Cmd
	m.branchInput, cmd = m.branchInput.Update(msg)
	return m, cmd
}

// activateProject records the highlighted project as the chosen one and signals
// a quit so its workspace gets built — or, for a project that prompts for its
// directory, switches into path-entry mode first. Shared by the enter key and a
// left-click; activating with nothing selectable is a no-op.
func (m projectsModel) activateProject() (tea.Model, tea.Cmd) {
	idx := m.list.selectedIndex()
	if idx < 0 {
		return m, nil
	}
	p := m.projects[idx]
	m.chosen = &p
	if p.promptsForDir() {
		return m.promptProjectDir(false)
	}
	return m, tea.Quit
}

// promptWorktreeBranch is the ctrl+g entry point: it records the highlighted
// project and switches into branch-entry mode — via the path prompt first when
// the project asks for its directory. With nothing selectable it is a no-op.
func (m projectsModel) promptWorktreeBranch() (tea.Model, tea.Cmd) {
	idx := m.list.selectedIndex()
	if idx < 0 {
		return m, nil
	}
	p := m.projects[idx]
	m.chosen = &p
	if p.promptsForDir() {
		return m.promptProjectDir(true)
	}
	return m.enterBranchMode()
}

// enterBranchMode resets and focuses the branch input and switches the browser
// into its branch-entry state. Shared by ctrl+g and the path prompt's
// continue-to-branch step, so both arrive with a clean input.
func (m projectsModel) enterBranchMode() (tea.Model, tea.Cmd) {
	m.worktree = false
	m.branch = ""
	m.branchInput.SetValue("")
	m.branchInput.Prompt = ""
	m.branchInput.Placeholder = "empty → generated name"
	cmd := m.branchInput.Focus()
	m.mode = modeBranch
	return m, cmd
}

// promptProjectDir switches the browser into path-entry mode for the already
// chosen project (one with working_dir = "{prompt}"). branchAfter carries
// whether the worktree branch prompt should follow once a directory is
// accepted (the ctrl+g flow).
func (m projectsModel) promptProjectDir(branchAfter bool) (tea.Model, tea.Cmd) {
	m.branchAfterPath = branchAfter
	m.pathErr = ""
	m.pathInput.SetValue("")
	cmd := m.pathInput.Focus()
	m.mode = modePath
	return m, cmd
}

// updatePath handles input while the project-directory prompt is showing (the
// modePath state a "{prompt}" project enters when activated). Enter accepts the
// typed path when it names an existing directory: the path is stamped into the
// chosen project as its working directory and the project takes the
// directory's basename as its name, so the workspace label says what it holds.
// It then quits to open the workspace, or continues to the worktree branch
// prompt in the ctrl+g flow. An invalid path shows an inline error and keeps
// the prompt up. Esc backs out to the list, clearing the pending choice.
func (m projectsModel) updatePath(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.chosen == nil {
				m.mode = modeList
				return m, nil
			}
			if strings.TrimSpace(m.pathInput.Value()) == "" {
				m.pathErr = "enter a project path"
				return m, nil
			}
			dir, err := expandPath(m.pathInput.Value())
			if err != nil {
				m.pathErr = err.Error()
				return m, nil
			}
			if abs, err := filepath.Abs(dir); err == nil {
				dir = abs
			}
			if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
				m.pathErr = "not a directory: " + dir
				return m, nil
			}
			m.chosen.WorkingDir = dir
			m.chosen.Name = filepath.Base(dir)
			if m.branchAfterPath {
				return m.enterBranchMode()
			}
			return m, tea.Quit
		case "esc":
			m.mode = modeList
			m.chosen = nil
			m.pathErr = ""
			m.branchAfterPath = false
			return m, nil
		}

		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return m, cmd

	case tea.MouseMsg:
		return m, nil
	}

	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	return m, cmd
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
	if m.mode == modeBranch {
		return m.branchView(w, h)
	}
	if m.mode == modePath {
		return m.pathView(w, h)
	}
	return m.browserView(w, h)
}

// browserView lays out the populated projects browser: a full-width title bar up
// top, the fuzzy list below it, and the highlighted project's detail bar pinned
// just above the footer at the bottom of the screen.
func (m projectsModel) browserView(w, h int) string {
	header := headerBarStyle.Width(w).Render(m.headerTitle())

	body := m.list.view("no matching projects")

	detail := m.detailBar(w)
	footer := footerStyle.Render("  ↑/↓ move · type to filter · click/enter open · ctrl+g worktree · " + m.cancelHint())

	top := header + "\n\n" + body
	bottom := detail + "\n" + footer

	// Pin the detail bar + footer to the bottom by padding the space between.
	gap := h - lipgloss.Height(top) - lipgloss.Height(bottom)
	if gap < 1 {
		gap = 1
	}
	return top + strings.Repeat("\n", gap) + bottom
}

// branchView renders the worktree branch prompt: the chosen project's name and
// working directory above a single-line input for the optional branch name, with
// a footer explaining enter/esc. It backs the modeBranch state.
func (m projectsModel) branchView(w, h int) string {
	header := headerBarStyle.Width(w).Render(m.headerTitle())

	name, dir := "", ""
	if m.chosen != nil {
		name = m.chosen.Name
		dir = m.chosen.displayWorkingDir()
	}

	body := nameStyle.Render(name) + "\n" +
		dirIconStyle.Render("📁 ") + pathStyle.Render(truncate(dir, w-6)) + "\n\n" +
		promptStyle.Render("❯ ") + m.branchInput.View()
	footer := footerStyle.Render("  enter create · esc back")

	top := header + "\n\n" + body
	gap := h - lipgloss.Height(top) - lipgloss.Height(footer)
	if gap < 1 {
		gap = 1
	}
	return top + strings.Repeat("\n", gap) + footer
}

// pathView renders the project-directory prompt for a "{prompt}" project: the
// chosen project's name above a single-line input for the path (~ and $VARS
// expand like working_dir), plus any validation error from the last attempt.
// It backs the modePath state.
func (m projectsModel) pathView(w, h int) string {
	header := headerBarStyle.Width(w).Render(projectsTitle)

	name := ""
	if m.chosen != nil {
		name = m.chosen.Name
	}

	body := nameStyle.Render(name) + "\n" +
		dirIconStyle.Render("📁 ") + pathStyle.Render("directory to open (~ and $VARS expand)") + "\n\n" +
		promptStyle.Render("❯ ") + m.pathInput.View()
	if m.pathErr != "" {
		body += "\n" + errorStyle.Render(truncate(m.pathErr, w-4))
	}
	action := "open"
	if m.branchAfterPath {
		action = "continue"
	}
	footer := footerStyle.Render("  enter " + action + " · esc back")

	top := header + "\n\n" + body
	gap := h - lipgloss.Height(top) - lipgloss.Height(footer)
	if gap < 1 {
		gap = 1
	}
	return top + strings.Repeat("\n", gap) + footer
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

	dirText := p.displayWorkingDir()
	if p.promptsForDir() {
		dirText = "asks for a path"
	}
	dirLine := dirIconStyle.Render("📁 ") + pathStyle.Render(truncate(dirText, inner-3))

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
