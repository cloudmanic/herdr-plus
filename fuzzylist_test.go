//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
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
