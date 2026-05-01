package usecase

import (
	"context"
	"testing"

	"github.com/jonsampson/rivit/internal/domain"
)

type fakeDiscoverer struct {
	repos []domain.Repository
	err   error
}

func (d fakeDiscoverer) Discover(context.Context, string) ([]domain.Repository, error) {
	return d.repos, d.err
}

func TestScanExecute(t *testing.T) {
	t.Run("adds discovered repositories", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"personal": {Path: "~/Code"}}}}
		discoverer := fakeDiscoverer{repos: []domain.Repository{{URL: "git@github.com:org/one.git"}, {URL: "git@github.com:org/two.git"}}}
		uc := NewScan(store, discoverer)

		out, err := uc.Execute(context.Background(), ScanInput{Path: "~/dev", Workspace: "personal"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if out.Added != 2 || out.Discovered != 2 {
			t.Fatalf("unexpected output: %+v", out)
		}

		ws := store.config.Workspaces["personal"]
		if len(ws.Repos) != 2 {
			t.Fatalf("expected 2 workspace repos, got %d", len(ws.Repos))
		}

		repo := store.config.Repos["github.com/org/one"]
		if repo.Secret == nil || repo.Secret.Source != "github.com/org/one.env.sops" {
			t.Fatalf("expected default secret metadata, got %+v", repo.Secret)
		}
	})

	t.Run("dry run does not save", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"personal": {Path: "~/Code"}}}}
		discoverer := fakeDiscoverer{repos: []domain.Repository{{URL: "git@github.com:org/one.git"}}}
		uc := NewScan(store, discoverer)

		out, err := uc.Execute(context.Background(), ScanInput{Path: "~/dev", Workspace: "personal", DryRun: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Added != 1 {
			t.Fatalf("expected one addition, got %+v", out)
		}

		if len(store.config.Workspaces["personal"].Repos) != 0 {
			t.Fatalf("dry run should not persist workspace changes")
		}
		if len(store.config.Repos) != 0 {
			t.Fatalf("dry run should not persist repo catalog changes")
		}
	})
}
