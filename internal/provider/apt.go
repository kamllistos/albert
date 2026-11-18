package provider

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/kamllistos/albert/internal/runner"
)

const searchCap = 20

type aptProvider struct{ r runner.Runner }

// NewAPT returns the Debian/Ubuntu apt provider.
func NewAPT(r runner.Runner) Provider { return &aptProvider{r: r} }

func (p *aptProvider) Name() string { return "apt" }

func (p *aptProvider) Available() bool {
	for _, c := range []string{"apt-get", "apt-mark", "dpkg-query"} {
		if _, err := p.r.LookPath(c); err != nil {
			return false
		}
	}
	return true
}

// manual returns the manually installed package names (apt-mark prints
// multiarch ones as name:arch).
func (p *aptProvider) manual(ctx context.Context) ([]string, error) {
	out, err := runOutput(ctx, p.r, "apt-mark", "showmanual")
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(out)), nil
}

func (p *aptProvider) List(ctx context.Context) ([]Tool, error) {
	names, err := p.manual(ctx)
	if err != nil || len(names) == 0 {
		return nil, err
	}
	// dpkg-query exits non-zero if any name is unknown but still prints the
	// rest, so only the output matters here.
	res, err := p.r.Run(ctx, "dpkg-query", append([]string{"-W", "-f=${Package}\t${Version}\n"}, names...)...)
	if err != nil {
		return nil, fmt.Errorf("dpkg-query: %w", err)
	}
	if res.ExitCode != 0 && len(res.Stdout) == 0 {
		return nil, cmdErr("dpkg-query", res)
	}
	versions := map[string]string{}
	for _, l := range strings.Split(string(res.Stdout), "\n") {
		if n, v, ok := strings.Cut(strings.TrimSpace(l), "\t"); ok {
			versions[n] = v
		}
	}
	tools := make([]Tool, 0, len(names))
	for _, n := range names {
		n, _, _ = strings.Cut(n, ":")
		tools = append(tools, Tool{Name: n, Source: p.Name(), Version: versions[n]})
	}
	return tools, nil
}

// Outdated reflects the last `apt update`; it is not run here because it
// needs root.
func (p *aptProvider) Outdated(ctx context.Context) ([]Update, error) {
	names, err := p.manual(ctx)
	if err != nil {
		return nil, err
	}
	manual := make(map[string]bool, len(names))
	for _, n := range names {
		n, _, _ = strings.Cut(n, ":")
		manual[n] = true
	}
	out, err := runOutput(ctx, p.r, "apt", "list", "--upgradable")
	if err != nil {
		return nil, err
	}
	var ups []Update
	for _, l := range strings.Split(string(out), "\n") {
		f := strings.Fields(l)
		name, _, ok := strings.Cut(strings.TrimSpace(l), "/") // skips "Listing..."
		if !ok || len(f) < 2 || !manual[name] {
			continue
		}
		ups = append(ups, Update{Name: name, Latest: f[1]})
	}
	return ups, nil
}

func (p *aptProvider) Upgrade(ctx context.Context, names []string, out io.Writer) error {
	if len(names) == 0 {
		return runStream(ctx, p.r, out, "sudo", "apt-get", "upgrade", "-y")
	}
	if err := checkNames("apt", names); err != nil {
		return err
	}
	return runStream(ctx, p.r, out, "sudo", append([]string{"apt-get", "install", "--only-upgrade", "-y"}, names...)...)
}

func (p *aptProvider) Install(ctx context.Context, name string, out io.Writer) error {
	if err := checkNames("apt", []string{name}); err != nil {
		return err
	}
	return runStream(ctx, p.r, out, "sudo", "apt-get", "install", "-y", name)
}

func (p *aptProvider) Remove(ctx context.Context, name string, out io.Writer) error {
	if err := checkNames("apt", []string{name}); err != nil {
		return err
	}
	return runStream(ctx, p.r, out, "sudo", "apt-get", "remove", "-y", name)
}

func (p *aptProvider) Search(ctx context.Context, q string) ([]Result, error) {
	q = strings.TrimSpace(q)
	if q == "" || strings.HasPrefix(q, "-") {
		return nil, nil // blank, or would be read as a flag
	}
	out, err := runOutput(ctx, p.r, "apt-cache", "search", "--names-only", q)
	if err != nil {
		return nil, err
	}
	var rs []Result
	for _, l := range strings.Split(string(out), "\n") {
		n, d, ok := strings.Cut(l, " - ")
		if n = strings.TrimSpace(n); !ok || n == "" {
			continue
		}
		rs = append(rs, Result{Name: n, Source: p.Name(), Description: strings.TrimSpace(d)})
		if len(rs) == searchCap {
			break
		}
	}
	return rs, nil
}

// WatchPaths is the dpkg state dir: apt replaces its status file by rename,
// which the watcher sees as a directory event.
func (p *aptProvider) WatchPaths() []string { return []string{"/var/lib/dpkg"} }
