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
	extraIssues := []domain.ValidationIssue{}
	for _, repoID := range ws.Repos {
		repo, exists := cfg.Repos[repoID]
		if !exists {
			extraIssues = append(extraIssues, domain.ValidationIssue{
				Scope:   repoID,
				Code:    "repo_missing_from_catalog",
				Message: "repository is referenced by workspace but missing from repo catalog",
			})
			continue
		}

		inputModel, err := buildRepositoryValidationInput(ctx, u.probe, cfg, repoID, repo, ws.Path)
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
	issues = append(issues, extraIssues...)

	return issues, nil
}
