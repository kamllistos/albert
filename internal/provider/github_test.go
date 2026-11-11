package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/kamllistos/albert/internal/registry"
	"github.com/kamllistos/albert/internal/runner"
)

func TestGitHubSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/repositories" || !strings.Contains(r.URL.Query().Get("q"), "language:go") {
			t.Errorf("request = %s", r.URL)
		}
		w.Write([]byte(`{"items":[{"full_name":"junegunn/fzf","description":"fuzzy finder","html_url":"https://github.com/junegunn/fzf","stargazers_count":70000}]}`))
	}))
	defer srv.Close()
	reg := registry.New()
	reg.GitHubURL = srv.URL
	got, err := NewGitHub(&runner.Fake{}, reg).Search(context.Background(), "fzf")
	if err != nil {
		t.Fatal(err)
	}
	want := []Result{{Name: "github.com/junegunn/fzf", Source: "github", Version: "★ 70000", Description: "fuzzy finder"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestGitHubInstall(t *testing.T) {
	f := &runner.Fake{Responses: map[string]runner.Result{
		"go install github.com/junegunn/fzf@latest": {},
	}}
	p := NewGitHub(f, nil)
	if err := p.Install(context.Background(), "github.com/junegunn/fzf", &strings.Builder{}); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"gitlab.com/a/b", "-x", "junegunn/fzf"} {
		if err := p.Install(context.Background(), bad, &strings.Builder{}); err == nil {
			t.Errorf("Install(%q) succeeded", bad)
		}
	}
	if err := p.Remove(context.Background(), "x", nil); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Remove err = %v", err)
	}
}

func TestGitHubAvailableNeedsGo(t *testing.T) {
	if NewGitHub(&runner.Fake{Missing: map[string]bool{"go": true}}, nil).Available() {
		t.Error("Available without go")
	}
}
