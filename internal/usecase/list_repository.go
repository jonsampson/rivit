package usecase

import (
	"context"
	"fmt"
	"sort"

	"github.com/jonsampson/rivit/internal/domain"
)

type repositoryReader interface {
	Load(context.Context) (domain.Config, error)
}

type ListedRepository struct {
	Workspace string
	URL       string
}

type ListRepository struct {
	store repositoryReader
}

func NewListRepository(store repositoryReader) ListRepository {
	return ListRepository{store: store}
}

func (u ListRepository) Execute(ctx context.Context) ([]ListedRepository, error) {
	cfg, err := u.store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	items := []ListedRepository{}
	for workspaceName, ws := range cfg.Workspaces {
		for _, repo := range ws.Repos {
			items = append(items, ListedRepository{Workspace: workspaceName, URL: repo.URL})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Workspace == items[j].Workspace {
			return items[i].URL < items[j].URL
		}
		return items[i].Workspace < items[j].Workspace
	})

	return items, nil
}
