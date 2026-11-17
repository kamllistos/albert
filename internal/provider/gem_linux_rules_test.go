package provider

import "testing"

func TestSystemGemDirOnLinux(t *testing.T) {
	for dir, want := range map[string]bool{
		"/usr/lib/ruby/gems/3.2.0":                     true,
		"/var/lib/gems/3.2.0":                          true,
		"/usr/local/lib/ruby/gems/3.2.0":               false,
		"/home/me/.local/share/gem/ruby/3.2.0":         false,
		"/home/linuxbrew/.linuxbrew/lib/ruby/gems/3.3": false,
	} {
		if got := systemGemDirOn("linux", dir); got != want {
			t.Errorf("%s: got %v, want %v", dir, got, want)
		}
	}
	if systemGemDirOn("darwin", "/usr/local/lib/ruby/gems/3.2.0") {
		t.Error("Intel Homebrew Ruby on macOS was treated as system Ruby")
	}
}
