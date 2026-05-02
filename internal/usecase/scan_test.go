package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/jonsampson/rivit/internal/domain"
)

type fakeDiscoverer struct {
	repos []domain.DiscoveredRepository
	err   error
}

func (d fakeDiscoverer) Discover(context.Context, string) ([]domain.DiscoveredRepository, error) {
	return d.repos, d.err
}

func TestScanExecute(t *testing.T) {
	t.Run("adds discovered repositories", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"personal": {Path: "~/Code"}}}}
		discoverer := fakeDiscoverer{repos: []domain.DiscoveredRepository{{Path: "/existing/one", URL: "git@github.com:org/one.git"}, {Path: "/existing/two", URL: "git@github.com:org/two.git"}}}
		paths := &memoryPathOps{exists: map[string]bool{"/existing/one/.env": true, "/secrets/github.com/org/one.env.sops": true}}
		uc := NewScan(store, discoverer, paths, &memoryEncryptOps{})

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

		repo := ws.Repos[0]
		if repo.Secret == nil || repo.Secret.Source != "github.com/org/one.env.sops" {
			t.Fatalf("expected default secret metadata, got %+v", repo.Secret)
		}
	})

	t.Run("dry run does not save", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"personal": {Path: "~/Code"}}}}
		discoverer := fakeDiscoverer{repos: []domain.DiscoveredRepository{{Path: "/existing/one", URL: "git@github.com:org/one.git"}}}
		paths := &memoryPathOps{exists: map[string]bool{"/existing/one/.env": true}}
		uc := NewScan(store, discoverer, paths, &memoryEncryptOps{})

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
	})

	t.Run("validates required input", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"personal": {Path: "~/Code"}}}}
		uc := NewScan(store, fakeDiscoverer{}, &memoryPathOps{exists: map[string]bool{}}, &memoryEncryptOps{})

		if _, err := uc.Execute(context.Background(), ScanInput{Path: " ", Workspace: "personal"}); !errors.Is(err, ErrScanPathRequired) {
			t.Fatalf("expected ErrScanPathRequired, got %v", err)
		}
		if _, err := uc.Execute(context.Background(), ScanInput{Path: "~/dev", Workspace: " "}); !errors.Is(err, ErrRepositoryWorkspaceReq) {
			t.Fatalf("expected ErrRepositoryWorkspaceReq, got %v", err)
		}
	})

	t.Run("workspace missing", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{}}}
		uc := NewScan(store, fakeDiscoverer{}, &memoryPathOps{exists: map[string]bool{}}, &memoryEncryptOps{})
		if _, err := uc.Execute(context.Background(), ScanInput{Path: "~/dev", Workspace: "personal"}); !errors.Is(err, ErrWorkspaceNotFound) {
			t.Fatalf("expected ErrWorkspaceNotFound, got %v", err)
		}
	})

	t.Run("load discover and save errors", func(t *testing.T) {
		uc := NewScan(&memoryConfigStore{loadErr: errors.New("boom")}, fakeDiscoverer{}, &memoryPathOps{exists: map[string]bool{}}, &memoryEncryptOps{})
		if _, err := uc.Execute(context.Background(), ScanInput{Path: "~/dev", Workspace: "personal"}); err == nil {
			t.Fatalf("expected load error")
		}

		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"personal": {Path: "~/Code"}}}}
		uc = NewScan(store, fakeDiscoverer{err: errors.New("boom")}, &memoryPathOps{exists: map[string]bool{}}, &memoryEncryptOps{})
		if _, err := uc.Execute(context.Background(), ScanInput{Path: "~/dev", Workspace: "personal"}); err == nil {
			t.Fatalf("expected discover error")
		}

		store = &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"personal": {Path: "~/Code"}}}, saveErr: errors.New("boom")}
		uc = NewScan(store, fakeDiscoverer{repos: []domain.DiscoveredRepository{{Path: "/existing/one", URL: "git@github.com:org/one.git"}}}, &memoryPathOps{exists: map[string]bool{}}, &memoryEncryptOps{})
		if _, err := uc.Execute(context.Background(), ScanInput{Path: "~/dev", Workspace: "personal"}); err == nil {
			t.Fatalf("expected save error")
		}
	})

	t.Run("skips invalid and existing repositories", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"personal": {Path: "~/Code", Repos: []domain.Repository{{URL: "git@github.com:org/existing.git"}}}}}}
		discoverer := fakeDiscoverer{repos: []domain.DiscoveredRepository{{Path: "/existing/repo", URL: "git@github.com:org/existing.git"}, {Path: "/existing/bad", URL: "not-a-url"}}}
		uc := NewScan(store, discoverer, &memoryPathOps{exists: map[string]bool{}}, &memoryEncryptOps{})

		out, err := uc.Execute(context.Background(), ScanInput{Path: "~/dev", Workspace: "personal"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Added != 0 || out.Skipped != 2 {
			t.Fatalf("unexpected output: %+v", out)
		}
	})

	t.Run("continues when absorb fails", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"personal": {Path: "~/Code"}}, Secrets: domain.SecretsConfig{Path: "/secrets"}}}
		discoverer := fakeDiscoverer{repos: []domain.DiscoveredRepository{{Path: "/existing/one", URL: "git@github.com:org/one.git"}}}
		paths := &memoryPathOps{exists: map[string]bool{"/existing/one/.env": true}}
		uc := NewScan(store, discoverer, paths, &memoryEncryptOps{err: errors.New("boom")})

		out, err := uc.Execute(context.Background(), ScanInput{Path: "~/dev", Workspace: "personal"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Added != 1 || out.Absorbed != 0 || out.Skipped != 1 {
			t.Fatalf("unexpected output: %+v", out)
		}
		if len(out.Failures) != 1 || out.Failures[0].Step != "absorb" {
			t.Fatalf("expected absorb failure details, got %+v", out.Failures)
		}
	})
}
