package main

import "strings"

// potetoModeCommandName is the /poteto-mode slash command. It loads pstack's
// poteto-mode skill, the entry point the pstack skills expect.
//
// Usage:
//
//	/poteto-mode                 enter poteto-mode and wait for the next message
//	/poteto-mode <your request>  enter poteto-mode, route the request to a playbook, act on it
//
// Both forms reply with a prompt rather than a display. A display is only a
// note to the user; the model needs the skill body in context to act on it.
// The skill tool loads the body by name, so the extension never reads
// SKILL.md itself.
const (
	potetoModeCommandName = "poteto-mode"
	potetoModeDescription = "enter pstack's poteto-mode; add a request to route to a playbook"
)

// potetoModeSetupPrompt loads the skill. Used with and without a request.
const potetoModeSetupPrompt = "Load the pstack:poteto-mode skill with the skill tool, and follow its instructions for the rest of this session. Load pstack:zot-adapter first if you have not already in this session, since it maps the Cursor-era instructions to zot."

// potetoModeRouteInstruction is appended when a request is given. The
// playbook list in the poteto-mode skill is the routing table. The model must
// use it, not answer the request freely, because the playbook files carry the
// steps it is supposed to follow.
const potetoModeRouteInstruction = `Route this request before doing anything else:

1. Match the request against the playbook list in the poteto-mode skill's "Playbooks" section.
2. If one playbook clearly fits, open its file at playbooks/<title>.md (for example playbooks/investigation.md) with the read tool, and follow its steps. Say which playbook you picked in one line.
3. If two playbooks fit equally well, or none fits, do not guess. Ask one question that names the top two options, or says that none fits and offers figure-it-out.

The request:`

// potetoModePrompt builds the prompt for /poteto-mode. An empty request loads
// the skill and waits. A request is routed through the playbook list.
func potetoModePrompt(request string) string {
	request = strings.TrimSpace(request)
	if request == "" {
		return potetoModeSetupPrompt
	}
	return potetoModeSetupPrompt + "\n\n" + potetoModeRouteInstruction + "\n\n" + request
}
