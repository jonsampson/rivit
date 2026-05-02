package usecase

import (
	"context"
	"testing"

	"github.com/jonsampson/rivit/internal/domain"
)

type memoryPathOps struct {
	exists map[string]bool
	mkdirs []string
}

func (m *memoryPathOps) PathExists(_ context.Context, path string) (bool, error) {
	return m.exists[path], nil
}

func (m *memoryPathOps) MkdirAll(_ context.Context, path string) error {
	m.mkdirs = append(m.mkdirs, path)
	m.exists[path] = true
	return nil
}

type memoryGitOps struct{ clones []string }

func (m *memoryGitOps) Clone(_ context.Context, remoteURL string, path string) error {
	m.clones = append(m.clones, remoteURL+"->"+path)
	return nil
}

type memorySecretOps struct{ writes []string }

func (m *memorySecretOps) DecryptFile(_ context.Context, sourcePath string, targetPath string) error {
	m.writes = append(m.writes, sourcePath+"->"+targetPath)
	return nil
}

func TestHydrateExecute(t *testing.T) {
	store := &memoryConfigStore{config: domain.Config{
		Version: 1,
		Workspaces: map[string]domain.Workspace{
			"personal": {Path: "/ws", Repos: []domain.Repository{{
				URL:    "git@github.com:org/repo.git",
				Secret: &domain.Secret{Source: "github.com/org/repo.env.sops", Target: ".env"},
			}}},
		},
		Secrets: domain.SecretsConfig{Path: "/secrets"},
	}}

	paths := &memoryPathOps{exists: map[string]bool{}}
	git := &memoryGitOps{}
	secrets := &memorySecretOps{}

	uc := NewHydrate(store, paths, git, secrets)
	out, err := uc.Execute(context.Background(), HydrateInput{Target: "personal"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ReposCloned != 1 || out.SecretsMaterialized != 1 || out.DirectoriesCreated != 1 {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestHydrateExecuteDryRun(t *testing.T) {
	store := &memoryConfigStore{config: domain.Config{
		Version: 1,
		Workspaces: map[string]domain.Workspace{
			"personal": {Path: "/ws", Repos: []domain.Repository{{
				URL:    "git@github.com:org/repo.git",
				Secret: &domain.Secret{Source: "github.com/org/repo.env.sops", Target: ".env"},
			}}},
		},
		Secrets: domain.SecretsConfig{Path: "/secrets"},
	}}
	paths := &memoryPathOps{exists: map[string]bool{}}
	uc := NewHydrate(store, paths, &memoryGitOps{}, &memorySecretOps{})

	out, err := uc.Execute(context.Background(), HydrateInput{Target: "personal", DryRun: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ReposCloned != 1 || out.SecretsMaterialized != 1 || out.DirectoriesCreated != 1 {
		t.Fatalf("unexpected output: %+v", out)
	}
}
