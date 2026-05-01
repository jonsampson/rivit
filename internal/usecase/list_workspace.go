package usecase

import (
	"context"
	"fmt"
	"sort"

	"github.com/jonsampson/rivit/internal/domain"
)

type workspaceReader interface {
	Load(context.Context) (domain.Config, error)
}

type ListedWorkspace struct {
	Name string
	Path string
}

type ListWorkspace struct {
	store workspaceReader
}

func NewListWorkspace(store workspaceReader) ListWorkspace {
	return ListWorkspace{store: store}
}

func (u ListWorkspace) Execute(ctx context.Context) ([]ListedWorkspace, error) {
	cfg, err := u.store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	items := make([]ListedWorkspace, 0, len(cfg.Workspaces))
	for name, ws := range cfg.Workspaces {
		items = append(items, ListedWorkspace{Name: name, Path: ws.Path})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items, nil
}
