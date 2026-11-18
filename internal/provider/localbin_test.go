package provider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestParseVersion(t *testing.T) {
	for in, want := range map[string]string{
		"kiro-cli 3.4.0":    "3.4.0",
		"v1.2.3-rc1":        "1.2.3-rc1",
		"tool version 2.10": "2.10",
		"no version":        "",
	} {
		if got := parseVersion(in); got != want {
			t.Errorf("parseVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLocalBin(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(filepath.Join(bin, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, body string, mode os.FileMode) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), mode); err != nil {
			t.Fatal(err)
		}
		_ = os.Chmod(path, mode)
	}
	write(filepath.Join(bin, "tool"), "#!/bin/sh\n", 0o755)
	write(filepath.Join(bin, "data.txt"), "x", 0o644)
	pipxApp := filepath.Join(root, "share", "pipx", "venvs", "x", "bin", "x")
	if err := os.MkdirAll(filepath.Dir(pipxApp), 0o755); err != nil {
		t.Fatal(err)
	}
	write(pipxApp, "#!/bin/sh\n", 0o755)
	if err := os.Symlink(pipxApp, filepath.Join(bin, "x")); err != nil {
		t.Fatal(err)
	}

	var calls atomic.Int32
	p := &localBinProvider{dir: bin, cacheDir: filepath.Join(root, "cache"), probe: func(_ context.Context, path string) string {
		calls.Add(1)
		return "1.0." + filepath.Base(path)[:1]
	}}
	if err := os.MkdirAll(p.cacheDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if !p.Available() {
		t.Fatal("not available")
	}

	check := func(wantVersion string) {
		t.Helper()
		tools, err := p.List(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(tools) != 1 {
			t.Fatalf("got %+v, want only tool", tools)
		}
		g := tools[0]
		if g.Name != "tool" || g.Version != wantVersion || !g.ReadOnly || g.Source != "localbin" || g.Path != filepath.Join(bin, "tool") {
			t.Errorf("tool = %+v", g)
		}
	}
	check("1.0.t")
	if calls.Load() != 1 {
		t.Fatalf("probe calls = %d, want 1", calls.Load())
	}
	check("1.0.t")
	if calls.Load() != 1 {
		t.Errorf("cache miss: probe calls = %d, want 1", calls.Load())
	}

	write(filepath.Join(bin, "tool"), "#!/bin/sh\n# changed\n", 0o755)
	p.probe = func(context.Context, string) string { calls.Add(1); return "2.0.0" }
	check("2.0.0")
	if calls.Load() != 2 {
		t.Errorf("probe calls after change = %d, want 2", calls.Load())
	}
}

func TestLocalBinReadOnly(t *testing.T) {
	p := &localBinProvider{}
	if err := p.Install(context.Background(), "a", nil); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Install err = %v", err)
	}
	if err := p.Upgrade(context.Background(), nil, nil); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Upgrade err = %v", err)
	}
	if err := p.Remove(context.Background(), "a", nil); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Remove err = %v", err)
	}
}
