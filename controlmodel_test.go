//
// Date: 2026-06-09
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

// step feeds one message to the model and returns the updated controlModel.
func step(t *testing.T, m controlModel, msg tea.Msg) controlModel {
	t.Helper()
	updated, _ := m.Update(msg)
	cm, ok := updated.(controlModel)
	if !ok {
		t.Fatalf("Update returned %T, want controlModel", updated)
	}
	return cm
}

// TestNewControlModelItems confirms each project becomes a selectable list row
// carrying its name, description, and original index.
func TestNewControlModelItems(t *testing.T) {
	m := newControlModel(sampleProjects(), "/cfg/projects")
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

// TestControlModelEnterSelects confirms pressing enter records the highlighted
// project (the first, since the cursor starts at the top) and signals a quit.
func TestControlModelEnterSelects(t *testing.T) {
	m := newControlModel(sampleProjects(), "/cfg/projects")
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

// TestControlModelEscCancels confirms esc records no selection.
func TestControlModelEscCancels(t *testing.T) {
	m := newControlModel(sampleProjects(), "/cfg/projects")
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEsc})

	if m.chosen != nil {
		t.Fatalf("esc should not choose a project, got %q", m.chosen.Name)
	}
	if !m.quitting {
		t.Fatal("esc should set quitting")
	}
}

// TestControlModelFilterThenSelect confirms typing narrows the list and enter
// then opens the single remaining match.
func TestControlModelFilterThenSelect(t *testing.T) {
	m := newControlModel(sampleProjects(), "/cfg/projects")
	m = step(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("brav")})
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.chosen == nil || m.chosen.Name != "Bravo" {
		t.Fatalf("after filtering to 'brav', chosen = %v, want Bravo", m.chosen)
	}
}

// TestControlModelMouseClickOpens confirms a left-button release over a project
// row opens it — the click counterpart to enter. With the title bar and query
// line above, the two projects sit at screen rows 4 (Alpha) and 5 (Bravo).
func TestControlModelMouseClickOpens(t *testing.T) {
	m := newControlModel(sampleProjects(), "/cfg/projects")
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

// TestControlModelMouseWheelMoves confirms the scroll wheel walks the highlight
// without opening anything.
func TestControlModelMouseWheelMoves(t *testing.T) {
	m := newControlModel(sampleProjects(), "/cfg/projects")
	m = step(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m = step(t, m, tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})

	if m.chosen != nil {
		t.Fatal("the wheel should not open a project")
	}
	if got := m.list.selectedIndex(); got != 1 {
		t.Fatalf("after wheel down selected ref = %d, want 1 (Bravo)", got)
	}
}

// TestControlModelBrowserView confirms the populated view shows the header, the
// project names, and the highlighted project's working directory in the detail
// bar.
func TestControlModelBrowserView(t *testing.T) {
	m := newControlModel(sampleProjects(), "/cfg/projects")
	m = step(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	view := m.View()

	for _, want := range []string{"Alpha", "Bravo", "/srv/alpha"} {
		if !strings.Contains(view, want) {
			t.Fatalf("browser view missing %q:\n%s", want, view)
		}
	}
}

// TestControlModelEmptyState confirms that with no projects the onboarding card
// renders (with the config path and docs link) and any exit key closes it.
func TestControlModelEmptyState(t *testing.T) {
	m := newControlModel(nil, "/cfg/projects")
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
