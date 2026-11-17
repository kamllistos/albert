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

func TestIsHomebrewPathOnLinux(t *testing.T) {
	if !isHomebrewPath("/home/linuxbrew/.linuxbrew/bin/albert") {
		t.Error("a Linuxbrew program was not treated as Homebrew's")
	}
	if isHomebrewPath("/home/me/.local/bin/albert") {
		t.Error("a ~/.local/bin program was treated as Homebrew's")
	}
}
