package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// autoSwarmOn reports whether zot's auto-swarm setting is on.
//
// zot does not send this flag to extensions in hello_ack, so it is read from
// $ZOT_HOME/config.json. The default matches zot's AutoSwarmEnabled(): a
// missing file, a missing key, or any value other than true all mean off.
// A read or parse error also counts as off, because the warning is the safe
// answer when the setting cannot be confirmed.
func autoSwarmOn() bool {
	path, err := zotConfigPath()
	if err != nil {
		return false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var c struct {
		AutoSwarmEnabled *bool `json:"auto_swarm_enabled"`
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		return false
	}
	return c.AutoSwarmEnabled != nil && *c.AutoSwarmEnabled
}

// zotConfigPath returns $ZOT_HOME/config.json, or zot's platform default
// when ZOT_HOME is unset. It shares the home lookup with ConfigPath.
func zotConfigPath() (string, error) {
	home, err := defaultZotHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "config.json"), nil
}
