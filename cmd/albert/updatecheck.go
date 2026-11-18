package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kamllistos/albert/internal/registry"
	"github.com/kamllistos/albert/internal/state"
)

const (
	updateCheckFile = "update-check.json"
	updateCheckTTL  = 24 * time.Hour
	updateWait      = 2 * time.Second
)

// fetchLatest looks up a newer release; tests replace it.
var fetchLatest = func(ctx context.Context, current string) (string, error) {
	return registry.New().NewerAlbert(ctx, current)
}

type updateCache struct {
	CheckedAt int64  `json:"checkedAt"`
	Latest    string `json:"latest"`
}

// updateCheck is one command's view of the daily release check.
type updateCheck struct {
	current string
	cached  string      // newer version from the cache, "" if none
	result  chan string // nil when the cache is fresh
}

// startUpdateCheck returns nil when the notice must not show: dev builds,
// help/agent, ALBERT_NO_UPDATE_CHECK=1, or stderr not a terminal.
func (a *app) startUpdateCheck(cmd string) *updateCheck {
	f, ok := a.stderr.(*os.File)
	if version == "dev" || cmd == "agent" || os.Getenv("ALBERT_NO_UPDATE_CHECK") == "1" || !ok || !isTerminal(f) {
		return nil
	}
	return newUpdateCheck(state.Dir(), version, fetchLatest, time.Now)
}

func newUpdateCheck(dir, current string, fetch func(context.Context, string) (string, error), now func() time.Time) *updateCheck {
	path := filepath.Join(dir, updateCheckFile)
	u := &updateCheck{current: current}
	var c updateCache
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &c)
	}
	u.cached = c.Latest
	if c.CheckedAt != 0 && now().Sub(time.Unix(c.CheckedAt, 0)) < updateCheckTTL {
		return u
	}
	u.result = make(chan string, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), updateWait)
		defer cancel()
		v, err := fetch(ctx, current)
		if err != nil {
			// Keep the old answer but mark it checked, so an offline user
			// pays the wait at most once a day.
			v = u.cached
		}
		writeUpdateCache(path, updateCache{CheckedAt: now().Unix(), Latest: v})
		u.result <- v
	}()
	return u
}

// writeUpdateCache is best effort: temp file + rename so readers never see half a file.
func writeUpdateCache(path string, c updateCache) {
	b, _ := json.Marshal(c)
	tmp, err := os.CreateTemp(filepath.Dir(path), "update-check-*")
	if err != nil {
		return
	}
	_, werr := tmp.Write(b)
	if cerr := tmp.Close(); werr != nil || cerr != nil || os.Rename(tmp.Name(), path) != nil {
		os.Remove(tmp.Name())
	}
}

// notice prints the update line if a newer release is known. With wait set it
// waits up to 2s for a running check (only when the cache was stale, so at
// most once a day); otherwise a fast command would exit before it finishes.
func (u *updateCheck) notice(w io.Writer, wait bool) {
	if u == nil {
		return
	}
	latest := u.cached
	if u.result != nil {
		if wait {
			select {
			case latest = <-u.result:
			case <-time.After(updateWait):
			}
		} else {
			select {
			case latest = <-u.result:
			default:
			}
		}
	}
	// Re-check against the running version: the cache may predate an upgrade.
	if latest != "" && registry.NewerVersion(latest, u.current) {
		fmt.Fprintf(w, "\nAlbert v%s is available (you have v%s). Update with: %s\n", strings.TrimPrefix(latest, "v"), strings.TrimPrefix(u.current, "v"), upgradeHint())
	}
}
