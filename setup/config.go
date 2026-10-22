package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Group names. Each role in the pstack skills belongs to exactly one.
const (
	groupWorkers   = "workers"
	groupJudges    = "judges"
	groupReviewers = "reviewers"
)

// Config is the file the pstack adapter reads. A missing key means the
// group inherits the parent chat model. Keep this shape in sync with the
// adapter's documentation in skills/zot-adapter/SKILL.md.
type Config struct {
	Provider  string   `json:"provider,omitempty"`
	Workers   string   `json:"workers,omitempty"`
	Judges    string   `json:"judges,omitempty"`
	Reviewers []string `json:"reviewers,omitempty"`
}

// Entry is one model from `zot --list-models-json`.
type Entry struct {
	Provider      string `json:"provider"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	Reasoning     bool   `json:"reasoning"`
	ContextWindow int    `json:"context_window"`
	MaxOutput     int    `json:"max_output"`
	Source        string `json:"source"`
}

// ConfigPath returns $ZOT_HOME/pstack/models.json, falling back to zot's
// platform default when ZOT_HOME is unset. The host passes its data dir
// in hello_ack, but that is the extension's own dir, not zot's home, so
// the default mirrors zot's ZotHome() resolution.
func ConfigPath(zotHome string) (string, error) {
	if zotHome == "" {
		var err error
		zotHome, err = defaultZotHome()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(zotHome, "pstack", "models.json"), nil
}

func defaultZotHome() (string, error) {
	if v := os.Getenv("ZOT_HOME"); v != "" {
		return v, nil
	}
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return filepath.Join(v, "zot"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "zot"), nil
	case "windows":
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return filepath.Join(v, "zot"), nil
		}
	}
	return filepath.Join(home, ".local", "state", "zot"), nil
}

// LoadConfig reads the config. A missing file returns an empty Config and
// no error, because "no file" means every group inherits the parent model.
func LoadConfig(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return c, nil
}

// SaveConfig writes the config atomically: temp file, then rename.
func SaveConfig(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Providers returns the distinct provider ids in entries, in first-seen
// order. entries are already sorted by provider, so this is alphabetical.
func Providers(entries []Entry) []string {
	var out []string
	seen := map[string]bool{}
	for _, e := range entries {
		if !seen[e.Provider] {
			seen[e.Provider] = true
			out = append(out, e.Provider)
		}
	}
	return out
}

// ModelsFor returns the entries for one provider.
func ModelsFor(entries []Entry, provider string) []Entry {
	var out []Entry
	for _, e := range entries {
		if e.Provider == provider {
			out = append(out, e)
		}
	}
	return out
}

// Describe renders one config line for the panel and for confirmations.
func Describe(c Config) []string {
	orInherit := func(s string) string {
		if s == "" {
			return "inherit parent model"
		}
		return s
	}
	rev := "inherit parent model"
	if len(c.Reviewers) > 0 {
		rev = strings.Join(c.Reviewers, ", ")
	}
	return []string{
		"provider:  " + orInherit(c.Provider),
		"workers:   " + orInherit(c.Workers),
		"judges:    " + orInherit(c.Judges),
		"reviewers: " + rev,
	}
}
