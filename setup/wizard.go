package main

import (
	"fmt"
	"sync"
)

// Wizard steps. The flow is: provider, workers, judges, reviewers, then a
// confirm screen that writes the file.
type step int

const (
	stepProvider step = iota
	stepWorkers
	stepJudges
	stepReviewers
	stepConfirm
)

const panelID = "pstack-setup"

// wizard holds the state for one /pstack-setup session.
type wizard struct {
	mu      sync.Mutex
	host    *host
	path    string
	entries []Entry
	cfg     Config

	step   step
	cursor int
	// saveErr holds the last save failure, shown on the confirm step.
	saveErr string
	// query is the model filter typed on a model step. It resets when the
	// step changes.
	query string
	// autoSwarm is zot's auto-swarm setting when the wizard opened. When it is
	// off, the provider step shows a warning, since pstack's parallel steps
	// need swarm_spawn.
	autoSwarm bool
}

// visibleModels is the model list the current step shows and picks from.
// Render, rowCount, and pick all read it, so the highlighted row and the
// chosen model always match.
func (w *wizard) visibleModels() []Entry {
	return filterModels(ModelsFor(w.entries, w.cfg.Provider), w.query)
}

// Render builds the panel content for the current step. It has no side
// effects, so it can be tested without a host.
func (w *wizard) Render() (title string, lines []string, footer string) {
	switch w.step {
	case stepProvider:
		title = "pstack setup: provider"
		lines = append(lines, "Pick the provider pstack models come from.", "")
		if !w.autoSwarm {
			lines = append(lines, "⚠ auto-swarm is off. pstack's parallel steps need it: turn it on in /settings.", "")
		}
		for i, p := range Providers(w.entries) {
			lines = append(lines, cursorRow(i == w.cursor, p))
		}
		footer = "↑/↓ move · enter pick · esc cancel"

	case stepWorkers, stepJudges:
		group := groupWorkers
		if w.step == stepJudges {
			group = groupJudges
		}
		title = "pstack setup: " + group
		lines = append(lines, fmt.Sprintf("Model for \x1b[1m%s\x1b[22m roles.", group), "")
		lines = append(lines, w.modelList()...)
		footer = "type to filter · ↑/↓ move · enter pick · esc back"

	case stepReviewers:
		title = "pstack setup: reviewers"
		lines = append(lines, "Each enter adds one reviewer. A longer list means more reviewers per review.", "")
		if len(w.cfg.Reviewers) == 0 {
			lines = append(lines, "  (none yet: inherits parent model)")
		}
		for _, r := range w.cfg.Reviewers {
			lines = append(lines, "  ✓ "+r)
		}
		lines = append(lines, "")
		lines = append(lines, w.modelList()...)
		footer = "type to filter · ↑/↓ move · enter add · tab done · delete clear · esc back"

	case stepConfirm:
		title = "pstack setup: confirm"
		lines = append(lines, "Will be written to:", "  "+w.path, "")
		lines = append(lines, Describe(w.cfg)...)
		if w.saveErr != "" {
			lines = append(lines, "", "save failed: "+w.saveErr)
		}
		footer = "enter save · esc back"
	}
	return title, lines, footer
}

// rowCount is how many selectable rows the current step shows.
func (w *wizard) rowCount() int {
	switch w.step {
	case stepProvider:
		return len(Providers(w.entries))
	case stepWorkers, stepJudges, stepReviewers:
		return len(w.visibleModels())
	}
	return 0
}

// HandleKey applies one key press. It returns true when the panel should
// close: Esc on the first step, Enter on confirm after saving, or a save error.
func (w *wizard) HandleKey(key, text string) (closePanel bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	switch key {
	case "up":
		if w.cursor > 0 {
			w.cursor--
		}
	case "down":
		if w.cursor < w.rowCount()-1 {
			w.cursor++
		}
	case "esc":
		if w.step == stepProvider {
			return true
		}
		w.back()
	case "enter":
		if w.step == stepConfirm {
			return w.save()
		}
		w.pick()
	case "rune":
		// On a model step every printable key goes into the filter, so names
		// like "sonnet" or "deepseek" work. Reviewer commands use keys the
		// filter cannot receive: Tab finishes, Delete clears.
		if w.onModelStep() {
			w.typeQuery(text)
		}
	case "tab":
		if w.step == stepReviewers {
			w.advance()
		}
	case "delete":
		if w.step == stepReviewers {
			w.cfg.Reviewers = nil
		}
	case "backspace":
		if w.onModelStep() && w.query != "" {
			r := []rune(w.query)
			w.setQuery(string(r[:len(r)-1]))
		}
	}
	w.redraw()
	return false
}

func (w *wizard) redraw() {
	title, lines, footer := w.Render()
	_ = w.host.send(panelRenderOut{Type: "panel_render", PanelID: panelID, Title: title, Lines: lines, Footer: footer})
}

// save writes the config and reports whether the panel should close. On
// failure it stays open on the confirm step so the user can retry or esc.
func (w *wizard) save() bool {
	if err := SaveConfig(w.path, w.cfg); err != nil {
		w.saveErr = err.Error()
		return false
	}
	w.saveErr = ""
	return true
}

func (w *wizard) back() {
	if w.step > stepProvider {
		w.step--
		w.cursor = 0
		w.query = ""
	}
}

func (w *wizard) advance() {
	if w.step < stepConfirm {
		w.step++
		w.cursor = 0
		w.query = ""
	}
}

func (w *wizard) pick() {
	models := w.visibleModels()
	switch w.step {
	case stepProvider:
		provs := Providers(w.entries)
		if w.cursor < len(provs) {
			// Changing provider invalidates models picked under the old one.
			if w.cfg.Provider != provs[w.cursor] {
				w.cfg.Workers, w.cfg.Judges, w.cfg.Reviewers = "", "", nil
			}
			w.cfg.Provider = provs[w.cursor]
			w.advance()
		}
	case stepWorkers:
		if w.cursor < len(models) {
			w.cfg.Workers = models[w.cursor].ID
			w.advance()
		}
	case stepJudges:
		if w.cursor < len(models) {
			w.cfg.Judges = models[w.cursor].ID
			w.advance()
		}
	case stepReviewers:
		if w.cursor < len(models) {
			w.cfg.Reviewers = append(w.cfg.Reviewers, models[w.cursor].ID)
		}
	}
}
