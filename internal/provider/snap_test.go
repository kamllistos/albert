package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/kamllistos/albert/internal/runner"
)

func TestSnapList(t *testing.T) {
	f := &runner.Fake{Responses: map[string]runner.Result{"snap list": {Stdout: []byte(
		"Name     Version   Rev    Tracking       Publisher   Notes\n" +
			"core22   20240111  1033   latest/stable  canonical** base\n" +
			"snapd    2.61      20290  latest/stable  canonical** snapd\n" +
			"firefox  125.0     4000   latest/stable  mozilla**   -\n")}}}
	tools, err := NewSnap(f).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []Tool{
		{Name: "core22", Source: "snap", Version: "20240111", Dependency: true},
		{Name: "snapd", Source: "snap", Version: "2.61", Dependency: true},
		{Name: "firefox", Source: "snap", Version: "125.0"},
	}
	if !reflect.DeepEqual(tools, want) {
		t.Errorf("List = %+v, want %+v", tools, want)
	}
}

func TestSnapOutdated(t *testing.T) {
	tests := []struct {
		name string
		res  runner.Result
		want []Update
	}{
		{"updates", runner.Result{Stdout: []byte("Name     Version  Rev   Size  Publisher  Notes\nfirefox  126.0    4100  90MB  mozilla**  -\n")},
			[]Update{{Name: "firefox", Latest: "126.0"}}},
		{"up to date on stderr", runner.Result{Stderr: []byte("All snaps up to date.\n")}, nil},
		{"up to date on stdout", runner.Result{Stdout: []byte("All snaps up to date.\n")}, nil},
		{"empty", runner.Result{}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &runner.Fake{Responses: map[string]runner.Result{"snap refresh --list": tt.res}}
			ups, err := NewSnap(f).Outdated(context.Background())
			if err != nil || !reflect.DeepEqual(ups, tt.want) {
				t.Errorf("Outdated = %+v, %v; want %+v", ups, err, tt.want)
			}
		})
	}
}

func TestSnapInstallRejectsFlag(t *testing.T) {
	f := &runner.Fake{}
	if err := NewSnap(f).Install(context.Background(), "-x", nil); err == nil || len(f.Calls) != 0 {
		t.Errorf("err = %v, calls = %q", err, f.Calls)
	}
}
