package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/jonsampson/rivit/internal/domain"
)

func TestRemoveWorkspaceExecute(t *testing.T) {
	t.Run("removes existing workspace", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"personal": {Path: "~/Code"}}}}
		uc := NewRemoveWorkspace(store)

		err := uc.Execute(context.Background(), RemoveWorkspaceInput{Name: "personal"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, exists := store.config.Workspaces["personal"]; exists {
			t.Fatalf("workspace still exists after remove")
		}
	})

	t.Run("workspace not found", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"work": {Path: "~/Work"}}}}
		uc := NewRemoveWorkspace(store)

		err := uc.Execute(context.Background(), RemoveWorkspaceInput{Name: "personal"})
		if !errors.Is(err, ErrWorkspaceNotFound) {
			t.Fatalf("expected ErrWorkspaceNotFound, got %v", err)
		}
	})

	t.Run("workspace name required", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1}}
		uc := NewRemoveWorkspace(store)
		err := uc.Execute(context.Background(), RemoveWorkspaceInput{Name: " "})
		if !errors.Is(err, ErrRemoveWorkspaceNameRequired) {
			t.Fatalf("expected ErrRemoveWorkspaceNameRequired, got %v", err)
		}
	})

	t.Run("load error", func(t *testing.T) {
		store := &memoryConfigStore{loadErr: errors.New("boom")}
		uc := NewRemoveWorkspace(store)
		err := uc.Execute(context.Background(), RemoveWorkspaceInput{Name: "personal"})
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("save error", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"personal": {Path: "~/Code"}}}, saveErr: errors.New("boom")}
		uc := NewRemoveWorkspace(store)
		err := uc.Execute(context.Background(), RemoveWorkspaceInput{Name: "personal"})
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("removes workspace with nested repos", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{
			Version: 1,
			Workspaces: map[string]domain.Workspace{
				"personal": {Path: "~/Code", Repos: []domain.Repository{{URL: "git@github.com:org/one.git"}, {URL: "git@github.com:org/shared.git"}}},
				"work":     {Path: "~/Work", Repos: []domain.Repository{{URL: "git@github.com:org/shared.git"}}},
			},
		}}
		uc := NewRemoveWorkspace(store)

		err := uc.Execute(context.Background(), RemoveWorkspaceInput{Name: "personal"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, exists := store.config.Workspaces["work"]; !exists {
			t.Fatalf("expected other workspace to remain")
		}
	})
}
