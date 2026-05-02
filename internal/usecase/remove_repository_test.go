package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/jonsampson/rivit/internal/domain"
)

func TestRemoveRepositoryExecute(t *testing.T) {
	t.Run("removes repo from workspaces", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{
			Version: 1,
			Workspaces: map[string]domain.Workspace{
				"personal": {Path: "~/Code", Repos: []domain.Repository{{URL: "git@github.com:org/one.git"}, {URL: "git@github.com:org/two.git"}}},
				"work":     {Path: "~/Work", Repos: []domain.Repository{{URL: "git@github.com:org/two.git"}}},
			},
		}}

		uc := NewRemoveRepository(store)
		err := uc.Execute(context.Background(), RemoveRepositoryInput{ID: "git@github.com:org/two.git"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(store.config.Workspaces["personal"].Repos) != 1 || store.config.Workspaces["personal"].Repos[0].URL != "git@github.com:org/one.git" {
			t.Fatalf("unexpected personal repos: %+v", store.config.Workspaces["personal"].Repos)
		}

		if len(store.config.Workspaces["work"].Repos) != 0 {
			t.Fatalf("unexpected work repos: %+v", store.config.Workspaces["work"].Repos)
		}
	})

	t.Run("repo not found", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1}}
		uc := NewRemoveRepository(store)

		err := uc.Execute(context.Background(), RemoveRepositoryInput{ID: "git@github.com:org/missing.git"})
		if !errors.Is(err, ErrRepositoryNotFound) {
			t.Fatalf("expected ErrRepositoryNotFound, got %v", err)
		}
	})

	t.Run("repo id required", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1}}
		uc := NewRemoveRepository(store)
		err := uc.Execute(context.Background(), RemoveRepositoryInput{ID: " "})
		if !errors.Is(err, ErrRepositoryIDRequired) {
			t.Fatalf("expected ErrRepositoryIDRequired, got %v", err)
		}
	})

	t.Run("load error", func(t *testing.T) {
		store := &memoryConfigStore{loadErr: errors.New("boom")}
		uc := NewRemoveRepository(store)
		err := uc.Execute(context.Background(), RemoveRepositoryInput{ID: "github.com/org/repo"})
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("save error", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"personal": {Path: "~/Code", Repos: []domain.Repository{{URL: "git@github.com:org/repo.git"}}}}}, saveErr: errors.New("boom")}
		uc := NewRemoveRepository(store)
		err := uc.Execute(context.Background(), RemoveRepositoryInput{ID: "git@github.com:org/repo.git"})
		if err == nil {
			t.Fatalf("expected error")
		}
	})
}
