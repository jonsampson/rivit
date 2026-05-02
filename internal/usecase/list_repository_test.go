package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/jonsampson/rivit/internal/domain"
)

func TestListRepositoryExecute(t *testing.T) {
	store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{
		"work": {Path: "~/work", Repos: []domain.Repository{{URL: "git@github.com:org/zeta.git"}}},
		"home": {Path: "~/home", Repos: []domain.Repository{{URL: "git@github.com:org/alpha.git"}}},
	}}}
	uc := NewListRepository(store)

	items, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	if items[0].Workspace != "home" || items[0].URL != "git@github.com:org/alpha.git" {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
	if items[1].Workspace != "work" || items[1].URL != "git@github.com:org/zeta.git" {
		t.Fatalf("unexpected second item: %+v", items[1])
	}

	t.Run("load error", func(t *testing.T) {
		store := &memoryConfigStore{loadErr: errors.New("boom")}
		uc := NewListRepository(store)
		if _, err := uc.Execute(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})
}
