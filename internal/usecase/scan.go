package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jonsampson/rivit/internal/domain"
)

var ErrScanPathRequired = errors.New("scan path is required")

type scanConfigStore interface {
	Load(context.Context) (domain.Config, error)
	Save(context.Context, domain.Config) error
}

type repositoryDiscoverer interface {
	Discover(context.Context, string) ([]domain.Repository, error)
}

type ScanInput struct {
	Path      string
	Workspace string
	DryRun    bool
}

type ScanOutput struct {
	Discovered int
	Added      int
	Skipped    int
}

type Scan struct {
	store      scanConfigStore
	discoverer repositoryDiscoverer
}

func NewScan(store scanConfigStore, discoverer repositoryDiscoverer) Scan {
	return Scan{store: store, discoverer: discoverer}
}

func (u Scan) Execute(ctx context.Context, input ScanInput) (ScanOutput, error) {
	path := strings.TrimSpace(input.Path)
	workspaceName := strings.TrimSpace(input.Workspace)

	if path == "" {
		return ScanOutput{}, ErrScanPathRequired
	}
	if workspaceName == "" {
		return ScanOutput{}, ErrRepositoryWorkspaceReq
	}

	cfg, err := u.store.Load(ctx)
	if err != nil {
		return ScanOutput{}, fmt.Errorf("load config: %w", err)
	}

	ws, ok := cfg.Workspaces[workspaceName]
	if !ok {
		return ScanOutput{}, fmt.Errorf("%w: %s", ErrWorkspaceNotFound, workspaceName)
	}

	found, err := u.discoverer.Discover(ctx, path)
	if err != nil {
		return ScanOutput{}, fmt.Errorf("discover repositories: %w", err)
	}

	knownInWorkspace := map[string]struct{}{}
	for _, repo := range ws.Repos {
		knownInWorkspace[repo.URL] = struct{}{}
	}

	added := 0
	skipped := 0
	for _, repo := range found {
		repoURL := strings.TrimSpace(repo.URL)
		repoID, err := domain.RepoIDFromRemoteURL(repoURL)
		if err != nil {
			skipped++
			continue
		}

		if _, exists := knownInWorkspace[repoURL]; exists {
			skipped++
			continue
		}

		knownInWorkspace[repoURL] = struct{}{}
		ws.Repos = append(ws.Repos, domain.Repository{
			URL: repoURL,
			Secret: &domain.Secret{
				Source: repoID + ".env.sops",
				Target: ".env",
			},
		})
		added++
	}

	if !input.DryRun {
		cfg.Workspaces[workspaceName] = ws
		if err := u.store.Save(ctx, cfg); err != nil {
			return ScanOutput{}, fmt.Errorf("save config: %w", err)
		}
	}

	return ScanOutput{Discovered: len(found), Added: added, Skipped: skipped}, nil
}
