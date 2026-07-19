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
// shows every item — separators included — in its natural order. A non-empty
// query fuzzy-matches only the selectable items (separators are dropped while
// searching) against name and description together, while highlighting only the
// matches that land inside the name.
func (l *fuzzyList) filter() {
	q := strings.TrimSpace(l.input.Value())
	l.filtered = l.filtered[:0]

	if q == "" {
		for _, it := range l.items {
			l.filtered = append(l.filtered, scoredItem{item: it})
		}
		l.clampCursor()
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
			return
		}
	}
}

func (l *fuzzyList) moveDown() {
	for i := l.cursor + 1; i < len(l.filtered); i++ {
		if l.filtered[i].item.selectable {
			l.cursor = i
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

// listPromptLines is how many lines the list renders before the first result
// row: the query/prompt line and the blank spacer beneath it. The renderer and
// the hit-testing both read buildLines, so their line accounting can never drift.
const listPromptLines = 2

// rowIndexAt maps a view-local screen line (line 0 is the query/prompt line) to
// the index into filtered of the selectable row drawn there, or -1 when the line
// is the prompt, a blank spacer, a separator/heading, a scroll hint, or past the
// end. This is the unbounded (whole-list) form, for callers that don't clamp
// height.
func (l *fuzzyList) rowIndexAt(y int) int {
	return l.rowIndexAtLimited(y, 0)
}

// rowIndexAtLimited is rowIndexAt for a height-limited list: maxLines caps the
// total lines the list may render (0 means unbounded). It shares buildLines with
// the renderer, so a click always maps to exactly the row shown at that line —
// scroll window and hint lines included. The caller must pass the same maxLines
// it rendered with.
func (l *fuzzyList) rowIndexAtLimited(y, maxLines int) int {
	if y < 0 {
		return -1
	}
	lines := l.buildLines("", maxLines)
	if y >= len(lines) {
		return -1
	}
	return lines[y].idx
}

// clickRow moves the highlight to the selectable row at view-local line y,
// reporting whether y landed on one. It is the mouse counterpart to moveUp /
// moveDown: the caller subtracts its own header height from the click's screen
// row before calling, so y is measured from the top of the list's own output.
// This unbounded form assumes the whole list is rendered.
func (l *fuzzyList) clickRow(y int) bool {
	return l.clickRowLimited(y, 0)
}

// clickRowLimited is clickRow for a height-limited list, where maxLines caps the
// rendered lines (0 means unbounded). The caller must pass the same maxLines it
// rendered with so the click lands on the row actually shown at line y.
func (l *fuzzyList) clickRowLimited(y, maxLines int) bool {
	idx := l.rowIndexAtLimited(y, maxLines)
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

// view renders the whole list — query line, match count, and every result row —
// with no height limit. Kept for callers and tests that don't clamp height.
func (l fuzzyList) view(emptyMsg string) string {
	return l.viewLimited(emptyMsg, 0)
}

// viewLimited renders the list clamped to maxLines total screen lines (0 means
// unbounded). When the results don't fit it shows a scrolling window around the
// cursor, bracketed by "↑ N more" / "↓ N more" hints, so the list never spills
// past its budget and clips the layout around it. The cursor row is always shown.
func (l fuzzyList) viewLimited(emptyMsg string, maxLines int) string {
	lines := l.buildLines(emptyMsg, maxLines)
	parts := make([]string, len(lines))
	for i, ln := range lines {
		parts[i] = ln.text
	}
	return strings.Join(parts, "\n") + "\n"
}

// listLine is one rendered screen line of the list. idx is the index into
// filtered of the selectable row shown on that line, or -1 for the prompt, a
// blank, a heading, a scroll hint, or the empty-state message. The renderer and
// the click hit-testing both read buildLines, so they share one source of truth
// for what sits on every line.
type listLine struct {
	text string
	idx  int
}

// counts returns how many selectable rows match the current query and how many
// exist in total (separators excluded from both).
func (l fuzzyList) counts() (matched, total int) {
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
	return matched, total
}

// entryLines is how many screen lines filtered[i] occupies: one for a selectable
// row, two for a heading separator (its leading blank plus the heading line), one
// for a blank spacer separator.
func (l fuzzyList) entryLines(i int) int {
	it := l.filtered[i].item
	if it.selectable {
		return 1
	}
	if it.name != "" {
		return 2
	}
	return 1
}

// window picks the contiguous slice [start,end) of filtered to render so the
// cursor stays visible and the slice — plus up to two scroll-hint lines — fits in
// maxLines. It also returns the hidden selectable-row counts above and below, for
// the hints. maxLines <= 0 (or a list that already fits) renders everything with
// no hints. Below the space for even one hint the hints are dropped rather than
// allowed to overflow the budget.
func (l fuzzyList) window(maxLines int) (start, end, topMore, botMore int) {
	n := len(l.filtered)
	if n == 0 {
		return 0, 0, 0, 0
	}
	if maxLines <= 0 {
		return 0, n, 0, 0
	}

	avail := maxLines - listPromptLines
	if avail < 1 {
		avail = 1
	}

	total := 0
	for i := 0; i < n; i++ {
		total += l.entryLines(i)
	}
	if total <= avail {
		return 0, n, 0, 0
	}

	reserve := 0
	if avail > 2 {
		reserve = 2
	}
	budget := avail - reserve
	if budget < 1 {
		budget = 1
	}

	// Grow a window outward from the cursor, filling below then above, so the
	// highlighted row is always drawn and roughly centered.
	start, end = l.cursor, l.cursor+1
	used := l.entryLines(l.cursor)
	i, j := l.cursor+1, l.cursor-1
	for {
		grew := false
		if i < n && used+l.entryLines(i) <= budget {
			used += l.entryLines(i)
			end = i + 1
			i++
			grew = true
		}
		if j >= 0 && used+l.entryLines(j) <= budget {
			used += l.entryLines(j)
			start = j
			j--
			grew = true
		}
		if !grew {
			break
		}
	}

	if reserve > 0 {
		for k := 0; k < start; k++ {
			if l.filtered[k].item.selectable {
				topMore++
			}
		}
		for k := end; k < n; k++ {
			if l.filtered[k].item.selectable {
				botMore++
			}
		}
	}
	return start, end, topMore, botMore
}

// buildLines renders the list into ordered screen lines: the query/count prompt,
// a blank, then the visible window of result rows (a scrolling slice with
// "↑/↓ N more" hints when the results outgrow maxLines). emptyMsg is shown when
// nothing matches. It is the single layout both the renderer and the click
// hit-testing read, so the two can never disagree about what sits on a line.
func (l fuzzyList) buildLines(emptyMsg string, maxLines int) []listLine {
	matched, total := l.counts()

	prompt := promptStyle.Render("❯ ") + l.input.View() + "   " +
		countStyle.Render(fmt.Sprintf("%d/%d", matched, total))
	out := []listLine{{text: prompt, idx: -1}, {text: "", idx: -1}}

	if matched == 0 {
		out = append(out, listLine{text: descStyle.Render("  " + emptyMsg), idx: -1})
		return out
	}

	start, end, topMore, botMore := l.window(maxLines)

	if topMore > 0 {
		out = append(out, listLine{text: scrollHintStyle.Render(fmt.Sprintf("  ↑ %d more", topMore)), idx: -1})
	}
	for i := start; i < end; i++ {
		s := l.filtered[i]
		it := s.item
		if !it.selectable {
			// A blank line separates groups; a heading (if any) labels the group.
			out = append(out, listLine{text: "", idx: -1})
			if it.name != "" {
				out = append(out, listLine{text: headingStyle.Render(it.name), idx: -1})
			}
			continue
		}
		selected := i == l.cursor
		var b strings.Builder
		if selected {
			b.WriteString(barStyle.Render("▌ "))
		} else {
			b.WriteString("  ")
		}
		b.WriteString(highlightName(it.name, s.matched, selected))
		if it.desc != "" {
			b.WriteString("  ")
			b.WriteString(descStyle.Render(it.desc))
		}
		out = append(out, listLine{text: b.String(), idx: i})
	}
	if botMore > 0 {
		out = append(out, listLine{text: scrollHintStyle.Render(fmt.Sprintf("  ↓ %d more", botMore)), idx: -1})
	}
	return out
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
