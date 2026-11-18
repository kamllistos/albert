package provider

import (
	"context"
	"io"
	"strings"

	"github.com/kamllistos/albert/internal/runner"
)

type snapProvider struct{ r runner.Runner }

// NewSnap returns the Snap provider.
func NewSnap(r runner.Runner) Provider { return &snapProvider{r: r} }

func (p *snapProvider) Name() string { return "snap" }

func (p *snapProvider) Available() bool {
	_, err := p.r.LookPath("snap")
	return err == nil
}

// snapRows returns the whitespace-split data rows of snap table output,
// without the header row.
func snapRows(out []byte) [][]string {
	var rows [][]string
	for i, l := range strings.Split(string(out), "\n") {
		if f := strings.Fields(l); i > 0 && len(f) >= 2 {
			rows = append(rows, f)
		}
	}
	return rows
}

func (p *snapProvider) List(ctx context.Context) ([]Tool, error) {
	out, err := runOutput(ctx, p.r, "snap", "list")
	if err != nil {
		return nil, err
	}
	var tools []Tool
	for _, f := range snapRows(out) {
		t := Tool{Name: f[0], Source: p.Name(), Version: f[1]}
		if len(f) >= 6 { // Notes is the last column, e.g. "base" or "disabled,base"
			for _, n := range strings.Split(f[len(f)-1], ",") {
				if n == "base" || n == "snapd" || n == "core" {
					t.Dependency = true
				}
			}
		}
		tools = append(tools, t)
	}
	return tools, nil
}

func (p *snapProvider) Outdated(ctx context.Context) ([]Update, error) {
	out, err := runOutput(ctx, p.r, "snap", "refresh", "--list")
	if err != nil {
		return nil, err
	}
	// "All snaps up to date." goes to stderr, so stdout is empty then.
	var ups []Update
	for _, f := range snapRows(out) {
		ups = append(ups, Update{Name: f[0], Latest: f[1]})
	}
	return ups, nil
}

func (p *snapProvider) Upgrade(ctx context.Context, names []string, out io.Writer) error {
	if err := checkNames("snap", names); err != nil {
		return err
	}
	return runStream(ctx, p.r, out, "sudo", append([]string{"snap", "refresh"}, names...)...)
}

func (p *snapProvider) Install(ctx context.Context, name string, out io.Writer) error {
	if err := checkNames("snap", []string{name}); err != nil {
		return err
	}
	return runStream(ctx, p.r, out, "sudo", "snap", "install", name)
}

func (p *snapProvider) Remove(ctx context.Context, name string, out io.Writer) error {
	if err := checkNames("snap", []string{name}); err != nil {
		return err
	}
	return runStream(ctx, p.r, out, "sudo", "snap", "remove", name)
}

func (p *snapProvider) Search(ctx context.Context, q string) ([]Result, error) {
	q = strings.TrimSpace(q)
	if q == "" || strings.HasPrefix(q, "-") {
		return nil, nil // blank, or would be read as a flag
	}
	out, err := runOutput(ctx, p.r, "snap", "find", q)
	if err != nil {
		return nil, err
	}
	var rs []Result
	for _, f := range snapRows(out) {
		r := Result{Name: f[0], Source: p.Name(), Version: f[1]}
		if len(f) > 4 { // Name Version Publisher Notes Summary...
			r.Description = strings.Join(f[4:], " ")
		}
		rs = append(rs, r)
		if len(rs) == searchCap {
			break
		}
	}
	return rs, nil
}

// WatchPaths is the dir holding the installed snap images; it changes on
// every install, remove and refresh.
func (p *snapProvider) WatchPaths() []string { return []string{"/var/lib/snapd/snaps"} }
