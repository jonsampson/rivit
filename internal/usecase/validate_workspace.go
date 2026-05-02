package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jonsampson/rivit/internal/domain"
)

var ErrValidateWorkspaceNameRequired = errors.New("workspace name is required")

type ValidateWorkspaceInput struct {
	WorkspaceName string
}

type ValidateWorkspace struct {
	store validateConfigReader
	probe repositoryProbe
}

func NewValidateWorkspace(store validateConfigReader, probe repositoryProbe) ValidateWorkspace {
	return ValidateWorkspace{store: store, probe: probe}
}

func (u ValidateWorkspace) Execute(ctx context.Context, input ValidateWorkspaceInput) ([]domain.ValidationIssue, error) {
	name := strings.TrimSpace(input.WorkspaceName)
	if name == "" {
		return nil, ErrValidateWorkspaceNameRequired
	}

	cfg, err := u.store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	ws, ok := cfg.Workspaces[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrWorkspaceNotFound, name)
	}

	workspaceExists, err := u.probe.PathExists(ctx, ws.Path)
	if err != nil {
		return nil, fmt.Errorf("check workspace path: %w", err)
	}

	repoInputs := make([]domain.RepositoryValidationInput, 0, len(ws.Repos))
	for _, repo := range ws.Repos {
		inputModel, err := buildRepositoryValidationInput(ctx, u.probe, cfg, repo.URL, repo, ws.Path)
		if err != nil {
			return nil, err
		}
		repoInputs = append(repoInputs, inputModel)
	}

	issues := domain.ValidateWorkspace(domain.WorkspaceValidationInput{
		WorkspaceName:   name,
		WorkspacePath:   ws.Path,
		WorkspaceExists: workspaceExists,
		Repositories:    repoInputs,
	})
	return issues, nil
}
