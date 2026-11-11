package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/kamllistos/albert/internal/registry"
	"github.com/kamllistos/albert/internal/runner"
)

const githubPrefix = "github.com/"

// githubProvider finds Go tools on GitHub and installs them with `go install`.
// It owns no inventory: what it installs shows up under the go source.
type githubProvider struct {
	r   runner.Runner
	reg *registry.Client
}

// NewGitHub returns the search-and-install source for Go tools hosted on GitHub.
func NewGitHub(r runner.Runner, reg *registry.Client) Provider {
	return &githubProvider{r: r, reg: reg}
}

func (p *githubProvider) Name() string { return "github" }

func (p *githubProvider) Available() bool {
	_, err := p.r.LookPath("go")
	return err == nil
}

func (p *githubProvider) List(context.Context) ([]Tool, error)       { return nil, nil }
func (p *githubProvider) Outdated(context.Context) ([]Update, error) { return nil, nil }

func (p *githubProvider) Upgrade(context.Context, []string, io.Writer) error { return ErrUnsupported }
func (p *githubProvider) Remove(context.Context, string, io.Writer) error    { return ErrUnsupported }

func (p *githubProvider) Install(ctx context.Context, name string, out io.Writer) error {
	if err := checkNames("github", []string{name}); err != nil {
		return err
	}
	if !strings.HasPrefix(name, githubPrefix) {
		return fmt.Errorf("github: %q is not a github.com repository", name)
	}
	return runStream(ctx, p.r, out, "go", goInstallArgs(name)...)
}

func (p *githubProvider) Search(ctx context.Context, q string) ([]Result, error) {
	if p.reg == nil {
		return nil, errors.New("github: no registry client")
	}
	pkgs, err := p.reg.GitHubSearch(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("github: %w", err)
	}
	rs := make([]Result, 0, len(pkgs))
	for _, k := range pkgs {
		rs = append(rs, Result{Name: githubPrefix + k.Name, Source: p.Name(), Version: k.Version, Description: k.Description})
	}
	return rs, nil
}

func (p *githubProvider) WatchPaths() []string { return nil }
