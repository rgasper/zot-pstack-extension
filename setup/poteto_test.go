package main

import (
	"strings"
	"testing"
)

func TestPotetoModePromptWithoutRequestIsSetupOnly(t *testing.T) {
	got := potetoModePrompt("")
	if got != potetoModeSetupPrompt {
		t.Fatalf("empty request should give the setup prompt only")
	}
	if strings.Contains(got, "user's request") {
		t.Fatal("setup-only prompt must not mention a request")
	}
}

func TestPotetoModePromptWhitespaceRequestIsSetupOnly(t *testing.T) {
	if potetoModePrompt("   \t ") != potetoModeSetupPrompt {
		t.Fatal("whitespace-only request should count as no request")
	}
}

func TestPotetoModePromptCarriesRequestAfterSetup(t *testing.T) {
	got := potetoModePrompt("  fix the flaky test  ")
	if !strings.HasPrefix(got, potetoModeSetupPrompt) {
		t.Fatal("request prompt must still load the skill first")
	}
	if !strings.HasSuffix(got, "fix the flaky test") {
		t.Fatalf("request must be carried, trimmed, at the end: %q", got)
	}
}
