package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/jonsampson/rivit/internal/domain"
)

func TestAddWorkspaceExecute(t *testing.T) {
	t.Run("adds workspace", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1}}
		uc := NewAddWorkspace(store)

		err := uc.Execute(context.Background(), AddWorkspaceInput{Name: "personal", Path: "~/Code"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, ok := store.config.Workspaces["personal"]
		if !ok {
			t.Fatalf("workspace not saved")
		}
	})

	t.Run("duplicate workspace", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"personal": {Path: "~/Code"}}}}
		uc := NewAddWorkspace(store)

		err := uc.Execute(context.Background(), AddWorkspaceInput{Name: "personal", Path: "~/Code"})
		if !errors.Is(err, ErrWorkspaceExists) {
			t.Fatalf("expected ErrWorkspaceExists, got %v", err)
		}
	})
}
