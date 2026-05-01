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

	t.Run("removes orphaned repos from catalog", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{
			Version: 1,
			Workspaces: map[string]domain.Workspace{
				"personal": {Path: "~/Code", Repos: []string{"github.com/org/one", "github.com/org/shared"}},
				"work":     {Path: "~/Work", Repos: []string{"github.com/org/shared"}},
			},
			Repos: map[string]domain.Repository{
				"github.com/org/one":    {URL: "git@github.com:org/one.git"},
				"github.com/org/shared": {URL: "git@github.com:org/shared.git"},
			},
		}}
		uc := NewRemoveWorkspace(store)

		err := uc.Execute(context.Background(), RemoveWorkspaceInput{Name: "personal"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, exists := store.config.Repos["github.com/org/one"]; exists {
			t.Fatalf("orphaned repo still present in catalog")
		}

		if _, exists := store.config.Repos["github.com/org/shared"]; !exists {
			t.Fatalf("shared repo should remain in catalog")
		}
	})
}
