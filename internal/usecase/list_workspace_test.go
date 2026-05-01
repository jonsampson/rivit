package usecase

import (
	"context"
	"testing"

	"github.com/jonsampson/rivit/internal/domain"
)

func TestListWorkspaceExecute(t *testing.T) {
	t.Run("lists workspaces sorted by name", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{
			"work":     {Path: "~/Work"},
			"personal": {Path: "~/Code"},
		}}}
		uc := NewListWorkspace(store)

		items, err := uc.Execute(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(items))
		}

		if items[0].Name != "personal" || items[0].Path != "~/Code" {
			t.Fatalf("unexpected first item: %+v", items[0])
		}

		if items[1].Name != "work" || items[1].Path != "~/Work" {
			t.Fatalf("unexpected second item: %+v", items[1])
		}
	})

	t.Run("returns empty list when no workspaces", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1}}
		uc := NewListWorkspace(store)

		items, err := uc.Execute(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(items) != 0 {
			t.Fatalf("expected no items, got %d", len(items))
		}
	})
}
