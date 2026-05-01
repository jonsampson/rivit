package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jonsampson/rivit/internal/domain"
)

var (
	ErrRepositoryURLRequired  = errors.New("repository url is required")
	ErrRepositoryWorkspaceReq = errors.New("workspace is required")
	ErrRepositoryExists       = errors.New("repository already exists")
)

type repositoryConfigStore interface {
	Load(context.Context) (domain.Config, error)
	Save(context.Context, domain.Config) error
}

type AddRepositoryInput struct {
	URL       string
	Workspace string
}

type AddRepository struct {
	store repositoryConfigStore
}

func NewAddRepository(store repositoryConfigStore) AddRepository {
	return AddRepository{store: store}
}

func (u AddRepository) Execute(ctx context.Context, input AddRepositoryInput) (string, error) {
	repoURL := strings.TrimSpace(input.URL)
	workspaceName := strings.TrimSpace(input.Workspace)

	if repoURL == "" {
		return "", ErrRepositoryURLRequired
	}
	if workspaceName == "" {
		return "", ErrRepositoryWorkspaceReq
	}

	repoID, err := domain.RepoIDFromRemoteURL(repoURL)
	if err != nil {
		return "", err
	}

	cfg, err := u.store.Load(ctx)
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}

	ws, ok := cfg.Workspaces[workspaceName]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrWorkspaceNotFound, workspaceName)
	}

	for _, existing := range ws.Repos {
		if existing == repoID {
			return "", fmt.Errorf("%w: %s already in workspace %s", ErrRepositoryExists, repoID, workspaceName)
		}
	}

	ws.Repos = append(ws.Repos, repoID)
	cfg.Workspaces[workspaceName] = ws

	if cfg.Repos == nil {
		cfg.Repos = map[string]domain.Repository{}
	}
	if _, exists := cfg.Repos[repoID]; !exists {
		cfg.Repos[repoID] = domain.Repository{URL: repoURL}
	}

	if err := u.store.Save(ctx, cfg); err != nil {
		return "", fmt.Errorf("save config: %w", err)
	}

	return repoID, nil
}
