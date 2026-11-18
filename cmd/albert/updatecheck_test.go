package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestUpdateCheck(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	nowFn := func() time.Time { return now }
	fresh := now.Add(-time.Hour).Unix()
	stale := now.Add(-48 * time.Hour).Unix()
	tests := []struct {
		name      string
		cache     string // "" = no file
		fetch     string
		fetchErr  error
		wait      bool
		wantCalls int
		wantOut   string // substring; "" = no output
		wantCache string // latest in cache file after, "" = unchanged/none
	}{
		{name: "fresh cache newer", cache: `{"checkedAt":FRESH,"latest":"0.2.0"}`, wantOut: "Albert v0.2.0 is available (you have v0.1.0)"},
		{name: "fresh cache already upgraded", cache: `{"checkedAt":FRESH,"latest":"0.1.0"}`},
		{name: "fresh cache empty", cache: `{"checkedAt":FRESH,"latest":""}`},
		{name: "missing cache, waits, newer", fetch: "0.3.0", wait: true, wantCalls: 1, wantOut: "0.3.0 is available", wantCache: "0.3.0"},
		{name: "stale cache, no wait, uses cached", cache: `{"checkedAt":STALE,"latest":"0.2.0"}`, fetch: "0.3.0", wantCalls: 1, wantOut: "0.2.0 is available"},
		{name: "stale, waits, up to date", cache: `{"checkedAt":STALE,"latest":"0.2.0"}`, fetch: "", wait: true, wantCalls: 1},
		{name: "fetch error keeps cached, marks checked", cache: `{"checkedAt":STALE,"latest":"0.2.0"}`, fetchErr: errors.New("offline"), wait: true, wantCalls: 1, wantOut: "0.2.0 is available", wantCache: "0.2.0"},
		{name: "corrupt cache refetches", cache: `nope`, fetch: "0.2.0", wait: true, wantCalls: 1, wantOut: "0.2.0 is available", wantCache: "0.2.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, updateCheckFile)
			if tt.cache != "" {
				c := strings.NewReplacer("FRESH", itoa(fresh), "STALE", itoa(stale)).Replace(tt.cache)
				if err := os.WriteFile(path, []byte(c), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			calls := 0
			fetch := func(context.Context, string) (string, error) {
				if !tt.wait {
					time.Sleep(200 * time.Millisecond) // still running when notice looks
				}
				calls++
				return tt.fetch, tt.fetchErr
			}
			u := newUpdateCheck(dir, "0.1.0", fetch, nowFn)
			var out bytes.Buffer
			u.notice(&out, tt.wait)
			if tt.wantOut == "" && out.Len() > 0 {
				t.Errorf("unexpected output %q", out.String())
			}
			if !strings.Contains(out.String(), tt.wantOut) {
				t.Errorf("output %q, want %q", out.String(), tt.wantOut)
			}
			if tt.wait && calls != tt.wantCalls {
				t.Errorf("fetch calls = %d, want %d", calls, tt.wantCalls)
			}
			if tt.wantCache != "" {
				b, _ := os.ReadFile(path)
				if tt.fetchErr == nil && !strings.Contains(string(b), `"latest":"`+tt.wantCache+`"`) {
					t.Errorf("cache = %s, want latest %s", b, tt.wantCache)
				}
			}
		})
	}
}

func TestUpdateCheckNilSafe(t *testing.T) {
	var u *updateCheck
	u.notice(&bytes.Buffer{}, true)
}

func TestUpgradeHintGoInstall(t *testing.T) {
	defer func(old bool) { goInstalled = old }(goInstalled)
	goInstalled = false
	if h := upgradeHint(); !strings.HasPrefix(h, "curl ") {
		t.Errorf("script hint = %q", h)
	}
	goInstalled = true
	if h := upgradeHint(); !strings.HasPrefix(h, "go install ") {
		t.Errorf("go hint = %q", h)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
