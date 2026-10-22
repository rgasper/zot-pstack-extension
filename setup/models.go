package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// listModels runs `zot --list-models-json` and decodes the result. The
// binary is zot's own, so the set matches what /models offers.
func listModels(ctx context.Context) ([]Entry, error) {
	cmd := exec.CommandContext(ctx, "zot", "--list-models-json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("zot --list-models-json failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var entries []Entry
	if err := json.Unmarshal(stdout.Bytes(), &entries); err != nil {
		return nil, fmt.Errorf("parse zot --list-models-json: %w", err)
	}
	return entries, nil
}
