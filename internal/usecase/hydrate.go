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
	Progress    func(HydrateProgress)
}

type HydrateProgress struct {
	Current       int
	Total         int
	RepositoryURL string
	Stage         string
}

type HydrateOutput struct {
	DirectoriesCreated  int
	ReposCloned         int
	SecretsMaterialized int
	Skipped             int
	SkipReasons         map[string]int
	Failures            []HydrateFailure
}

type HydrateFailure struct {
	RepositoryURL string
	Step          string
	Message       string
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

	out := HydrateOutput{SkipReasons: map[string]int{}}

	for i, ref := range refs {
		if input.Progress != nil {
			input.Progress(HydrateProgress{Current: i + 1, Total: len(refs), RepositoryURL: ref.Repository.URL, Stage: "start"})
		}

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
				out.SkipReasons["repo_exists"]++
				if input.Progress != nil {
					input.Progress(HydrateProgress{Current: i + 1, Total: len(refs), RepositoryURL: ref.Repository.URL, Stage: "repo_exists"})
				}
			} else if input.DryRun {
				out.ReposCloned++
				if input.Progress != nil {
					input.Progress(HydrateProgress{Current: i + 1, Total: len(refs), RepositoryURL: ref.Repository.URL, Stage: "clone_dry_run"})
				}
			} else if err := u.git.Clone(ctx, ref.Repository.URL, repoPath); err != nil {
				out.Skipped++
				out.SkipReasons["clone_failed"]++
				out.Failures = append(out.Failures, HydrateFailure{
					RepositoryURL: ref.Repository.URL,
					Step:          "clone",
					Message:       err.Error(),
				})
				if input.Progress != nil {
					input.Progress(HydrateProgress{Current: i + 1, Total: len(refs), RepositoryURL: ref.Repository.URL, Stage: "clone_failed"})
				}
				continue
			} else {
				out.ReposCloned++
				repoExists = true
				if input.Progress != nil {
					input.Progress(HydrateProgress{Current: i + 1, Total: len(refs), RepositoryURL: ref.Repository.URL, Stage: "cloned"})
				}
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
		secretExists, err := u.paths.PathExists(ctx, secretPath)
		if err != nil {
			return HydrateOutput{}, fmt.Errorf("check secret path: %w", err)
		}
		if !secretExists {
			out.Skipped++
			out.SkipReasons["secret_missing"]++
			if input.Progress != nil {
				input.Progress(HydrateProgress{Current: i + 1, Total: len(refs), RepositoryURL: ref.Repository.URL, Stage: "secret_missing"})
			}
			continue
		}

		envExists, err := u.paths.PathExists(ctx, envPath)
		if err != nil {
			return HydrateOutput{}, fmt.Errorf("check env path: %w", err)
		}
		if envExists && !input.ForceEnv {
			out.Skipped++
			out.SkipReasons["env_exists"]++
			if input.Progress != nil {
				input.Progress(HydrateProgress{Current: i + 1, Total: len(refs), RepositoryURL: ref.Repository.URL, Stage: "env_exists"})
			}
			continue
		}

		if input.DryRun {
			out.SecretsMaterialized++
			if input.Progress != nil {
				input.Progress(HydrateProgress{Current: i + 1, Total: len(refs), RepositoryURL: ref.Repository.URL, Stage: "decrypt_dry_run"})
			}
			continue
		}

		if err := u.secrets.DecryptFile(ctx, secretPath, envPath); err != nil {
			out.Skipped++
			out.SkipReasons["decrypt_failed"]++
			out.Failures = append(out.Failures, HydrateFailure{
				RepositoryURL: ref.Repository.URL,
				Step:          "decrypt",
				Message:       err.Error(),
			})
			if input.Progress != nil {
				input.Progress(HydrateProgress{Current: i + 1, Total: len(refs), RepositoryURL: ref.Repository.URL, Stage: "decrypt_failed"})
			}
			continue
		}
		out.SecretsMaterialized++
		if input.Progress != nil {
			input.Progress(HydrateProgress{Current: i + 1, Total: len(refs), RepositoryURL: ref.Repository.URL, Stage: "decrypted"})
		}
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
		for _, repo := range ws.Repos {
			repoID, err := domain.RepoIDFromRemoteURL(repo.URL)
			if err != nil {
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

	for _, ws := range cfg.Workspaces {
		for _, repo := range ws.Repos {
			if repo.URL != target {
				continue
			}
			repoID, err := domain.RepoIDFromRemoteURL(repo.URL)
			if err != nil {
				return nil, err
			}
			return []hydrateRepoRef{{WorkspacePath: ws.Path, RepositoryID: repoID, Repository: repo}}, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrHydrateTargetNotFound, target)
}
