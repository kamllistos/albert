package main

import "testing"

func TestOpenCommandPerOS(t *testing.T) {
	if got := openCommand("darwin"); got != "open" {
		t.Errorf("macOS: got %q, want open", got)
	}
	if got := openCommand("linux"); got != "xdg-open" {
		t.Errorf("Linux: got %q, want xdg-open", got)
	}
}
