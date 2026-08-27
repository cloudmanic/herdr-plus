//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestFuzzyListSkipsSeparators confirms the cursor starts on a selectable row
// and that navigation steps over separator rows.
func TestFuzzyListSkipsSeparators(t *testing.T) {
	items := []listItem{
		{name: "alpha", selectable: true, ref: 0},
		{name: "Group", selectable: false}, // heading separator
		{name: "bravo", selectable: true, ref: 2},
		{name: "charlie", selectable: true, ref: 3},
	}
	l := newFuzzyList("", items)

	if got := l.selectedIndex(); got != 0 {
		t.Fatalf("initial selected = %d, want 0", got)
	}
	l.moveDown()
	if got := l.selectedIndex(); got != 2 {
		t.Fatalf("after moveDown selected = %d, want 2 (skip separator)", got)
	}
	l.moveDown()
	if got := l.selectedIndex(); got != 3 {
		t.Fatalf("after 2nd moveDown selected = %d, want 3", got)
	}
	l.moveUp()
	if got := l.selectedIndex(); got != 2 {
		t.Fatalf("after moveUp selected = %d, want 2", got)
	}
}

// TestFuzzyListStartsOnSelectableAfterLeadingSeparator confirms a list that
// begins with a heading parks the cursor on the first real option.
func TestFuzzyListStartsOnSelectableAfterLeadingSeparator(t *testing.T) {
	items := []listItem{
		{name: "Heading", selectable: false},
		{name: "first", selectable: true, ref: 1},
	}
	l := newFuzzyList("", items)
	if got := l.selectedIndex(); got != 1 {
		t.Fatalf("selected = %d, want 1 (skip leading heading)", got)
	}
}

// TestFuzzyListHidesSeparatorsWhileFiltering confirms separators drop out of the
// results once a query is typed, leaving only matching selectable rows.
func TestFuzzyListHidesSeparatorsWhileFiltering(t *testing.T) {
	items := []listItem{
		{name: "alpha", selectable: true, ref: 0},
		{name: "Group", selectable: false},
		{name: "bravo", selectable: true, ref: 2},
	}
	l := newFuzzyList("", items)
	l.input.SetValue("alpha")
	l.filter()

	for _, s := range l.filtered {
		if !s.item.selectable {
			t.Fatalf("a separator leaked into filtered results while searching")
		}
	}
	if got := l.selectedIndex(); got != 0 {
		t.Fatalf("selected = %d, want 0", got)
	}
}

// TestFuzzyListViewCountExcludesSeparators confirms the match count reflects
// only selectable rows, not separators.
func TestFuzzyListViewCountExcludesSeparators(t *testing.T) {
	items := []listItem{
		{name: "alpha", selectable: true, ref: 0},
		{name: "Group", selectable: false},
		{name: "bravo", selectable: true, ref: 2},
	}
	l := newFuzzyList("", items)
	if got := l.view("none"); !strings.Contains(got, "2/2") {
		t.Fatalf("view count should be 2/2 (separators excluded); got:\n%s", got)
	}
}

// TestFuzzyListRowIndexAtMatchesView confirms rowIndexAt's line accounting agrees
// with what view() actually renders — including the blank line and heading a
// separator consumes — by finding each row's real screen line in the rendered
// output and asking rowIndexAt to map it back to the right row. This guards the
// "rowIndexAt and view() must change together" invariant.
func TestFuzzyListRowIndexAtMatchesView(t *testing.T) {
	items := []listItem{
		{name: "alpha", selectable: true, ref: 0},
		{name: "Group", selectable: false}, // blank + heading between the rows
		{name: "bravo", selectable: true, ref: 2},
		{name: "charlie", selectable: true, ref: 3},
	}
	l := newFuzzyList("", items)
	lines := strings.Split(l.view("none"), "\n")

	for _, tc := range []struct {
		name string
		ref  int
	}{{"alpha", 0}, {"bravo", 2}, {"charlie", 3}} {
		y := -1
		for i, line := range lines {
			if strings.Contains(line, tc.name) {
				y = i
				break
			}
		}
		if y < 0 {
			t.Fatalf("row %q not found in rendered view:\n%s", tc.name, l.view("none"))
		}
		idx := l.rowIndexAt(y)
		if idx < 0 {
			t.Fatalf("rowIndexAt(%d) for %q = -1, want a selectable row", y, tc.name)
		}
		if got := l.filtered[idx].item.ref; got != tc.ref {
			t.Fatalf("rowIndexAt(%d) -> ref %d, want %d (%q)", y, got, tc.ref, tc.name)
		}
	}

	// The blank spacer under the prompt and a line past the end are not rows.
	if got := l.rowIndexAt(1); got != -1 {
		t.Fatalf("rowIndexAt(1) = %d, want -1 (blank spacer under the prompt)", got)
	}
	if got := l.rowIndexAt(len(lines) + 5); got != -1 {
		t.Fatalf("rowIndexAt past the end = %d, want -1", got)
	}
}

// TestFuzzyListClickRow confirms clicking a row's line moves the highlight there
// and reports success, while clicking a non-row line is a no-op that leaves the
// cursor put.
func TestFuzzyListClickRow(t *testing.T) {
	items := []listItem{
		{name: "alpha", selectable: true, ref: 0}, // view line 2
		{name: "bravo", selectable: true, ref: 1}, // view line 3
	}
	l := newFuzzyList("", items)

	if !l.clickRow(3) {
		t.Fatal("clickRow(3) should land on bravo")
	}
	if got := l.selectedIndex(); got != 1 {
		t.Fatalf("after clickRow(3) selected ref = %d, want 1 (bravo)", got)
	}

	if l.clickRow(1) { // the blank spacer line
		t.Fatal("clickRow on a blank line should report no hit")
	}
	if got := l.selectedIndex(); got != 1 {
		t.Fatalf("a missed click moved the cursor: ref = %d, want 1", got)
	}
}

// TestFuzzyListQueryChangeResetsCursor verifies that when the query changes,
// the cursor resets to 0 (the top matched item), preventing the issue where
// starting with a separator (which defaults the initial cursor to index 1)
// causes the second matching item (index 1 of filtered) to be highlighted
// instead of the first matched item (index 0 of filtered).
func TestFuzzyListQueryChangeResetsCursor(t *testing.T) {
	items := []listItem{
		{name: "Group Heading", selectable: false},
		{name: "alpha", selectable: true, ref: 1},
		{name: "bravo", selectable: true, ref: 2},
	}
	l := newFuzzyList("", items)

	// Since index 0 is a heading, the initial cursor must skip it and land on index 1 ("alpha").
	if got := l.cursor; got != 1 {
		t.Fatalf("initial cursor = %d, want 1", got)
	}
	if got := l.selectedIndex(); got != 1 {
		t.Fatalf("initial selectedIndex = %d, want 1 (alpha)", got)
	}

	// Change query to "a" so "alpha" and "bravo" both match.
	l.input.SetValue("a")
	l.filter()

	// Since the query changed, the cursor should have been reset to 0, which points to "alpha".
	if got := l.cursor; got != 0 {
		t.Fatalf("cursor after typing 'a' = %d, want 0", got)
	}
	if got := l.selectedIndex(); got != 1 {
		t.Fatalf("selectedIndex after typing 'a' = %d, want 1 (alpha)", got)
	}

	// Clear query back to ""
	l.input.SetValue("")
	l.filter()

	// Cursor should go back to 1 (skipping heading).
	if got := l.cursor; got != 1 {
		t.Fatalf("cursor after clearing query = %d, want 1", got)
	}
	if got := l.selectedIndex(); got != 1 {
		t.Fatalf("selectedIndex after clearing query = %d, want 1 (alpha)", got)
	}
}

// TestFuzzyListViewportScrollsWithCursor confirms that with a row budget set,
// view() emits at most that many body lines, the window follows the cursor in
// both directions, and rowIndexAt maps clicks through the scrolled window.
func TestFuzzyListViewportScrollsWithCursor(t *testing.T) {
	var items []listItem
	names := []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot"}
	for i, n := range names {
		items = append(items, listItem{name: n, selectable: true, ref: i})
	}
	l := newFuzzyList("", items)
	l.setViewport(3, 80)

	bodyLines := func() []string {
		out := strings.Split(strings.TrimRight(l.view("none"), "\n"), "\n")
		return out[listPromptLines:] // drop the query line and its blank spacer
	}

	if got := len(bodyLines()); got != 3 {
		t.Fatalf("visible body lines = %d, want 3", got)
	}

	// Walk to the last row: the window must scroll so the cursor stays visible.
	for i := 0; i < len(names)-1; i++ {
		l.moveDown()
	}
	if got := l.selectedIndex(); got != 5 {
		t.Fatalf("after walking down selected = %d, want 5", got)
	}
	lines := bodyLines()
	if got := len(lines); got != 3 {
		t.Fatalf("visible body lines after scroll = %d, want 3", got)
	}
	if !strings.Contains(lines[len(lines)-1], "foxtrot") {
		t.Fatalf("cursor row not visible after scrolling down: %q", lines)
	}

	// A click on the first visible body line must resolve through the scroll
	// offset to the row actually drawn there.
	if got := l.rowIndexAt(listPromptLines); got != 3 {
		t.Fatalf("rowIndexAt(top of window) = %d, want 3 (delta)", got)
	}

	// Walking back up scrolls the window back to the top.
	for i := 0; i < len(names)-1; i++ {
		l.moveUp()
	}
	lines = bodyLines()
	if !strings.Contains(lines[0], "alpha") {
		t.Fatalf("cursor row not visible after scrolling back up: %q", lines)
	}
}

// TestFuzzyListViewportTruncatesRows confirms the width cap keeps every emitted
// line within the viewport so nothing soft-wraps in the terminal.
func TestFuzzyListViewportTruncatesRows(t *testing.T) {
	long := strings.Repeat("x", 60)
	l := newFuzzyList("", []listItem{
		{name: long, desc: strings.Repeat("y", 60), selectable: true, ref: 0},
	})
	l.setViewport(5, 20)

	for _, line := range strings.Split(strings.TrimRight(l.view("none"), "\n"), "\n")[listPromptLines:] {
		if w := lipgloss.Width(line); w > 20 {
			t.Fatalf("body line is %d columns wide, want <= 20: %q", w, line)
		}
	}
}
