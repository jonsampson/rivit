package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jonsampson/rivit/internal/domain"
)

var (
	ErrRepositoryIDRequired = errors.New("repository id is required")
	ErrRepositoryNotFound   = errors.New("repository not found")
)

type removeRepositoryConfigStore interface {
	Load(context.Context) (domain.Config, error)
	Save(context.Context, domain.Config) error
}

type RemoveRepositoryInput struct {
	ID string
}

type RemoveRepository struct {
	store removeRepositoryConfigStore
}

func NewRemoveRepository(store removeRepositoryConfigStore) RemoveRepository {
	return RemoveRepository{store: store}
}

func (u RemoveRepository) Execute(ctx context.Context, input RemoveRepositoryInput) error {
	repoID := strings.TrimSpace(input.ID)
	if repoID == "" {
		return ErrRepositoryIDRequired
	}

	cfg, err := u.store.Load(ctx)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if _, ok := cfg.Repos[repoID]; !ok {
		return fmt.Errorf("%w: %s", ErrRepositoryNotFound, repoID)
	}

	delete(cfg.Repos, repoID)

	for name, ws := range cfg.Workspaces {
		filtered := make([]string, 0, len(ws.Repos))
		for _, id := range ws.Repos {
			if id != repoID {
				filtered = append(filtered, id)
			}
		}
		ws.Repos = filtered
		cfg.Workspaces[name] = ws
	}

	if err := u.store.Save(ctx, cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	return nil
}
