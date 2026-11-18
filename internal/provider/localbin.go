package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kamllistos/albert/internal/state"
)

const (
	probeTimeout = 3 * time.Second
	probeMaxOut  = 4 << 10
	probeLimit   = 8
	localbinFile = "localbin-versions.json"
)

var localbinVersion = regexp.MustCompile(`\bv?(\d+\.\d+(?:\.\d+)?(?:[-+][0-9A-Za-z.-]+)?)\b`)

// binStamp is a cached probe result, valid while size and mtime match.
type binStamp struct {
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTimeUnixNano"`
	Version string `json:"version"`
}

type localBinProvider struct {
	dir      string
	cacheDir string
	probe    func(ctx context.Context, path string) string // injectable for tests
}

// NewLocalBin returns the read-only ~/.local/bin provider.
func NewLocalBin() Provider {
	home, _ := os.UserHomeDir()
	return &localBinProvider{dir: filepath.Join(home, ".local", "bin"), cacheDir: state.Dir(), probe: probeVersion}
}

func (p *localBinProvider) Name() string { return "localbin" }

func (p *localBinProvider) Available() bool {
	fi, err := os.Stat(p.dir)
	return err == nil && fi.IsDir()
}

// capBuf keeps the first probeMaxOut bytes and discards the rest, so a noisy
// program never blocks on a full pipe or grows our memory.
type capBuf struct{ b []byte }

func (c *capBuf) Write(d []byte) (int, error) {
	if room := probeMaxOut - len(c.b); room > 0 {
		c.b = append(c.b, d[:min(room, len(d))]...)
	}
	return len(d), nil
}

// parseVersion returns the first version-looking token in s, or "".
func parseVersion(s string) string {
	if m := localbinVersion.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

// probeVersion runs `path --version`, then `path version`. It uses os/exec
// directly: the shared runner has no output cap and its SIGTERM-first cancel
// is too gentle for arbitrary user programs.
func probeVersion(ctx context.Context, path string) string {
	for _, arg := range []string{"--version", "version"} {
		cctx, cancel := context.WithTimeout(ctx, probeTimeout)
		cmd := exec.CommandContext(cctx, path, arg) // nil Stdin = /dev/null
		buf := &capBuf{}
		cmd.Stdout, cmd.Stderr = buf, buf
		cmd.WaitDelay = time.Second // a child holding the pipe cannot hang us
		_ = cmd.Run()
		cancel()
		if v := parseVersion(string(buf.b)); v != "" {
			return v
		}
	}
	return ""
}

type binEntry struct {
	name, path, key string
	stamp           binStamp
}

// entries lists the programs Albert should show, skipping anything other
// providers own or that is not a plain executable of ours.
func (p *localBinProvider) entries() []binEntry {
	des, err := os.ReadDir(p.dir)
	if err != nil {
		return nil
	}
	uid := uint32(os.Getuid())
	var out []binEntry
	for _, de := range des {
		path := filepath.Join(p.dir, de.Name())
		real, err := filepath.EvalSymlinks(path)
		if err != nil {
			continue
		}
		slash := filepath.ToSlash(real)
		if strings.Contains(slash, "/pipx/") || strings.Contains(slash, "/uv/tools/") || strings.Contains(slash, "/Cellar/") {
			continue
		}
		fi, err := os.Stat(real)
		if err != nil || !fi.Mode().IsRegular() || fi.Mode().Perm()&0o100 == 0 || fi.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
			continue
		}
		if st, ok := fi.Sys().(*syscall.Stat_t); !ok || st.Uid != uid {
			continue
		}
		out = append(out, binEntry{de.Name(), path, real, binStamp{Size: fi.Size(), ModTime: fi.ModTime().UnixNano()}})
	}
	return out
}

func (p *localBinProvider) List(ctx context.Context) ([]Tool, error) {
	if !p.Available() {
		return nil, nil
	}
	cacheFile := filepath.Join(p.cacheDir, localbinFile)
	var old map[string]binStamp
	if b, err := os.ReadFile(cacheFile); err == nil {
		_ = json.Unmarshal(b, &old) // a bad cache is an empty cache
	}
	es := p.entries()
	fresh := make(map[string]binStamp, len(es))
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, probeLimit)
	for i := range es {
		e := &es[i]
		if c, ok := old[e.key]; ok && c.Size == e.stamp.Size && c.ModTime == e.stamp.ModTime {
			e.stamp.Version = c.Version
			fresh[e.key] = e.stamp
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem; wg.Done() }()
			e.stamp.Version = p.probe(ctx, e.path)
			mu.Lock()
			fresh[e.key] = e.stamp
			mu.Unlock()
		}()
	}
	wg.Wait()
	if ctx.Err() != nil {
		return nil, ctx.Err() // probes were cut short; do not cache their blanks
	}
	if !sameCache(old, fresh) {
		writeCache(cacheFile, fresh)
	}
	tools := make([]Tool, len(es))
	for i, e := range es {
		tools[i] = Tool{Name: e.name, Source: p.Name(), Version: e.stamp.Version, Path: e.path, ReadOnly: true}
	}
	return tools, nil
}

func sameCache(a, b map[string]binStamp) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// writeCache is best effort: a failed write only costs a re-probe next time.
func writeCache(path string, m map[string]binStamp) {
	b, err := json.Marshal(m)
	if err != nil {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), localbinFile+".*")
	if err != nil {
		return
	}
	_, werr := tmp.Write(b)
	if cerr := tmp.Close(); werr != nil || cerr != nil || os.Rename(tmp.Name(), path) != nil {
		os.Remove(tmp.Name())
	}
}

func (p *localBinProvider) Outdated(context.Context) ([]Update, error) { return nil, nil }

func (p *localBinProvider) Upgrade(_ context.Context, names []string, _ io.Writer) error {
	return fmt.Errorf("localbin: %w", ErrReadOnly)
}

func (p *localBinProvider) Install(_ context.Context, name string, _ io.Writer) error {
	return fmt.Errorf("localbin: %s: %w", name, ErrReadOnly)
}

func (p *localBinProvider) Remove(_ context.Context, name string, _ io.Writer) error {
	return fmt.Errorf("localbin: %s: %w", name, ErrReadOnly)
}

func (p *localBinProvider) Search(context.Context, string) ([]Result, error) { return nil, nil }

func (p *localBinProvider) WatchPaths() []string { return []string{p.dir} }
