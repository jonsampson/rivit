package usecase

import (
	"context"
	"testing"

	"github.com/jonsampson/rivit/internal/domain"
)

func TestListRepositoryExecute(t *testing.T) {
	store := &memoryConfigStore{config: domain.Config{Version: 1, Repos: map[string]domain.Repository{
		"github.com/org/zeta":  {URL: "git@github.com:org/zeta.git"},
		"github.com/org/alpha": {URL: "git@github.com:org/alpha.git"},
	}}}
	uc := NewListRepository(store)

	items, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	if items[0].ID != "github.com/org/alpha" {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
	if items[1].ID != "github.com/org/zeta" {
		t.Fatalf("unexpected second item: %+v", items[1])
	}
}
