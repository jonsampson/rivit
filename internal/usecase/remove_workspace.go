package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jonsampson/rivit/internal/domain"
)

var (
	ErrRemoveWorkspaceNameRequired = errors.New("workspace name is required")
	ErrWorkspaceNotFound           = errors.New("workspace not found")
)

type removeWorkspaceConfigStore interface {
	Load(context.Context) (domain.Config, error)
	Save(context.Context, domain.Config) error
}

type RemoveWorkspaceInput struct {
	Name string
}

type RemoveWorkspace struct {
	store removeWorkspaceConfigStore
}

func NewRemoveWorkspace(store removeWorkspaceConfigStore) RemoveWorkspace {
	return RemoveWorkspace{store: store}
}

func (u RemoveWorkspace) Execute(ctx context.Context, input RemoveWorkspaceInput) error {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return ErrRemoveWorkspaceNameRequired
	}

	cfg, err := u.store.Load(ctx)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.Workspaces) == 0 {
		return fmt.Errorf("%w: %s", ErrWorkspaceNotFound, name)
	}

	if _, ok := cfg.Workspaces[name]; !ok {
		return fmt.Errorf("%w: %s", ErrWorkspaceNotFound, name)
	}

	delete(cfg.Workspaces, name)

	if err := u.store.Save(ctx, cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	return nil
}
