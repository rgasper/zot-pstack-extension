// pstack-setup is a zot extension that records which provider and model
// each pstack role group uses. It registers /pstack-setup, opens a panel
// that walks the user through provider, workers, judges, and reviewers, and
// writes $ZOT_HOME/pstack/models.json for the pstack adapter to read.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

const (
	protocolVersion = 1
	extName         = "pstack-setup"
	extVersion      = "0.1.0"
	commandName     = "pstack-setup"

	// frameLimit matches zot's scanner buffer. Host frames can carry large
	// model lists, so the default 64 KiB limit is too small.
	frameLimit = 4 * 1024 * 1024
)

func main() {
	if err := run(os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "pstack-setup:", err)
		os.Exit(1)
	}
}

// session owns the single open wizard. Only one panel can be open at a
// time, and keys for any other panel id are ignored.
type session struct {
	mu  sync.Mutex
	h   *host
	cur *wizard
}

func run(in io.Reader, out io.Writer, logw io.Writer) error {
	h := &host{out: out}
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), frameLimit)

	if err := h.send(helloOut{Type: "hello", Name: extName, Version: extVersion, Capabilities: []string{"commands", "panels"}}); err != nil {
		return err
	}
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return err
		}
		return fmt.Errorf("zot closed stdin before hello_ack")
	}
	var ack struct {
		Type            string `json:"type"`
		ProtocolVersion int    `json:"protocol_version"`
	}
	if err := json.Unmarshal(scanner.Bytes(), &ack); err != nil {
		return fmt.Errorf("parse hello_ack: %w", err)
	}
	if ack.Type != "hello_ack" {
		return fmt.Errorf("first host frame must be hello_ack, got %q", ack.Type)
	}
	if ack.ProtocolVersion != protocolVersion {
		return fmt.Errorf("protocol mismatch: extension speaks v%d, zot speaks v%d", protocolVersion, ack.ProtocolVersion)
	}

	if err := h.send(registerCommandOut{Type: "register_command", Name: commandName, Description: "choose pstack models per role group"}); err != nil {
		return err
	}
	if err := h.send(registerCommandOut{Type: "register_command", Name: potetoModeCommandName, Description: potetoModeDescription}); err != nil {
		return err
	}
	if err := h.send(readyOut{Type: "ready"}); err != nil {
		return err
	}

	s := &session{h: h}
	for scanner.Scan() {
		var f inFrame
		if err := json.Unmarshal(scanner.Bytes(), &f); err != nil {
			fmt.Fprintln(logw, "malformed frame:", err)
			continue
		}
		switch f.Type {
		case "command_invoked":
			switch f.Name {
			case potetoModeCommandName:
				_ = h.send(commandResponseOut{Type: "command_response", ID: f.ID, Action: "prompt", Prompt: potetoModePrompt(f.Args)})
			case commandName:
				// Loading models shells out to zot, so run it off the read loop.
				go s.open(f.ID)
			default:
				_ = h.send(commandResponseOut{Type: "command_response", ID: f.ID, Action: "noop", Error: "no handler for /" + f.Name})
			}
		case "panel_key":
			s.key(f.PanelID, f.Key, f.Text)
		case "panel_close":
			s.closed(f.PanelID)
		case "shutdown":
			_ = h.send(shutdownAckOut{Type: "shutdown_ack"})
			return nil
		}
	}
	return scanner.Err()
}

// open answers a /pstack-setup invocation. It loads the model list, then
// either opens the panel or returns the reason it could not.
func (s *session) open(id string) {
	w, err := newWizard(s.h)
	if err != nil {
		_ = s.h.send(commandResponseOut{Type: "command_response", ID: id, Action: "noop", Error: err.Error()})
		return
	}
	s.mu.Lock()
	s.cur = w
	s.mu.Unlock()
	title, lines, footer := w.Render()
	_ = s.h.send(commandResponseOut{
		Type:      "command_response",
		ID:        id,
		Action:    "open_panel",
		OpenPanel: &panelSpec{ID: panelID, Title: title, Lines: lines, Footer: footer},
	})
}

func (s *session) key(panel, key, text string) {
	s.mu.Lock()
	w := s.cur
	s.mu.Unlock()
	if w == nil || panel != panelID {
		return
	}
	if closePanel := w.HandleKey(key, text); closePanel {
		s.closed(panel)
		_ = s.h.send(panelCloseOut{Type: "panel_close", PanelID: panelID})
	}
}

func (s *session) closed(panel string) {
	if panel != panelID {
		return
	}
	s.mu.Lock()
	s.cur = nil
	s.mu.Unlock()
}

// newWizard loads the model list from zot and the saved config, and
// returns a wizard on the provider step.
func newWizard(h *host) (*wizard, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	entries, err := listModels(ctx)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no models available: log in to a provider first")
	}
	path, err := ConfigPath("")
	if err != nil {
		return nil, err
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}
	return &wizard{host: h, path: path, entries: entries, cfg: cfg, step: stepProvider, autoSwarm: autoSwarmOn()}, nil
}
