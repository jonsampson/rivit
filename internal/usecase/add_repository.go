package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/url"
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

	repoID, err := repoIDFromRemoteURL(repoURL)
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

func repoIDFromRemoteURL(remoteURL string) (string, error) {
	if strings.HasPrefix(remoteURL, "git@") {
		parts := strings.SplitN(strings.TrimPrefix(remoteURL, "git@"), ":", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid ssh repository url: %s", remoteURL)
		}
		host := strings.TrimSpace(parts[0])
		path := normalizeRemotePath(parts[1])
		if host == "" || path == "" {
			return "", fmt.Errorf("invalid ssh repository url: %s", remoteURL)
		}
		return host + "/" + path, nil
	}

	u, err := url.Parse(remoteURL)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("invalid repository url: %s", remoteURL)
	}

	path := normalizeRemotePath(strings.TrimPrefix(u.Path, "/"))
	if strings.EqualFold(u.Host, "dev.azure.com") {
		path = strings.Replace(path, "/_git/", "/", 1)
	}
	if path == "" {
		return "", fmt.Errorf("invalid repository url: %s", remoteURL)
	}

	return u.Host + "/" + path, nil
}

func normalizeRemotePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "/")
	path = strings.TrimSuffix(path, ".git")
	return path
}
