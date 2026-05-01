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
}
