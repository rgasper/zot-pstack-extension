package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleEntries() []Entry {
	return []Entry{
		{Provider: "amazon-bedrock", ID: "anthropic.claude-haiku-5-5", Name: "Haiku 5.5", Reasoning: true, ContextWindow: 200000},
		{Provider: "amazon-bedrock", ID: "anthropic.claude-opus-5-5", Name: "Opus 5.5", Reasoning: true, ContextWindow: 200000},
		{Provider: "amazon-bedrock", ID: "global.xai.grok-4.7", Name: "Grok 4.7", Reasoning: true, ContextWindow: 256000},
		{Provider: "openai", ID: "gpt-x", Name: "GPT X", ContextWindow: 400000},
	}
}

// newTestWizard builds a wizard whose host writes into a buffer, so the
// tests can check what the panel would have shown.
func newTestWizard(t *testing.T) (*wizard, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	return &wizard{
		host:    &host{out: &buf},
		path:    filepath.Join(t.TempDir(), "pstack", "models.json"),
		entries: sampleEntries(),
		step:    stepProvider,
	}, &buf
}

func TestProvidersAreDistinctAndOrdered(t *testing.T) {
	got := Providers(sampleEntries())
	want := []string{"amazon-bedrock", "openai"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestPickingProviderAdvancesToWorkers(t *testing.T) {
	w, _ := newTestWizard(t)
	w.cursor = 0
	w.HandleKey("enter", "")
	if w.cfg.Provider != "amazon-bedrock" {
		t.Fatalf("provider: got %q", w.cfg.Provider)
	}
	if w.step != stepWorkers {
		t.Fatalf("step: want workers, got %v", w.step)
	}
}

func TestGroupsOnlyOfferModelsFromChosenProvider(t *testing.T) {
	w, _ := newTestWizard(t)
	w.cfg.Provider = "openai"
	w.step = stepWorkers
	if n := w.rowCount(); n != 1 {
		t.Fatalf("openai worker rows: want 1, got %d", n)
	}
	w.cfg.Provider = "amazon-bedrock"
	if n := w.rowCount(); n != 3 {
		t.Fatalf("bedrock worker rows: want 3, got %d", n)
	}
}

func TestChangingProviderClearsModelsChosenUnderOldOne(t *testing.T) {
	w, _ := newTestWizard(t)
	w.cfg = Config{Provider: "amazon-bedrock", Workers: "anthropic.claude-haiku-5-5", Judges: "anthropic.claude-opus-5-5", Reviewers: []string{"global.xai.grok-4.7"}}
	w.step = stepProvider
	w.cursor = 1 // openai
	w.HandleKey("enter", "")
	if w.cfg.Provider != "openai" {
		t.Fatalf("provider: got %q", w.cfg.Provider)
	}
	if w.cfg.Workers != "" || w.cfg.Judges != "" || len(w.cfg.Reviewers) != 0 {
		t.Fatalf("stale models kept: %+v", w.cfg)
	}
}

func TestReviewersAddsOnePerEnter(t *testing.T) {
	w, _ := newTestWizard(t)
	w.cfg.Provider = "amazon-bedrock"
	w.step = stepReviewers
	w.cursor = 1
	w.HandleKey("enter", "")
	w.HandleKey("enter", "")
	if len(w.cfg.Reviewers) != 2 {
		t.Fatalf("want 2 reviewers, got %v", w.cfg.Reviewers)
	}
	w.HandleKey("delete", "")
	if len(w.cfg.Reviewers) != 0 {
		t.Fatalf("delete should clear, got %v", w.cfg.Reviewers)
	}
}

func TestEscOnFirstStepCancels(t *testing.T) {
	w, _ := newTestWizard(t)
	if !w.HandleKey("esc", "") {
		t.Fatal("esc on provider step should close the panel")
	}
}

func TestEscElsewhereGoesBack(t *testing.T) {
	w, _ := newTestWizard(t)
	w.step = stepJudges
	if w.HandleKey("esc", "") {
		t.Fatal("esc on judges should not close the panel")
	}
	if w.step != stepWorkers {
		t.Fatalf("want workers after esc, got %v", w.step)
	}
}

func TestSaveWritesConfigAndCloses(t *testing.T) {
	w, _ := newTestWizard(t)
	w.cfg = Config{Provider: "amazon-bedrock", Workers: "anthropic.claude-haiku-5-5", Judges: "anthropic.claude-opus-5-5", Reviewers: []string{"global.xai.grok-4.7"}}
	w.step = stepConfirm
	if !w.HandleKey("enter", "") {
		t.Fatal("enter on confirm should close after a good save")
	}
	got, err := LoadConfig(w.path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Workers != "anthropic.claude-haiku-5-5" || got.Judges != "anthropic.claude-opus-5-5" || got.Provider != "amazon-bedrock" {
		t.Fatalf("saved config wrong: %+v", got)
	}
	if len(got.Reviewers) != 1 || got.Reviewers[0] != "global.xai.grok-4.7" {
		t.Fatalf("reviewers wrong: %v", got.Reviewers)
	}
}

func TestSaveFailureStaysOpenAndShowsError(t *testing.T) {
	w, _ := newTestWizard(t)
	w.cfg.Provider = "amazon-bedrock"
	w.step = stepConfirm
	// Make the target path unwritable: a regular file where the directory should be.
	dir := filepath.Dir(filepath.Dir(w.path))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	blocker := filepath.Join(dir, "pstack")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if w.HandleKey("enter", "") {
		t.Fatal("a failed save must not close the panel")
	}
	_, lines, _ := w.Render()
	if !strings.Contains(strings.Join(lines, "\n"), "save failed") {
		t.Fatalf("save error not shown: %v", lines)
	}
}

func TestLoadMissingConfigInheritsEverything(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if cfg.Provider != "" || cfg.Workers != "" || cfg.Judges != "" || len(cfg.Reviewers) != 0 {
		t.Fatalf("missing file should be empty, got %+v", cfg)
	}
}

func TestConfigJSONUsesAdapterKeys(t *testing.T) {
	raw, err := json.Marshal(Config{Provider: "p", Workers: "w", Judges: "j", Reviewers: []string{"r"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{`"provider"`, `"workers"`, `"judges"`, `"reviewers"`} {
		if !strings.Contains(string(raw), k) {
			t.Errorf("missing key %s in %s", k, raw)
		}
	}
}

func TestConfigPathUsesZotHome(t *testing.T) {
	got, err := ConfigPath("/tmp/zh")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join("/tmp/zh", "pstack", "models.json") {
		t.Fatalf("got %s", got)
	}
}

func TestRenderShowsEachStep(t *testing.T) {
	w, _ := newTestWizard(t)
	titles := map[step]string{
		stepProvider:  "provider",
		stepWorkers:   "workers",
		stepJudges:    "judges",
		stepReviewers: "reviewers",
		stepConfirm:   "confirm",
	}
	w.cfg.Provider = "amazon-bedrock"
	for s, want := range titles {
		w.step = s
		title, _, _ := w.Render()
		if !strings.Contains(title, want) {
			t.Errorf("step %v: title %q lacks %q", s, title, want)
		}
	}
}

// On macOS the Delete key sends backspace. With an empty filter it must clear
// the reviewer list, the same as the delete key does.
func TestBackspaceClearsReviewersWhenFilterEmpty(t *testing.T) {
	w, _ := newTestWizard(t)
	w.cfg.Provider = "amazon-bedrock"
	w.step = stepReviewers
	w.cursor = 1
	w.HandleKey("enter", "")
	w.HandleKey("enter", "")
	if len(w.cfg.Reviewers) != 2 {
		t.Fatalf("want 2 reviewers, got %v", w.cfg.Reviewers)
	}
	w.HandleKey("backspace", "")
	if len(w.cfg.Reviewers) != 0 {
		t.Fatalf("backspace with empty filter should clear reviewers, got %v", w.cfg.Reviewers)
	}
}

// Backspace with text in the filter trims the filter and keeps the reviewers.
func TestBackspaceTrimsFilterWithoutClearingReviewers(t *testing.T) {
	w, _ := newTestWizard(t)
	w.cfg.Provider = "amazon-bedrock"
	w.step = stepReviewers
	w.cursor = 1
	w.HandleKey("enter", "")
	w.HandleKey("rune", "g")
	w.HandleKey("backspace", "")
	if w.query != "" {
		t.Fatalf("filter should be trimmed to empty, got %q", w.query)
	}
	if len(w.cfg.Reviewers) != 1 {
		t.Fatalf("backspace on a filter must not clear reviewers, got %v", w.cfg.Reviewers)
	}
}
