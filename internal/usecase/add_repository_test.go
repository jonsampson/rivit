package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/jonsampson/rivit/internal/domain"
)

func TestAddRepositoryExecute(t *testing.T) {
	t.Run("adds repo to workspace and catalog", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"personal": {Path: "~/Code"}}}}
		uc := NewAddRepository(store)

		repoID, err := uc.Execute(context.Background(), AddRepositoryInput{
			URL:       "git@github.com:jonsampson/rivit.git",
			Workspace: "personal",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if repoID != "github.com/jonsampson/rivit" {
			t.Fatalf("unexpected repo id: %s", repoID)
		}

		ws := store.config.Workspaces["personal"]
		if len(ws.Repos) != 1 || ws.Repos[0] != repoID {
			t.Fatalf("workspace repo not linked: %+v", ws.Repos)
		}

		repo := store.config.Repos[repoID]
		if repo.URL != "git@github.com:jonsampson/rivit.git" {
			t.Fatalf("repo catalog not saved: %+v", repo)
		}
	})

	t.Run("fails when workspace missing", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{}}}
		uc := NewAddRepository(store)

		_, err := uc.Execute(context.Background(), AddRepositoryInput{URL: "git@github.com:jonsampson/rivit.git", Workspace: "personal"})
		if !errors.Is(err, ErrWorkspaceNotFound) {
			t.Fatalf("expected ErrWorkspaceNotFound, got %v", err)
		}
	})

	t.Run("url required", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"personal": {Path: "~/Code"}}}}
		uc := NewAddRepository(store)
		_, err := uc.Execute(context.Background(), AddRepositoryInput{URL: " ", Workspace: "personal"})
		if !errors.Is(err, ErrRepositoryURLRequired) {
			t.Fatalf("expected ErrRepositoryURLRequired, got %v", err)
		}
	})

	t.Run("workspace required", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1}}
		uc := NewAddRepository(store)
		_, err := uc.Execute(context.Background(), AddRepositoryInput{URL: "git@github.com:org/one.git", Workspace: " "})
		if !errors.Is(err, ErrRepositoryWorkspaceReq) {
			t.Fatalf("expected ErrRepositoryWorkspaceReq, got %v", err)
		}
	})

	t.Run("duplicate in workspace", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"personal": {Path: "~/Code", Repos: []string{"github.com/jonsampson/rivit"}}}}}
		uc := NewAddRepository(store)
		_, err := uc.Execute(context.Background(), AddRepositoryInput{URL: "git@github.com:jonsampson/rivit.git", Workspace: "personal"})
		if !errors.Is(err, ErrRepositoryExists) {
			t.Fatalf("expected ErrRepositoryExists, got %v", err)
		}
	})

	t.Run("load error", func(t *testing.T) {
		store := &memoryConfigStore{loadErr: errors.New("boom")}
		uc := NewAddRepository(store)
		_, err := uc.Execute(context.Background(), AddRepositoryInput{URL: "git@github.com:org/one.git", Workspace: "personal"})
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("save error", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"personal": {Path: "~/Code"}}}, saveErr: errors.New("boom")}
		uc := NewAddRepository(store)
		_, err := uc.Execute(context.Background(), AddRepositoryInput{URL: "git@github.com:org/one.git", Workspace: "personal"})
		if err == nil {
			t.Fatalf("expected error")
		}
	})
}
