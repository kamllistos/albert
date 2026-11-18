package provider

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/kamllistos/albert/internal/runner"
)

func aptFake() *runner.Fake {
	return &runner.Fake{Responses: map[string]runner.Result{
		"apt-mark showmanual": {Stdout: []byte("curl\nvim\nlibfoo:i386\n")},
		"dpkg-query -W -f=${Package}\t${Version}\n curl vim libfoo:i386": {
			Stdout: []byte("curl\t8.5.0-2\nvim\t2:9.1\nlibfoo\t1.0\n")},
		"apt list --upgradable": {
			Stdout: []byte("Listing...\ncurl/noble-updates 8.5.0-2ubuntu10.6 amd64 [upgradable from: 8.5.0-2ubuntu10.5]\nopenssl/noble-updates 3.0.13 amd64 [upgradable from: 3.0.12]\n"),
			Stderr: []byte("WARNING: apt does not have a stable CLI interface.\n")},
	}}
}

func TestAptList(t *testing.T) {
	tools, err := NewAPT(aptFake()).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []Tool{
		{Name: "curl", Source: "apt", Version: "8.5.0-2"},
		{Name: "vim", Source: "apt", Version: "2:9.1"},
		{Name: "libfoo", Source: "apt", Version: "1.0"},
	}
	if !reflect.DeepEqual(tools, want) {
		t.Errorf("List = %+v, want %+v", tools, want)
	}
}

func TestAptOutdatedOnlyManual(t *testing.T) {
	ups, err := NewAPT(aptFake()).Outdated(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []Update{{Name: "curl", Latest: "8.5.0-2ubuntu10.6"}}
	if !reflect.DeepEqual(ups, want) {
		t.Errorf("Outdated = %+v, want %+v", ups, want)
	}
}

func TestAptCommands(t *testing.T) {
	tests := []struct {
		name string
		run  func(p Provider) error
		want string
	}{
		{"upgrade all", func(p Provider) error { return p.Upgrade(context.Background(), nil, &bytes.Buffer{}) }, "sudo apt-get upgrade -y"},
		{"upgrade some", func(p Provider) error { return p.Upgrade(context.Background(), []string{"curl"}, &bytes.Buffer{}) }, "sudo apt-get install --only-upgrade -y curl"},
		{"install", func(p Provider) error { return p.Install(context.Background(), "vim", &bytes.Buffer{}) }, "sudo apt-get install -y vim"},
		{"remove", func(p Provider) error { return p.Remove(context.Background(), "vim", &bytes.Buffer{}) }, "sudo apt-get remove -y vim"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &runner.Fake{Responses: map[string]runner.Result{tt.want: {}}}
			if err := tt.run(NewAPT(f)); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(f.Calls, []string{tt.want}) {
				t.Errorf("Calls = %q", f.Calls)
			}
		})
	}
}

func TestAptRejectsBadNames(t *testing.T) {
	f := &runner.Fake{}
	p := NewAPT(f)
	if err := p.Install(context.Background(), "-rf", &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "invalid name") {
		t.Errorf("Install(-rf) err = %v", err)
	}
	if err := p.Upgrade(context.Background(), []string{"ok", "--x"}, &bytes.Buffer{}); err == nil {
		t.Error("Upgrade accepted --x")
	}
	if len(f.Calls) != 0 {
		t.Errorf("ran %q", f.Calls)
	}
}

func TestAptSearch(t *testing.T) {
	f := &runner.Fake{Responses: map[string]runner.Result{
		"apt-cache search --names-only vim": {Stdout: []byte("vim - Vi IMproved\nvim-tiny - compact vim\n")},
	}}
	rs, err := NewAPT(f).Search(context.Background(), "vim")
	if err != nil || len(rs) != 2 || rs[0].Name != "vim" || rs[0].Description != "Vi IMproved" {
		t.Errorf("Search = %+v, %v", rs, err)
	}
	if rs, _ := NewAPT(f).Search(context.Background(), "-x"); rs != nil {
		t.Errorf("Search(-x) = %+v", rs)
	}
}
