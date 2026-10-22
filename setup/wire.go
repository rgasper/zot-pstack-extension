// Package main is the pstack-setup command, a zot extension that records
// which provider and model each pstack role group uses.
//
// The wire types here copy the subset of zot's extension protocol this
// command needs (zot packages/agent/extproto, protocol v1). They are copied
// rather than imported so the module builds on any machine with plain Go.
// Re-check them when zot's protocol version changes.
package main

import (
	"encoding/json"
	"io"
	"sync"
)

// inFrame holds every host frame field this command reads.
type inFrame struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Args    string `json:"args,omitempty"`
	PanelID string `json:"panel_id"`
	Key     string `json:"key"`
	Text    string `json:"text"`
}

type helloOut struct {
	Type         string   `json:"type"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type registerCommandOut struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type readyOut struct {
	Type string `json:"type"`
}

type panelSpec struct {
	ID     string   `json:"id"`
	Title  string   `json:"title,omitempty"`
	Lines  []string `json:"lines,omitempty"`
	Footer string   `json:"footer,omitempty"`
}

type commandResponseOut struct {
	Type      string     `json:"type"`
	ID        string     `json:"id"`
	Action    string     `json:"action"`
	Prompt    string     `json:"prompt,omitempty"`
	OpenPanel *panelSpec `json:"open_panel,omitempty"`
	Error     string     `json:"error,omitempty"`
}

type panelRenderOut struct {
	Type    string   `json:"type"`
	PanelID string   `json:"panel_id"`
	Title   string   `json:"title,omitempty"`
	Lines   []string `json:"lines,omitempty"`
	Footer  string   `json:"footer,omitempty"`
}

type panelCloseOut struct {
	Type    string `json:"type"`
	PanelID string `json:"panel_id"`
}

type shutdownAckOut struct {
	Type string `json:"type"`
}

// host writes one JSON frame per line to zot. Command handlers and panel
// key handlers can write at the same time, so writes are serialised.
type host struct {
	mu  sync.Mutex
	out io.Writer
}

func (h *host) send(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err = h.out.Write(append(b, '\n'))
	return err
}
