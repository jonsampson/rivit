package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/jonsampson/rivit/internal/domain"
)

type memoryEncryptOps struct{ writes []string }

func (m *memoryEncryptOps) EncryptFile(_ context.Context, sourcePath string, targetPath string) error {
	m.writes = append(m.writes, sourcePath+"->"+targetPath)
	return nil
}

func TestAbsorbExecute(t *testing.T) {
	store := &memoryConfigStore{config: domain.Config{
		Version: 1,
		Workspaces: map[string]domain.Workspace{
			"personal": {Path: "/ws", Repos: []string{"github.com/org/repo"}},
		},
		Repos: map[string]domain.Repository{
			"github.com/org/repo": {URL: "git@github.com:org/repo.git", Secret: &domain.Secret{Source: "github.com/org/repo.env.sops", Target: ".env"}},
		},
		Secrets: domain.SecretsConfig{Path: "/secrets"},
	}}

	paths := &memoryPathOps{exists: map[string]bool{"/ws/github.com/org/repo/.env": true}}
	enc := &memoryEncryptOps{}
	uc := NewAbsorb(store, paths, enc)

	out, err := uc.Execute(context.Background(), AbsorbInput{Target: "personal", Yes: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Updated != 1 {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestAbsorbExecuteRequiresYes(t *testing.T) {
	uc := NewAbsorb(&memoryConfigStore{}, &memoryPathOps{exists: map[string]bool{}}, &memoryEncryptOps{})
	if _, err := uc.Execute(context.Background(), AbsorbInput{}); !errors.Is(err, ErrAbsorbConfirmationRequired) {
		t.Fatalf("expected confirmation error, got %v", err)
	}
}
