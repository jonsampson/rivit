package usecase

import (
	"context"
	"testing"

	"github.com/jonsampson/rivit/internal/domain"
)

func TestValidateWorkspaceExecute(t *testing.T) {
	store := &memoryConfigStore{config: domain.Config{
		Version: 1,
		Workspaces: map[string]domain.Workspace{
			"personal": {Path: "/ws", Repos: []string{"github.com/org/repo"}},
		},
		Repos: map[string]domain.Repository{
			"github.com/org/repo": {
				URL:    "git@github.com:org/repo.git",
				Secret: &domain.Secret{Source: "github.com/org/repo.env.sops", Target: ".env"},
			},
		},
		Secrets: domain.SecretsConfig{Path: "/secrets"},
	}}

	probe := memoryProbe{paths: map[string]probeResult{
		"/ws":                                   {exists: true},
		"/ws/github.com/org/repo":               {exists: true},
		"/secrets/github.com/org/repo.env.sops": {exists: true},
		"/ws/github.com/org/repo/.env":          {exists: true},
	}, remotes: map[string]string{"/ws/github.com/org/repo": "git@github.com:org/repo.git"}}

	uc := NewValidateWorkspace(store, probe)
	issues, err := uc.Execute(context.Background(), ValidateWorkspaceInput{WorkspaceName: "personal"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %+v", issues)
	}
}
