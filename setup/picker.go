package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// pickerWindow is how many model rows the panel shows at once. The host
// gives extensions no terminal height, so the window size is fixed. It is
// small enough to fit a typical terminal alongside the panel chrome.
const pickerWindow = 12

// fuzzyScore reports whether every character of query appears in text, in
// order, ignoring case. A lower score is a tighter match: it is the gap count
// of the tightest window in text that contains query in order, so "hai"
// scores "haiku" (a contiguous run, gap 0) better than "happy-images". ok is
// false when query is not a subsequence of text.
func fuzzyScore(query, text string) (score int, ok bool) {
	q := []rune(strings.ToLower(query))
	t := []rune(strings.ToLower(text))
	if len(q) == 0 {
		return 0, true
	}
	best := -1
	for start := range t {
		if t[start] != q[0] {
			continue
		}
		// Greedily match the rest of the query from this start. Greedy
		// matching from a fixed start gives the tightest end for it.
		qi, end := 1, start
		for ti := start + 1; ti < len(t) && qi < len(q); ti++ {
			if t[ti] == q[qi] {
				qi++
				end = ti
			}
		}
		if qi < len(q) {
			continue
		}
		if span := end - start + 1 - len(q); best < 0 || span < best {
			best = span
		}
	}
	if best < 0 {
		return 0, false
	}
	return best, true
}

// filterModels returns the entries whose provider, id, or name fuzzily
// match query, best matches first. An empty query keeps the input order.
//
// Matching is scored on the model id first, because that is what the user
// types. Provider and display name are a fallback, and a fallback match always
// ranks below any id match.
func filterModels(entries []Entry, query string) []Entry {
	query = strings.TrimSpace(query)
	if query == "" {
		return entries
	}
	type hit struct {
		e     Entry
		score int
	}
	var hits []hit
	for _, e := range entries {
		if s, ok := fuzzyScore(query, e.ID); ok {
			hits = append(hits, hit{e, s})
			continue
		}
		// Fallback: provider or name. Offset by a large constant so these
		// always sort after every id match.
		if s, ok := fuzzyScore(query, e.Provider+" "+e.Name); ok {
			hits = append(hits, hit{e, 1_000_000 + s})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score < hits[j].score })
	out := make([]Entry, len(hits))
	for i, h := range hits {
		out[i] = h.e
	}
	return out
}

// windowAround returns the start and end (exclusive) of the visible slice
// of a list of n rows that keeps cursor visible. The window starts at the top
// of the list and only scrolls once the cursor passes its bottom edge.
func windowAround(n, cursor int) (start, end int) {
	if n <= pickerWindow {
		return 0, n
	}
	start = cursor - pickerWindow + 1
	if start < 0 {
		start = 0
	}
	if start > n-pickerWindow {
		start = n - pickerWindow
	}
	return start, start + pickerWindow
}

// isPrintable reports whether a key event carries a character the filter
// should take. Control characters and empty text are ignored.
func isPrintable(text string) bool {
	if text == "" {
		return false
	}
	for _, r := range text {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

// modelList renders the filter line and the visible window of models for a
// model step. It reads visibleModels so the rows match what pick chooses.
func (w *wizard) modelList() []string {
	models := w.visibleModels()
	var out []string

	switch {
	case w.query == "" && len(models) == 0:
		out = append(out, "  (no models for this provider)")
	case w.query == "":
		out = append(out, fmt.Sprintf("  %d models", len(models)))
	case len(models) == 0:
		out = append(out, fmt.Sprintf("  filter: %s (no matches)", w.query))
	default:
		out = append(out, fmt.Sprintf("  filter: %s (%d matches)", w.query, len(models)))
	}
	out = append(out, "")

	start, end := windowAround(len(models), w.cursor)
	if start > 0 {
		out = append(out, "  ↑ more")
	}
	for i := start; i < end; i++ {
		out = append(out, cursorRow(i == w.cursor, modelRow(models[i])))
	}
	if end < len(models) {
		out = append(out, "  ↓ more")
	}
	return out
}

// onModelStep reports whether the current step shows a filterable model list.
func (w *wizard) onModelStep() bool {
	return w.step == stepWorkers || w.step == stepJudges || w.step == stepReviewers
}

// typeQuery appends one printable key to the filter. Keys that are not
// printable are ignored, so arrows and control keys never change the query.
func (w *wizard) typeQuery(text string) {
	if !isPrintable(text) {
		return
	}
	w.setQuery(w.query + text)
}

// setQuery replaces the filter and moves the cursor back to the top, since
// the list it applies to has changed.
func (w *wizard) setQuery(q string) {
	w.query = q
	w.cursor = 0
}
