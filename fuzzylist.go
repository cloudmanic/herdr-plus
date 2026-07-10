//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/sahilm/fuzzy"
)

// listItem is one row in a fuzzyList. A selectable row shows a name (matched and
// highlighted) plus an optional dimmer description, and carries ref — the
// caller's identifier for the row (e.g. an index into its own slice). A row with
// selectable=false is a non-selectable separator: with a name it renders as a
// dim group heading, without one as a blank spacer. Either way it is skipped
// during navigation and hidden while filtering.
type listItem struct {
	name       string
	desc       string
	selectable bool
	ref        int
}

// scoredItem is a listItem that survived the current query filter, carrying the
// name-character positions that matched, for highlighting.
type scoredItem struct {
	item    listItem
	matched []int
}

// fuzzyList is a reusable fuzzy-filtered, keyboard-navigable list with a query
// box. The projects browser is a fuzzyList, so the matching, navigation,
// separators, and rendering live here once (and stay reusable for future
// pickers).
type fuzzyList struct {
	input     textinput.Model
	items     []listItem
	filtered  []scoredItem
	cursor    int
	lastQuery string

	// Viewport. maxRows caps how many body lines view() emits, scrolling the
	// window (lineOffset) so the cursor's row stays visible; maxWidth
	// hard-truncates every rendered line so nothing soft-wraps in the terminal
	// and breaks the list's line accounting. Zero means unbounded.
	maxRows    int
	maxWidth   int
	lineOffset int
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

// setViewport tells the list how much room it has: rows is the body-line budget
// below the query/prompt block, width the hard cap on any rendered line's
// display columns. Zero or negative disables the corresponding limit.
func (l *fuzzyList) setViewport(rows, width int) {
	l.maxRows = rows
	l.maxWidth = width
	l.ensureVisible()
}

// lineOf returns the body-line index (0 = the first line after the query/prompt
// block) where filtered[target] renders, using the same accounting as view()
// and rowIndexAt: one line per selectable row, a blank plus an optional heading
// line per separator. A target past the end returns the body's total line
// count.
func (l *fuzzyList) lineOf(target int) int {
	line := 0
	for i, s := range l.filtered {
		if i == target {
			break
		}
		if !s.item.selectable {
			line++ // the separator's leading blank line
			if s.item.name != "" {
				line++ // its heading line
			}
			continue
		}
		line++
	}
	return line
}

// ensureVisible scrolls lineOffset so the cursor's row sits inside the
// viewport, clamping the window to the body's real extent. With no row budget
// the list never scrolls.
func (l *fuzzyList) ensureVisible() {
	if l.maxRows <= 0 {
		l.lineOffset = 0
		return
	}
	if max := l.lineOf(len(l.filtered)) - l.maxRows; l.lineOffset > max {
		l.lineOffset = max
	}
	if l.lineOffset < 0 {
		l.lineOffset = 0
	}
	if len(l.filtered) == 0 {
		return
	}
	cur := l.lineOf(l.cursor)
	if cur < l.lineOffset {
		l.lineOffset = cur
	}
	if cur >= l.lineOffset+l.maxRows {
		l.lineOffset = cur - l.maxRows + 1
	}
}

// filter recomputes the visible rows from the current query. An empty query
// shows every item — separators included — in its natural order. A non-empty
// query fuzzy-matches only the selectable items (separators are dropped while
// searching) against name and description together, while highlighting only the
// matches that land inside the name.
func (l *fuzzyList) filter() {
	q := strings.TrimSpace(l.input.Value())
	if q != l.lastQuery {
		l.cursor = 0
		l.lastQuery = q
	}
	l.filtered = l.filtered[:0]

	if q == "" {
		for _, it := range l.items {
			l.filtered = append(l.filtered, scoredItem{item: it})
		}
		l.clampCursor()
		l.ensureVisible()
		return
	}

	var sel []listItem
	for _, it := range l.items {
		if it.selectable {
			sel = append(sel, it)
		}
	}
	haystacks := make([]string, len(sel))
	nameLens := make([]int, len(sel))
	for i, it := range sel {
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
		l.filtered = append(l.filtered, scoredItem{item: sel[mt.Index], matched: inName})
	}

	l.clampCursor()
	l.ensureVisible()
}

// clampCursor keeps the cursor in range and parked on a selectable row, moving
// to the nearest selectable row (searching down, then up) if it landed on a
// separator.
func (l *fuzzyList) clampCursor() {
	if len(l.filtered) == 0 {
		l.cursor = 0
		return
	}
	if l.cursor >= len(l.filtered) {
		l.cursor = len(l.filtered) - 1
	}
	if l.cursor < 0 {
		l.cursor = 0
	}
	if l.filtered[l.cursor].item.selectable {
		return
	}
	for i := l.cursor; i < len(l.filtered); i++ {
		if l.filtered[i].item.selectable {
			l.cursor = i
			return
		}
	}
	for i := l.cursor; i >= 0; i-- {
		if l.filtered[i].item.selectable {
			l.cursor = i
			return
		}
	}
}

// moveUp and moveDown move the highlight to the previous/next selectable row,
// skipping any separators in between.
func (l *fuzzyList) moveUp() {
	for i := l.cursor - 1; i >= 0; i-- {
		if l.filtered[i].item.selectable {
			l.cursor = i
			l.ensureVisible()
			return
		}
	}
}

func (l *fuzzyList) moveDown() {
	for i := l.cursor + 1; i < len(l.filtered); i++ {
		if l.filtered[i].item.selectable {
			l.cursor = i
			l.ensureVisible()
			return
		}
	}
}

// selectedIndex returns the ref of the highlighted selectable row, or -1 when
// nothing is selectable (empty list, or all matches filtered away).
func (l *fuzzyList) selectedIndex() int {
	if len(l.filtered) == 0 {
		return -1
	}
	it := l.filtered[l.cursor].item
	if !it.selectable {
		return -1
	}
	return it.ref
}

// listPromptLines is how many lines view() renders before the first result row:
// the query/prompt line and the blank spacer beneath it. rowIndexAt and view()
// share it, so the two must always agree about the list's layout.
const listPromptLines = 2

// rowIndexAt maps a view-local screen line (line 0 is the query/prompt line) to
// the index into filtered of the selectable row drawn there, or -1 when the line
// is the prompt, a blank spacer, a separator/heading, or past the end of the
// list. It mirrors view()'s exact line accounting — the prompt line, the blank
// spacer, then one line per selectable row and a blank (plus an optional heading
// line) per separator, offset by the viewport's scroll position — so this and
// view() must change together.
func (l *fuzzyList) rowIndexAt(y int) int {
	if y < listPromptLines {
		return -1 // the query/prompt block, never a row
	}
	body := y - listPromptLines + l.lineOffset
	if l.maxRows > 0 && body >= l.lineOffset+l.maxRows {
		return -1 // below the visible window
	}
	line := 0
	for i, s := range l.filtered {
		if !s.item.selectable {
			line++ // the separator's leading blank line
			if s.item.name != "" {
				line++ // its heading line
			}
			continue
		}
		if line == body {
			return i
		}
		line++
	}
	return -1
}

// clickRow moves the highlight to the selectable row at view-local line y,
// reporting whether y landed on one. It is the mouse counterpart to moveUp /
// moveDown: the caller subtracts its own header height from the click's screen
// row before calling, so y is measured from the top of view()'s own output.
func (l *fuzzyList) clickRow(y int) bool {
	idx := l.rowIndexAt(y)
	if idx < 0 {
		return false
	}
	l.cursor = idx
	return true
}

// editQuery feeds a message to the query box and re-filters. Non-key messages
// (such as the cursor blink tick) pass through harmlessly.
func (l *fuzzyList) editQuery(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	l.input, cmd = l.input.Update(msg)
	l.filter()
	return cmd
}

// view renders the query line, the match count (selectable rows only), and the
// result rows. Separators render as a blank line plus an optional dim heading;
// emptyMsg is shown when no selectable row matches the query. When a viewport
// is set (setViewport) only the maxRows-line window at lineOffset is emitted,
// and every line is truncated to maxWidth so nothing soft-wraps.
func (l fuzzyList) view(emptyMsg string) string {
	var b strings.Builder

	matched, total := 0, 0
	for _, it := range l.items {
		if it.selectable {
			total++
		}
	}
	for _, s := range l.filtered {
		if s.item.selectable {
			matched++
		}
	}

	b.WriteString(promptStyle.Render("❯ "))
	b.WriteString(l.input.View())
	b.WriteString("   ")
	b.WriteString(countStyle.Render(fmt.Sprintf("%d/%d", matched, total)))
	b.WriteString("\n\n")

	// Build the body one screen line at a time so the viewport can slice it.
	// The accounting here must stay in step with lineOf / rowIndexAt.
	var lines []string
	if matched == 0 {
		lines = append(lines, descStyle.Render("  "+emptyMsg))
	}
	for i, s := range l.filtered {
		it := s.item
		if !it.selectable {
			// A blank line separates groups; a heading (if any) labels the group.
			lines = append(lines, "")
			if it.name != "" {
				lines = append(lines, headingStyle.Render(it.name))
			}
			continue
		}
		selected := i == l.cursor
		var row strings.Builder
		if selected {
			row.WriteString(barStyle.Render("▌ "))
		} else {
			row.WriteString("  ")
		}
		row.WriteString(highlightName(it.name, s.matched, selected))
		if it.desc != "" {
			row.WriteString("  ")
			row.WriteString(descStyle.Render(it.desc))
		}
		lines = append(lines, row.String())
	}

	if l.maxRows > 0 && len(lines) > l.maxRows {
		start := l.lineOffset
		if max := len(lines) - l.maxRows; start > max {
			start = max
		}
		if start < 0 {
			start = 0
		}
		lines = lines[start : start+l.maxRows]
	}

	for _, ln := range lines {
		if l.maxWidth > 0 {
			ln = ansi.Truncate(ln, l.maxWidth, "…")
		}
		b.WriteString(ln)
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
