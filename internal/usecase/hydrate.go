package usecase

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jonsampson/rivit/internal/domain"
)

var ErrHydrateTargetNotFound = errors.New("hydrate target not found")

type hydrateConfigReader interface {
	Load(context.Context) (domain.Config, error)
}

type hydratePathOps interface {
	PathExists(context.Context, string) (bool, error)
	MkdirAll(context.Context, string) error
}

type hydrateGitOps interface {
	Clone(context.Context, string, string) error
}

type hydrateSecretOps interface {
	DecryptFile(context.Context, string, string) error
}

type HydrateInput struct {
	Target      string
	DryRun      bool
	ReposOnly   bool
	SecretsOnly bool
	ForceEnv    bool
}

type HydrateOutput struct {
	DirectoriesCreated  int
	ReposCloned         int
	SecretsMaterialized int
	Skipped             int
}

type Hydrate struct {
	store   hydrateConfigReader
	paths   hydratePathOps
	git     hydrateGitOps
	secrets hydrateSecretOps
}

func NewHydrate(store hydrateConfigReader, paths hydratePathOps, git hydrateGitOps, secrets hydrateSecretOps) Hydrate {
	return Hydrate{store: store, paths: paths, git: git, secrets: secrets}
}

func (u Hydrate) Execute(ctx context.Context, input HydrateInput) (HydrateOutput, error) {
	cfg, err := u.store.Load(ctx)
	if err != nil {
		return HydrateOutput{}, fmt.Errorf("load config: %w", err)
	}

	if input.ReposOnly && input.SecretsOnly {
		return HydrateOutput{}, fmt.Errorf("repos-only and secrets-only cannot both be set")
	}

	refs, err := resolveHydrateTargets(cfg, strings.TrimSpace(input.Target))
	if err != nil {
		return HydrateOutput{}, err
	}

	out := HydrateOutput{}

	for _, ref := range refs {
		workspaceExists, err := u.paths.PathExists(ctx, ref.WorkspacePath)
		if err != nil {
			return HydrateOutput{}, fmt.Errorf("check workspace path: %w", err)
		}
		if !workspaceExists {
			if input.DryRun {
				out.DirectoriesCreated++
			} else if err := u.paths.MkdirAll(ctx, ref.WorkspacePath); err != nil {
				return HydrateOutput{}, fmt.Errorf("create workspace path: %w", err)
			} else {
				out.DirectoriesCreated++
			}
		}

		repoPath := filepath.Join(ref.WorkspacePath, ref.RepositoryID)
		repoExists, err := u.paths.PathExists(ctx, repoPath)
		if err != nil {
			return HydrateOutput{}, fmt.Errorf("check repo path: %w", err)
		}

		if !input.SecretsOnly {
			if repoExists {
				out.Skipped++
			} else if input.DryRun {
				out.ReposCloned++
			} else if err := u.git.Clone(ctx, ref.Repository.URL, repoPath); err != nil {
				return HydrateOutput{}, fmt.Errorf("clone repo: %w", err)
			} else {
				out.ReposCloned++
				repoExists = true
			}
		}

		if input.ReposOnly || ref.Repository.Secret == nil {
			continue
		}

		if !repoExists && input.DryRun {
			out.SecretsMaterialized++
			continue
		}

		secretPath := filepath.Join(cfg.Secrets.Path, ref.Repository.Secret.Source)
		envPath := filepath.Join(repoPath, ref.Repository.Secret.Target)
		envExists, err := u.paths.PathExists(ctx, envPath)
		if err != nil {
			return HydrateOutput{}, fmt.Errorf("check env path: %w", err)
		}
		if envExists && !input.ForceEnv {
			out.Skipped++
			continue
		}

		if input.DryRun {
			out.SecretsMaterialized++
			continue
		}

		if err := u.secrets.DecryptFile(ctx, secretPath, envPath); err != nil {
			return HydrateOutput{}, fmt.Errorf("materialize secret: %w", err)
		}
		out.SecretsMaterialized++
	}

	return out, nil
}

type hydrateRepoRef struct {
	WorkspacePath string
	RepositoryID  string
	Repository    domain.Repository
}

func resolveHydrateTargets(cfg domain.Config, target string) ([]hydrateRepoRef, error) {
	refs := []hydrateRepoRef{}
	seen := map[string]struct{}{}

	appendWorkspace := func(ws domain.Workspace) {
		for _, repoID := range ws.Repos {
			repo, ok := cfg.Repos[repoID]
			if !ok {
				continue
			}
			key := ws.Path + "|" + repoID
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			refs = append(refs, hydrateRepoRef{WorkspacePath: ws.Path, RepositoryID: repoID, Repository: repo})
		}
	}

	if target == "" {
		for _, ws := range cfg.Workspaces {
			appendWorkspace(ws)
		}
		return refs, nil
	}

	if ws, ok := cfg.Workspaces[target]; ok {
		appendWorkspace(ws)
		return refs, nil
	}

	if repo, ok := cfg.Repos[target]; ok {
		for _, ws := range cfg.Workspaces {
			for _, repoID := range ws.Repos {
				if repoID == target {
					return []hydrateRepoRef{{WorkspacePath: ws.Path, RepositoryID: target, Repository: repo}}, nil
				}
			}
		}
		return nil, fmt.Errorf("repository not attached to workspace: %s", target)
	}

	return nil, fmt.Errorf("%w: %s", ErrHydrateTargetNotFound, target)
}
