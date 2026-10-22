package main

import "fmt"

// cursorRow marks the selected row. Plain text, so it renders the same
// in any terminal.
func cursorRow(selected bool, text string) string {
	if selected {
		return "› " + text
	}
	return "  " + text
}

// modelRow formats one catalogue entry for the panel.
func modelRow(m Entry) string {
	r := " "
	if m.Reasoning {
		r = "✓"
	}
	return fmt.Sprintf("%s  ctx %d  reasoning %s  %s", m.ID, m.ContextWindow, r, m.Name)
}
