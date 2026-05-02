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

	cfg, err := u.store.Load(ctx)
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}

	ws, ok := cfg.Workspaces[workspaceName]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrWorkspaceNotFound, workspaceName)
	}

	for _, existing := range ws.Repos {
		if existing.URL == repoURL {
			return "", fmt.Errorf("%w: %s already in workspace %s", ErrRepositoryExists, repoURL, workspaceName)
		}
	}

	repoID, err := domain.RepoIDFromRemoteURL(repoURL)
	if err != nil {
		return "", err
	}

	ws.Repos = append(ws.Repos, domain.Repository{
		URL: repoURL,
		Secret: &domain.Secret{
			Source: repoID + ".env.sops",
			Target: ".env",
		},
	})
	cfg.Workspaces[workspaceName] = ws

	if err := u.store.Save(ctx, cfg); err != nil {
		return "", fmt.Errorf("save config: %w", err)
	}

	return repoURL, nil
}
