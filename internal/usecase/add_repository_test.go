package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/jonsampson/rivit/internal/domain"
)

func TestAddRepositoryExecute(t *testing.T) {
	t.Run("adds repo to workspace and catalog", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{"personal": {Path: "~/Code"}}}}
		uc := NewAddRepository(store)

		repoID, err := uc.Execute(context.Background(), AddRepositoryInput{
			URL:       "git@github.com:jonsampson/rivit.git",
			Workspace: "personal",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if repoID != "github.com/jonsampson/rivit" {
			t.Fatalf("unexpected repo id: %s", repoID)
		}

		ws := store.config.Workspaces["personal"]
		if len(ws.Repos) != 1 || ws.Repos[0] != repoID {
			t.Fatalf("workspace repo not linked: %+v", ws.Repos)
		}

		repo := store.config.Repos[repoID]
		if repo.URL != "git@github.com:jonsampson/rivit.git" {
			t.Fatalf("repo catalog not saved: %+v", repo)
		}
	})

	t.Run("fails when workspace missing", func(t *testing.T) {
		store := &memoryConfigStore{config: domain.Config{Version: 1, Workspaces: map[string]domain.Workspace{}}}
		uc := NewAddRepository(store)

		_, err := uc.Execute(context.Background(), AddRepositoryInput{URL: "git@github.com:jonsampson/rivit.git", Workspace: "personal"})
		if !errors.Is(err, ErrWorkspaceNotFound) {
			t.Fatalf("expected ErrWorkspaceNotFound, got %v", err)
		}
	})
}

func TestRepoIDFromRemoteURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "github ssh", in: "git@github.com:jonsampson/rivit.git", want: "github.com/jonsampson/rivit"},
		{name: "github https", in: "https://github.com/jonsampson/rivit.git", want: "github.com/jonsampson/rivit"},
		{name: "azure https", in: "https://dev.azure.com/org/project/_git/repo", want: "dev.azure.com/org/project/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repoIDFromRemoteURL(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
