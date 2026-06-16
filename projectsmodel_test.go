//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// sampleProjects returns two pre-sorted projects for the model tests.
func sampleProjects() []Project {
	return []Project{
		{Name: "Alpha", Description: "first one", WorkingDir: "/srv/alpha", Tabs: []ProjectTab{{Name: "run", Command: "make serve"}}},
		{Name: "Bravo", Description: "second one", WorkingDir: "/srv/bravo", Tabs: []ProjectTab{{Name: "edit", Command: "vim"}, {Name: "shell"}}},
	}
}

// step feeds one message to the model and returns the updated projectsModel.
func step(t *testing.T, m projectsModel, msg tea.Msg) projectsModel {
	t.Helper()
	updated, _ := m.Update(msg)
	pm, ok := updated.(projectsModel)
	if !ok {
		t.Fatalf("Update returned %T, want projectsModel", updated)
	}
	return pm
}

// TestNewProjectsModelItems confirms each project becomes a selectable list row
// carrying its name, description, and original index.
func TestNewProjectsModelItems(t *testing.T) {
	m := newProjectsModel(sampleProjects(), "/cfg/projects")
	if len(m.list.items) != 2 {
		t.Fatalf("got %d list items, want 2", len(m.list.items))
	}
	if !m.list.items[0].selectable || m.list.items[0].name != "Alpha" || m.list.items[0].desc != "first one" || m.list.items[0].ref != 0 {
		t.Fatalf("item0 = %+v", m.list.items[0])
	}
	if m.list.items[1].ref != 1 {
		t.Fatalf("item1 ref = %d, want 1", m.list.items[1].ref)
	}
}

// groupedProjects returns four name-sorted projects spanning two named groups
// and one group-less project, the way loadProjects would hand them over.
func groupedProjects() []Project {
	return []Project{
		{Name: "Acme API", Group: "Acme", Tabs: []ProjectTab{{Name: "run", Command: "make"}}},
		{Name: "Acme Web", Group: "Acme", Tabs: []ProjectTab{{Name: "run", Command: "make"}}},
		{Name: "Loose", Tabs: []ProjectTab{{Name: "run", Command: "make"}}},
		{Name: "Zeta", Group: "beta", Tabs: []ProjectTab{{Name: "run", Command: "make"}}},
	}
}

// TestOrderProjectsByGroupUngrouped confirms that with no project declaring a
// group, the rows are a plain, heading-less list whose refs index straight into
// the unchanged input.
func TestOrderProjectsByGroupUngrouped(t *testing.T) {
	in := sampleProjects() // neither sample sets a group
	ordered, items := orderProjectsByGroup(in)

	if len(ordered) != len(in) || ordered[0].Name != "Alpha" || ordered[1].Name != "Bravo" {
		t.Fatalf("ungrouped order changed: %+v", ordered)
	}
	if len(items) != 2 {
		t.Fatalf("got %d rows, want 2 (no headings)", len(items))
	}
	for i, it := range items {
		if !it.selectable || it.ref != i {
			t.Fatalf("row %d = %+v, want a selectable row with ref %d", i, it, i)
		}
	}
}

// TestOrderProjectsByGroupHeadings confirms that once any project sets a group,
// the rows become heading-bracketed: named groups in case-insensitive
// alphabetical order ("Acme" before "beta"), then an "Ungrouped" heading, with
// each selectable row's ref pointing at the right project in the returned order.
func TestOrderProjectsByGroupHeadings(t *testing.T) {
	ordered, items := orderProjectsByGroup(groupedProjects())

	// The display order interleaves groups: Acme's two, then beta's one, then the
	// group-less one last.
	wantOrder := []string{"Acme API", "Acme Web", "Zeta", "Loose"}
	if len(ordered) != len(wantOrder) {
		t.Fatalf("got %d ordered projects, want %d", len(ordered), len(wantOrder))
	}
	for i, name := range wantOrder {
		if ordered[i].Name != name {
			t.Fatalf("ordered[%d] = %q, want %q", i, ordered[i].Name, name)
		}
	}

	// The row sequence: heading, its projects, … then the Ungrouped heading.
	var headings []string
	for _, it := range items {
		if !it.selectable {
			headings = append(headings, it.name)
			continue
		}
		// Every selectable row's ref must resolve to the project of the same name.
		if ordered[it.ref].Name != it.name {
			t.Fatalf("row %q ref %d resolves to %q", it.name, it.ref, ordered[it.ref].Name)
		}
	}
	wantHeadings := []string{"Acme", "beta", ungroupedHeading}
	if strings.Join(headings, ",") != strings.Join(wantHeadings, ",") {
		t.Fatalf("headings = %v, want %v", headings, wantHeadings)
	}
}

// TestProjectsModelGroupedSelect confirms the grouped browser is fully navigable:
// the cursor starts on the first project under the first heading (skipping the
// heading row), and entering opens the project its ref points at.
func TestProjectsModelGroupedSelect(t *testing.T) {
	m := newProjectsModel(groupedProjects(), "/cfg/projects")
	m = step(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	// Cursor parks on the first selectable row, not the "Acme" heading.
	if got := m.list.selectedIndex(); got != 0 || m.projects[got].Name != "Acme API" {
		t.Fatalf("initial selection ref = %d, want 0 (Acme API)", got)
	}

	// Down once lands on Acme Web; once more skips the "beta" heading to Zeta.
	m = step(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = step(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.chosen == nil || m.chosen.Name != "Zeta" {
		t.Fatalf("after two downs + enter, chosen = %v, want Zeta", m.chosen)
	}
}

// TestProjectsModelGroupedFilterDropsHeadings confirms search is unchanged: a
// query collapses the grouped view to a single ranked list with no headings, and
// enter opens the lone match.
func TestProjectsModelGroupedFilterDropsHeadings(t *testing.T) {
	m := newProjectsModel(groupedProjects(), "/cfg/projects")
	m = step(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("loose")})

	for _, it := range m.list.filtered {
		if !it.item.selectable {
			t.Fatalf("a heading row survived filtering: %+v", it.item)
		}
	}
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.chosen == nil || m.chosen.Name != "Loose" {
		t.Fatalf("filtering to 'loose' then enter chose %v, want Loose", m.chosen)
	}
}

// TestProjectsModelEnterSelects confirms pressing enter records the highlighted
// project and signals a quit.
func TestProjectsModelEnterSelects(t *testing.T) {
	m := newProjectsModel(sampleProjects(), "/cfg/projects")
	m = step(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m = step(t, m, tea.KeyMsg{Type: tea.KeyDown}) // move to Bravo
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.chosen == nil {
		t.Fatal("expected a chosen project, got nil")
	}
	if m.chosen.Name != "Bravo" {
		t.Fatalf("chosen = %q, want Bravo", m.chosen.Name)
	}
}

// TestProjectsModelEscCancels confirms esc records no selection.
func TestProjectsModelEscCancels(t *testing.T) {
	m := newProjectsModel(sampleProjects(), "/cfg/projects")
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEsc})

	if m.chosen != nil {
		t.Fatalf("esc should not choose a project, got %q", m.chosen.Name)
	}
	if !m.quitting {
		t.Fatal("esc should set quitting")
	}
}

// TestProjectsModelFilterThenSelect confirms typing narrows the list and enter
// then opens the single remaining match.
func TestProjectsModelFilterThenSelect(t *testing.T) {
	m := newProjectsModel(sampleProjects(), "/cfg/projects")
	m = step(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("brav")})
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.chosen == nil || m.chosen.Name != "Bravo" {
		t.Fatalf("after filtering to 'brav', chosen = %v, want Bravo", m.chosen)
	}
}

// TestProjectsModelMouseClickOpens confirms a left-button release over a project
// row opens it — the click counterpart to enter. With the title bar and query
// line above, the two projects sit at screen rows 4 (Alpha) and 5 (Bravo).
func TestProjectsModelMouseClickOpens(t *testing.T) {
	m := newProjectsModel(sampleProjects(), "/cfg/projects")
	m = step(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m = step(t, m, tea.MouseMsg{
		Action: tea.MouseActionRelease,
		Button: tea.MouseButtonLeft,
		Y:      5,
	})

	if m.chosen == nil || m.chosen.Name != "Bravo" {
		t.Fatalf("clicking the Bravo row chose %v, want Bravo", m.chosen)
	}
}

// TestProjectsModelMouseWheelMoves confirms the scroll wheel walks the highlight
// without opening anything.
func TestProjectsModelMouseWheelMoves(t *testing.T) {
	m := newProjectsModel(sampleProjects(), "/cfg/projects")
	m = step(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m = step(t, m, tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})

	if m.chosen != nil {
		t.Fatal("the wheel should not open a project")
	}
	if got := m.list.selectedIndex(); got != 1 {
		t.Fatalf("after wheel down selected ref = %d, want 1 (Bravo)", got)
	}
}

// TestProjectsModelBrowserView confirms the populated view shows the header, the
// project names, and the highlighted project's working directory in the detail
// bar.
func TestProjectsModelBrowserView(t *testing.T) {
	m := newProjectsModel(sampleProjects(), "/cfg/projects")
	m = step(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	view := m.View()

	for _, want := range []string{"Alpha", "Bravo", "/srv/alpha"} {
		if !strings.Contains(view, want) {
			t.Fatalf("browser view missing %q:\n%s", want, view)
		}
	}
}

// TestProjectsModelEmptyState confirms that with no projects the onboarding card
// renders (with the config path and docs link) and any exit key closes it.
func TestProjectsModelEmptyState(t *testing.T) {
	m := newProjectsModel(nil, "/cfg/projects")
	m = step(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})
	view := m.View()

	for _, want := range []string{"Welcome to Herdr Plus", "/cfg/projects", docsURL} {
		if !strings.Contains(view, want) {
			t.Fatalf("empty-state view missing %q:\n%s", want, view)
		}
	}

	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.quitting {
		t.Fatal("empty-state should quit on a key press")
	}
	if m.chosen != nil {
		t.Fatal("empty-state must never choose a project")
	}
}
