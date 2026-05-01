package usecase

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jonsampson/rivit/internal/domain"
)

var ErrValidateRepositoryIDRequired = errors.New("repository id is required")

type validateConfigReader interface {
	Load(context.Context) (domain.Config, error)
}

type repositoryProbe interface {
	PathExists(context.Context, string) (bool, error)
	OriginRemote(context.Context, string) (string, error)
}

type ValidateRepositoryInput struct {
	RepositoryID string
}

type ValidateRepository struct {
	store validateConfigReader
	probe repositoryProbe
}

func NewValidateRepository(store validateConfigReader, probe repositoryProbe) ValidateRepository {
	return ValidateRepository{store: store, probe: probe}
}

func (u ValidateRepository) Execute(ctx context.Context, input ValidateRepositoryInput) ([]domain.ValidationIssue, error) {
	repoID := strings.TrimSpace(input.RepositoryID)
	if repoID == "" {
		return nil, ErrValidateRepositoryIDRequired
	}

	cfg, err := u.store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	repo, ok := cfg.Repos[repoID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrRepositoryNotFound, repoID)
	}

	wsPath, err := workspacePathForRepository(cfg, repoID)
	if err != nil {
		return nil, err
	}

	inputModel, err := buildRepositoryValidationInput(ctx, u.probe, cfg, repoID, repo, wsPath)
	if err != nil {
		return nil, err
	}

	return domain.ValidateRepository(inputModel), nil
}

func workspacePathForRepository(cfg domain.Config, repoID string) (string, error) {
	for _, ws := range cfg.Workspaces {
		for _, id := range ws.Repos {
			if id == repoID {
				return ws.Path, nil
			}
		}
	}
	return "", fmt.Errorf("repository not attached to any workspace: %s", repoID)
}

func buildRepositoryValidationInput(ctx context.Context, probe repositoryProbe, cfg domain.Config, repoID string, repo domain.Repository, workspacePath string) (domain.RepositoryValidationInput, error) {
	repoPath := filepath.Join(workspacePath, repoID)
	pathExists, err := probe.PathExists(ctx, repoPath)
	if err != nil {
		return domain.RepositoryValidationInput{}, fmt.Errorf("check repository path: %w", err)
	}

	actualRemote := ""
	remoteLookupFailed := false
	if pathExists {
		actualRemote, err = probe.OriginRemote(ctx, repoPath)
		if err != nil {
			remoteLookupFailed = true
		}
	}

	model := domain.RepositoryValidationInput{
		RepositoryID:       repoID,
		ExpectedPath:       repoPath,
		PathExists:         pathExists,
		ExpectedRemoteURL:  repo.URL,
		ActualRemoteURL:    strings.TrimSpace(actualRemote),
		RemoteLookupFailed: remoteLookupFailed,
	}

	if repo.Secret != nil {
		secretPath := filepath.Join(cfg.Secrets.Path, repo.Secret.Source)
		envPath := filepath.Join(repoPath, repo.Secret.Target)

		secretExists, err := probe.PathExists(ctx, secretPath)
		if err != nil {
			return domain.RepositoryValidationInput{}, fmt.Errorf("check secret source path: %w", err)
		}
		envExists, err := probe.PathExists(ctx, envPath)
		if err != nil {
			return domain.RepositoryValidationInput{}, fmt.Errorf("check env target path: %w", err)
		}

		model.HasSecret = true
		model.SecretSourcePath = secretPath
		model.SecretSourceExists = secretExists
		model.EnvTargetPath = envPath
		model.EnvTargetExists = envExists
	}

	return model, nil
}
