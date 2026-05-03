package usecase

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jonsampson/rivit/internal/domain"
)

var ErrScanPathRequired = errors.New("scan path is required")

type scanConfigStore interface {
	Load(context.Context) (domain.Config, error)
	Save(context.Context, domain.Config) error
}

type repositoryDiscoverer interface {
	Discover(context.Context, string, func(repoPath string, remoteURL string) error) error
}

type scanPathOps interface {
	PathExists(context.Context, string) (bool, error)
}

type scanSecretOps interface {
	EncryptFile(context.Context, string, string) error
}

type ScanInput struct {
	Path      string
	Workspace string
	DryRun    bool
}

type ScanOutput struct {
	Discovered int
	Added      int
	Absorbed   int
	Skipped    int
	SkipReasons map[string]int
	Failures   []ScanFailure
}

type ScanFailure struct {
	RepositoryURL string
	Step          string
	Message       string
}

type Scan struct {
	store      scanConfigStore
	discoverer repositoryDiscoverer
	paths      scanPathOps
	secrets    scanSecretOps
}

func NewScan(store scanConfigStore, discoverer repositoryDiscoverer, paths scanPathOps, secrets scanSecretOps) Scan {
	return Scan{store: store, discoverer: discoverer, paths: paths, secrets: secrets}
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

	knownInWorkspace := map[string]struct{}{}
	for _, repo := range ws.Repos {
		knownInWorkspace[repo.URL] = struct{}{}
	}

	added := 0
	skipped := 0
	absorbed := 0
	skipReasons := map[string]int{}
	failures := []ScanFailure{}
	discovered := 0
	err = u.discoverer.Discover(ctx, path, func(repoPath string, remoteURL string) error {
		discovered++
		repoURL := strings.TrimSpace(remoteURL)
		repoID, err := domain.RepoIDFromRemoteURL(repoURL)
		if err != nil {
			skipped++
			skipReasons["invalid_remote"]++
			failures = append(failures, ScanFailure{RepositoryURL: repoURL, Step: "normalize", Message: err.Error()})
			return nil
		}

		if _, exists := knownInWorkspace[repoURL]; exists {
			skipped++
			skipReasons["already_tracked"]++
			return nil
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

		envPath := filepath.Join(repoPath, ".env")
		envExists, err := u.paths.PathExists(ctx, envPath)
		if err != nil {
			return fmt.Errorf("check env file: %w", err)
		}
		if envExists {
			secretPath := filepath.Join(cfg.Secrets.Path, repoID+".env.sops")
			secretExists, err := u.paths.PathExists(ctx, secretPath)
			if err != nil {
				return fmt.Errorf("check secret file: %w", err)
			}
			if !secretExists {
				if !input.DryRun {
					if err := u.secrets.EncryptFile(ctx, envPath, secretPath); err != nil {
						skipped++
						skipReasons["absorb_failed"]++
						failures = append(failures, ScanFailure{RepositoryURL: repoURL, Step: "absorb", Message: err.Error()})
						return nil
					}
				}
				absorbed++
			}
		}

		return nil
	})
	if err != nil {
		return ScanOutput{}, fmt.Errorf("discover repositories: %w", err)
	}
	if !input.DryRun {
		cfg.Workspaces[workspaceName] = ws
		if err := u.store.Save(ctx, cfg); err != nil {
			return ScanOutput{}, fmt.Errorf("save config: %w", err)
		}
	}

	out := ScanOutput{Discovered: discovered, Added: added, Absorbed: absorbed, Skipped: skipped, SkipReasons: skipReasons, Failures: failures}
	return out, nil
}
