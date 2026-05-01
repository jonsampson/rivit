package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jonsampson/rivit/internal/domain"
)

var (
	ErrWorkspaceNameRequired = errors.New("workspace name is required")
	ErrWorkspacePathRequired = errors.New("workspace path is required")
	ErrWorkspaceExists       = errors.New("workspace already exists")
)

type workspaceConfigStore interface {
	Load(context.Context) (domain.Config, error)
	Save(context.Context, domain.Config) error
}

type AddWorkspaceInput struct {
	Name string
	Path string
}

type AddWorkspace struct {
	store workspaceConfigStore
}

func NewAddWorkspace(store workspaceConfigStore) AddWorkspace {
	return AddWorkspace{store: store}
}

func (u AddWorkspace) Execute(ctx context.Context, input AddWorkspaceInput) error {
	name := strings.TrimSpace(input.Name)
	path := strings.TrimSpace(input.Path)

	if name == "" {
		return ErrWorkspaceNameRequired
	}
	if path == "" {
		return ErrWorkspacePathRequired
	}

	cfg, err := u.store.Load(ctx)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.Workspaces == nil {
		cfg.Workspaces = map[string]domain.Workspace{}
	}

	if _, exists := cfg.Workspaces[name]; exists {
		return fmt.Errorf("%w: %s", ErrWorkspaceExists, name)
	}

	cfg.Workspaces[name] = domain.Workspace{Path: path}

	if err := u.store.Save(ctx, cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	return nil
}
