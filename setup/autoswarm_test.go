package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withZotHome points ZOT_HOME at a temp dir and writes config.json there
// when content is non-empty. An empty content leaves the file missing.
func withZotHome(t *testing.T, content string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("ZOT_HOME", dir)
	if content != "" {
		if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAutoSwarmOnWhenTrue(t *testing.T) {
	withZotHome(t, `{"auto_swarm_enabled": true}`)
	if !autoSwarmOn() {
		t.Fatal("true should read as on")
	}
}

func TestAutoSwarmOffWhenFalse(t *testing.T) {
	withZotHome(t, `{"auto_swarm_enabled": false}`)
	if autoSwarmOn() {
		t.Fatal("false should read as off")
	}
}

func TestAutoSwarmOffWhenKeyMissing(t *testing.T) {
	withZotHome(t, `{"reasoning": "high"}`)
	if autoSwarmOn() {
		t.Fatal("a missing key should read as off, matching zot's default")
	}
}

func TestAutoSwarmOffWhenFileMissing(t *testing.T) {
	withZotHome(t, "")
	if autoSwarmOn() {
		t.Fatal("a missing config.json should read as off")
	}
}

func TestAutoSwarmOffWhenFileUnparseable(t *testing.T) {
	withZotHome(t, `{not json`)
	if autoSwarmOn() {
		t.Fatal("an unreadable config should read as off, so the warning shows")
	}
}

func TestAutoSwarmOffWhenValueNotBoolean(t *testing.T) {
	withZotHome(t, `{"auto_swarm_enabled": "yes"}`)
	if autoSwarmOn() {
		t.Fatal("a non-boolean value should read as off")
	}
}

func TestProviderStepShowsWarningWhenAutoSwarmOff(t *testing.T) {
	w, _ := newTestWizard(t)
	w.autoSwarm = false
	_, lines, _ := w.Render()
	if !strings.Contains(strings.Join(lines, "\n"), "auto-swarm is off") {
		t.Fatalf("warning missing when auto-swarm is off: %v", lines)
	}
}

func TestProviderStepHasNoWarningWhenAutoSwarmOn(t *testing.T) {
	w, _ := newTestWizard(t)
	w.autoSwarm = true
	_, lines, _ := w.Render()
	if strings.Contains(strings.Join(lines, "\n"), "auto-swarm") {
		t.Fatalf("no warning expected when auto-swarm is on: %v", lines)
	}
}

func TestWarningDoesNotBlockSaving(t *testing.T) {
	w, _ := newTestWizard(t)
	w.autoSwarm = false
	w.cfg = Config{Provider: "amazon-bedrock", Workers: "anthropic.claude-haiku-5-5", Judges: "anthropic.claude-opus-5-5"}
	w.step = stepConfirm
	if !w.HandleKey("enter", "") {
		t.Fatal("the warning must not stop the user from saving the config")
	}
}
