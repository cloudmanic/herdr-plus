//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"fmt"
	"strings"
	"testing"
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

// manyItems returns n selectable rows, useful for exercising the scroll window.
func manyItems(n int) []listItem {
	items := make([]listItem, n)
	for i := 0; i < n; i++ {
		items[i] = listItem{name: fmt.Sprintf("row-%02d", i), selectable: true, ref: i}
	}
	return items
}

// TestFuzzyListWindowFitsBudget confirms a list taller than its budget is clamped
// to at most maxLines rendered lines, keeps the cursor row visible, and brackets
// the clipped list with scroll hints.
func TestFuzzyListWindowFitsBudget(t *testing.T) {
	l := newFuzzyList("", manyItems(40))

	const maxLines = 12
	// Move the cursor into the middle so both hints should appear.
	for i := 0; i < 20; i++ {
		l.moveDown()
	}

	lines := l.buildLines("none", maxLines)
	if len(lines) > maxLines {
		t.Fatalf("rendered %d lines, want <= %d", len(lines), maxLines)
	}

	// The highlighted row must be somewhere in the rendered window.
	found := false
	for _, ln := range lines {
		if ln.idx == l.cursor {
			found = true
		}
	}
	if !found {
		t.Fatalf("cursor row (filtered idx %d) not in the rendered window", l.cursor)
	}

	joined := l.viewLimited("none", maxLines)
	if !strings.Contains(joined, "↑") || !strings.Contains(joined, "↓") {
		t.Fatalf("expected both scroll hints on a mid-list window:\n%s", joined)
	}
}

// TestFuzzyListWindowUnboundedRendersAll confirms maxLines == 0 keeps the old
// whole-list behaviour: every row present, no scroll hints.
func TestFuzzyListWindowUnboundedRendersAll(t *testing.T) {
	l := newFuzzyList("", manyItems(40))
	got := l.view("none")
	if strings.Contains(got, "↑") || strings.Contains(got, "↓") {
		t.Fatalf("unbounded view should have no scroll hints:\n%s", got)
	}
	if !strings.Contains(got, "row-39") {
		t.Fatal("unbounded view should render the last row")
	}
}

// TestFuzzyListClickRowLimitedMapsWindow confirms a click on a scrolled list maps
// to the row actually shown at that screen line, not the unscrolled position.
func TestFuzzyListClickRowLimitedMapsWindow(t *testing.T) {
	l := newFuzzyList("", manyItems(40))
	const maxLines = 12
	for i := 0; i < 20; i++ {
		l.moveDown()
	}

	lines := l.buildLines("none", maxLines)
	// Find a rendered screen line that carries a selectable row and click it.
	for y, ln := range lines {
		if ln.idx < 0 {
			continue
		}
		want := ln.idx
		if !l.clickRowLimited(y, maxLines) {
			t.Fatalf("clickRowLimited(%d) reported no hit on a row line", y)
		}
		if l.cursor != want {
			t.Fatalf("clickRowLimited(%d) set cursor %d, want %d", y, l.cursor, want)
		}
		return
	}
	t.Fatal("no selectable row line found in the windowed render")
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
