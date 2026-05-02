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
	if remoteURL == "git@github.com:org/fail.git" {
		return context.DeadlineExceeded
	}
	m.clones = append(m.clones, remoteURL+"->"+path)
	return nil
}

type memorySecretOps struct{ writes []string }

func (m *memorySecretOps) DecryptFile(_ context.Context, sourcePath string, targetPath string) error {
	if sourcePath == "/secrets/github.com/org/fail.env.sops" {
		return context.DeadlineExceeded
	}
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

	paths := &memoryPathOps{exists: map[string]bool{"/secrets/github.com/org/repo.env.sops": true}}
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
	paths := &memoryPathOps{exists: map[string]bool{"/secrets/github.com/org/repo.env.sops": true}}
	uc := NewHydrate(store, paths, &memoryGitOps{}, &memorySecretOps{})

	out, err := uc.Execute(context.Background(), HydrateInput{Target: "personal", DryRun: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ReposCloned != 1 || out.SecretsMaterialized != 1 || out.DirectoriesCreated != 1 {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestHydrateExecuteSkipsWhenSecretSourceMissing(t *testing.T) {
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
	if out.ReposCloned != 1 || out.SecretsMaterialized != 0 || out.Skipped != 1 || out.DirectoriesCreated != 1 {
		t.Fatalf("unexpected output: %+v", out)
	}
	if len(secrets.writes) != 0 {
		t.Fatalf("expected no secret writes, got %+v", secrets.writes)
	}
}

func TestHydrateExecuteContinuesOnCloneOrDecryptFailure(t *testing.T) {
	store := &memoryConfigStore{config: domain.Config{
		Version: 1,
		Workspaces: map[string]domain.Workspace{
			"personal": {Path: "/ws", Repos: []domain.Repository{
				{URL: "git@github.com:org/repo.git", Secret: &domain.Secret{Source: "github.com/org/repo.env.sops", Target: ".env"}},
				{URL: "git@github.com:org/fail.git", Secret: &domain.Secret{Source: "github.com/org/fail.env.sops", Target: ".env"}},
			}},
		},
		Secrets: domain.SecretsConfig{Path: "/secrets"},
	}}

	paths := &memoryPathOps{exists: map[string]bool{
		"/secrets/github.com/org/repo.env.sops": true,
		"/secrets/github.com/org/fail.env.sops": true,
	}}

	uc := NewHydrate(store, paths, &memoryGitOps{}, &memorySecretOps{})
	out, err := uc.Execute(context.Background(), HydrateInput{Target: "personal"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ReposCloned != 1 || out.SecretsMaterialized != 1 || out.Skipped != 1 {
		t.Fatalf("unexpected output: %+v", out)
	}
	if len(out.Failures) != 1 || out.Failures[0].Step != "clone" {
		t.Fatalf("expected one clone failure, got %+v", out.Failures)
	}
}
