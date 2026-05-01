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
	ID  string
	URL string
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

	items := make([]ListedRepository, 0, len(cfg.Repos))
	for id, repo := range cfg.Repos {
		items = append(items, ListedRepository{ID: id, URL: repo.URL})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	return items, nil
}
