//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
)

// listItem is one row in a fuzzyList: a name (matched and highlighted) plus an
// optional dimmer description.
type listItem struct {
	name string
	desc string
}

// scoredItem is a listItem that survived the current query filter, carrying its
// original index and the name-character positions that matched, for
// highlighting.
type scoredItem struct {
	item    listItem
	index   int
	matched []int
}

// fuzzyList is a reusable fuzzy-filtered, keyboard-navigable list with a query
// box. Both the action picker and the "select" option screen are fuzzyLists, so
// the matching, navigation, and rendering live here once.
type fuzzyList struct {
	input    textinput.Model
	items    []listItem
	filtered []scoredItem
	cursor   int
}

// newFuzzyList builds a list over items with a focused, empty query box.
func newFuzzyList(placeholder string, items []listItem) fuzzyList {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Prompt = ""
	ti.Focus()

	l := fuzzyList{input: ti, items: items}
	l.filter()
	return l
}

// filter recomputes the visible rows from the current query. An empty query
// shows every item in its natural order. Matching runs against name and
// description together so a description fragment also narrows the list, while
// only matches that land inside the name are highlighted.
func (l *fuzzyList) filter() {
	q := strings.TrimSpace(l.input.Value())
	l.filtered = l.filtered[:0]

	if q == "" {
		for i, it := range l.items {
			l.filtered = append(l.filtered, scoredItem{item: it, index: i})
		}
	} else {
		haystacks := make([]string, len(l.items))
		nameLens := make([]int, len(l.items))
		for i, it := range l.items {
			haystacks[i] = it.name + "  " + it.desc
			nameLens[i] = len(it.name)
		}
		for _, mt := range fuzzy.Find(q, haystacks) {
			var inName []int
			for _, idx := range mt.MatchedIndexes {
				if idx < nameLens[mt.Index] {
					inName = append(inName, idx)
				}
			}
			l.filtered = append(l.filtered, scoredItem{item: l.items[mt.Index], index: mt.Index, matched: inName})
		}
	}

	if l.cursor >= len(l.filtered) {
		l.cursor = max(0, len(l.filtered)-1)
	}
}

// moveUp and moveDown nudge the highlight, clamped to the visible rows.
func (l *fuzzyList) moveUp() {
	if l.cursor > 0 {
		l.cursor--
	}
}

func (l *fuzzyList) moveDown() {
	if l.cursor < len(l.filtered)-1 {
		l.cursor++
	}
}

// selectedIndex returns the original items index of the highlighted row, or -1
// when nothing matches the current query.
func (l *fuzzyList) selectedIndex() int {
	if len(l.filtered) == 0 {
		return -1
	}
	return l.filtered[l.cursor].index
}

// editQuery feeds a message to the query box and re-filters. Non-key messages
// (such as the cursor blink tick) pass through harmlessly.
func (l *fuzzyList) editQuery(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	l.input, cmd = l.input.Update(msg)
	l.filter()
	return cmd
}

// view renders the query line, the match count, and the result rows. emptyMsg
// is shown when no row matches the query.
func (l fuzzyList) view(emptyMsg string) string {
	var b strings.Builder

	b.WriteString(promptStyle.Render("❯ "))
	b.WriteString(l.input.View())
	b.WriteString("   ")
	b.WriteString(countStyle.Render(fmt.Sprintf("%d/%d", len(l.filtered), len(l.items))))
	b.WriteString("\n\n")

	if len(l.filtered) == 0 {
		b.WriteString(descStyle.Render("  " + emptyMsg))
		b.WriteString("\n")
	}
	for i, s := range l.filtered {
		selected := i == l.cursor
		if selected {
			b.WriteString(barStyle.Render("▌ "))
		} else {
			b.WriteString("  ")
		}
		b.WriteString(highlightName(s.item.name, s.matched, selected))
		if s.item.desc != "" {
			b.WriteString("  ")
			b.WriteString(descStyle.Render(s.item.desc))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// highlightName renders a row's name with the fuzzy-matched characters
// emphasized. matched holds byte indexes into the name string (names are
// effectively ASCII for matching, so byte and rune indexes coincide).
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
