package main

// autoSwarmWarning is shown on the provider and confirm steps when zot's
// auto-swarm setting is off. It is bold, with a marker, so it stands out
// from the plain rows around it. The ANSI codes are closed on the same line.
const autoSwarmWarning = "\x1b[1m⚠ WARNING: auto-swarm is OFF. pstack's parallel steps need it. Turn it on in /settings.\x1b[22m"
