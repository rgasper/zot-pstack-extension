package main

import (
	"strconv"
	"strings"
	"testing"
)

// bigCatalog builds n models under one provider, the shape that overflowed
// the terminal in the first version of the picker.
func bigCatalog(n int) []Entry {
	out := make([]Entry, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, Entry{Provider: "amazon-bedrock", ID: "model-" + strconv.Itoa(i), Name: "M"})
	}
	return out
}

func TestRenderedModelRowsStayWithinWindow(t *testing.T) {
	w, _ := newTestWizard(t)
	w.entries = bigCatalog(50)
	w.cfg.Provider = "amazon-bedrock"
	w.step = stepWorkers
	for _, cursor := range []int{0, 25, 49} {
		w.cursor = cursor
		_, lines, _ := w.Render()
		rows := 0
		for _, l := range lines {
			if strings.Contains(l, "model-") {
				rows++
			}
		}
		if rows != pickerWindow {
			t.Fatalf("cursor %d: want %d model rows, got %d", cursor, pickerWindow, rows)
		}
	}
}

func TestTopOfListRendersFirstModelFirst(t *testing.T) {
	w, _ := newTestWizard(t)
	w.entries = bigCatalog(50)
	w.cfg.Provider = "amazon-bedrock"
	w.step = stepWorkers
	_, lines, _ := w.Render()
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "› model-0 ") {
		t.Fatalf("cursor on first model should be visible at top, lines:\n%s", joined)
	}
	if strings.Contains(joined, "model-49") {
		t.Fatal("last model should not render when the list starts at the top")
	}
}

func TestFuzzyMatchesSubsequenceCaseInsensitive(t *testing.T) {
	if _, ok := fuzzyScore("hai", "anthropic.claude-haiku-5-5"); !ok {
		t.Fatal("hai should match haiku")
	}
	if _, ok := fuzzyScore("HAI", "haiku"); !ok {
		t.Fatal("matching should ignore case")
	}
	if _, ok := fuzzyScore("zq", "haiku"); ok {
		t.Fatal("zq should not match haiku")
	}
}

func TestFuzzyPrefersContiguousOverSpread(t *testing.T) {
	tight, _ := fuzzyScore("hai", "haiku")
	spread, _ := fuzzyScore("hai", "happy-images")
	if tight >= spread {
		t.Fatalf("haiku (%d) should score tighter than happy-images (%d)", tight, spread)
	}
}

func TestFilterRanksIDMatchesAboveFallbacks(t *testing.T) {
	entries := []Entry{
		{Provider: "openai", ID: "gpt-x", Name: "hai helper"}, // only name matches
		{Provider: "amazon-bedrock", ID: "anthropic.claude-haiku-5-5", Name: "Haiku"},
	}
	got := filterModels(entries, "hai")
	if len(got) != 2 {
		t.Fatalf("want 2 matches, got %d", len(got))
	}
	if got[0].ID != "anthropic.claude-haiku-5-5" {
		t.Fatalf("id match should rank first, got %s", got[0].ID)
	}
}

func TestFilterEmptyQueryKeepsOrder(t *testing.T) {
	entries := sampleEntries()
	got := filterModels(entries, "")
	if len(got) != len(entries) || got[0].ID != entries[0].ID {
		t.Fatal("empty query should return the input unchanged")
	}
}

func TestWindowKeepsCursorVisibleAndStartsAtTop(t *testing.T) {
	n := 50
	for cursor := 0; cursor < n; cursor++ {
		start, end := windowAround(n, cursor)
		if cursor < start || cursor >= end {
			t.Fatalf("cursor %d outside window [%d,%d)", cursor, start, end)
		}
		if end-start > pickerWindow {
			t.Fatalf("window too tall: %d", end-start)
		}
	}
	if s, _ := windowAround(n, 0); s != 0 {
		t.Fatalf("top of list should start the window at 0, got %d", s)
	}
	if s, e := windowAround(n, n-1); e != n || s != n-pickerWindow {
		t.Fatalf("bottom window wrong: [%d,%d)", s, e)
	}
}

func TestWindowSmallListShowsAll(t *testing.T) {
	if s, e := windowAround(3, 1); s != 0 || e != 3 {
		t.Fatalf("small list should show all rows, got [%d,%d)", s, e)
	}
}

func TestTypingFiltersModelStepAndResetsCursor(t *testing.T) {
	w, _ := newTestWizard(t)
	w.cfg.Provider = "amazon-bedrock"
	w.step = stepWorkers
	w.cursor = 2
	w.HandleKey("rune", "h")
	w.HandleKey("rune", "a")
	w.HandleKey("rune", "i")
	if w.query != "hai" {
		t.Fatalf("query: want hai, got %q", w.query)
	}
	if w.cursor != 0 {
		t.Fatalf("cursor should reset on query change, got %d", w.cursor)
	}
	for _, m := range w.visibleModels() {
		if !strings.Contains(strings.ToLower(m.ID+m.Name), "h") {
			t.Fatalf("filtered row %q does not look like a match", m.ID)
		}
	}
}

func TestPickUsesFilteredRow(t *testing.T) {
	w, _ := newTestWizard(t)
	w.cfg.Provider = "amazon-bedrock"
	w.step = stepWorkers
	w.HandleKey("rune", "o")
	w.HandleKey("rune", "p")
	w.HandleKey("rune", "u")
	w.HandleKey("rune", "s")
	// Only Opus matches "opus" in the test catalogue, so the first visible row is it.
	w.HandleKey("enter", "")
	if w.cfg.Workers != "anthropic.claude-opus-5-5" {
		t.Fatalf("picked %q, want the filtered Opus row", w.cfg.Workers)
	}
}

func TestBackspaceShortensQuery(t *testing.T) {
	w, _ := newTestWizard(t)
	w.cfg.Provider = "amazon-bedrock"
	w.step = stepWorkers
	w.HandleKey("rune", "a")
	w.HandleKey("rune", "b")
	w.HandleKey("backspace", "")
	if w.query != "a" {
		t.Fatalf("after backspace want a, got %q", w.query)
	}
}

func TestLetterSOnWorkersIsFilterNotCommand(t *testing.T) {
	w, _ := newTestWizard(t)
	w.cfg.Provider = "amazon-bedrock"
	w.step = stepWorkers
	w.HandleKey("rune", "s")
	if w.step != stepWorkers {
		t.Fatalf("s on workers should not advance, step=%v", w.step)
	}
	if w.query != "s" {
		t.Fatalf("s should type into the filter, got %q", w.query)
	}
}

func TestReviewerDoneAndClearWorkWithEmptyFilter(t *testing.T) {
	w, _ := newTestWizard(t)
	w.cfg.Provider = "amazon-bedrock"
	w.cfg.Reviewers = []string{"x"}
	w.step = stepReviewers
	w.HandleKey("delete", "")
	if len(w.cfg.Reviewers) != 0 {
		t.Fatalf("delete should clear reviewers")
	}
	w.HandleKey("tab", "")
	if w.step != stepConfirm {
		t.Fatalf("tab should finish reviewers, step=%v", w.step)
	}
}

// The regression: on the reviewers step, typing s or d must enter the
// filter, not finish or clear the list.
func TestReviewerLettersSAndDTypeIntoFilter(t *testing.T) {
	w, _ := newTestWizard(t)
	w.cfg.Provider = "amazon-bedrock"
	w.cfg.Reviewers = []string{"x"}
	w.step = stepReviewers
	w.HandleKey("rune", "s")
	if w.step != stepReviewers {
		t.Fatalf("typing s must not finish reviewers, step=%v", w.step)
	}
	w.HandleKey("rune", "d")
	if len(w.cfg.Reviewers) != 1 {
		t.Fatalf("typing d must not clear reviewers, got %v", w.cfg.Reviewers)
	}
	if w.query != "sd" {
		t.Fatalf("want query sd, got %q", w.query)
	}
}

func TestReviewerLettersTypeWhenFilterActive(t *testing.T) {
	w, _ := newTestWizard(t)
	w.cfg.Provider = "amazon-bedrock"
	w.step = stepReviewers
	w.HandleKey("rune", "o")
	w.HandleKey("rune", "s")
	if w.step != stepReviewers {
		t.Fatalf("s inside a filter should not finish the step")
	}
	if w.query != "os" {
		t.Fatalf("want query os, got %q", w.query)
	}
}

func TestNonPrintableKeysIgnoredByFilter(t *testing.T) {
	w, _ := newTestWizard(t)
	w.cfg.Provider = "amazon-bedrock"
	w.step = stepWorkers
	w.HandleKey("rune", "")
	w.HandleKey("rune", "\x07")
	if w.query != "" {
		t.Fatalf("control input should not enter the filter, got %q", w.query)
	}
}

func TestEscClearsFilterOnBack(t *testing.T) {
	w, _ := newTestWizard(t)
	w.cfg.Provider = "amazon-bedrock"
	w.step = stepWorkers
	w.HandleKey("rune", "h")
	w.HandleKey("esc", "")
	if w.query != "" {
		t.Fatalf("esc should clear the filter, got %q", w.query)
	}
	if w.step != stepProvider {
		t.Fatalf("esc should go back to provider, got %v", w.step)
	}
}
