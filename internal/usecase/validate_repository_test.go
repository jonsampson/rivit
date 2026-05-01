package usecase

import (
	"context"
	"testing"

	"github.com/jonsampson/rivit/internal/domain"
)

func TestValidateRepositoryExecute(t *testing.T) {
	store := &memoryConfigStore{config: domain.Config{
		Version: 1,
		Workspaces: map[string]domain.Workspace{
			"personal": {Path: "/ws", Repos: []string{"github.com/org/repo"}},
		},
		Repos: map[string]domain.Repository{
			"github.com/org/repo": {URL: "git@github.com:org/repo.git"},
		},
	}}

	probe := memoryProbe{paths: map[string]probeResult{
		"/ws/github.com/org/repo": {exists: true},
	}, remotes: map[string]string{"/ws/github.com/org/repo": "git@github.com:org/other.git"}}

	uc := NewValidateRepository(store, probe)
	issues, err := uc.Execute(context.Background(), ValidateRepositoryInput{RepositoryID: "github.com/org/repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %+v", issues)
	}
}
