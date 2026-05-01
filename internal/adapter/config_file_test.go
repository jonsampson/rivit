package adapter

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonsampson/rivit/internal/domain"
)

func TestConfigFileStoreSaveQuotesAndSortsMapKeys(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "rivit.yaml")
	store := NewConfigFileStore(configPath)

	cfg := domain.Config{
		Version: 1,
		Workspaces: map[string]domain.Workspace{
			"zeta": {Path: "/tmp/zeta", Repos: []string{"b", "a"}},
			"alpha": {Path: "/tmp/alpha", Repos: []string{"a"}},
		},
		Repos: map[string]domain.Repository{
			"gitlab.ncci.com:2222/zeta/repo": {
				URL: "ssh://git@gitlab.ncci.com:2222/zeta/repo.git",
				Secret: &domain.Secret{
					Source: "gitlab.ncci.com:2222/zeta/repo.env.sops",
					Target: ".env",
				},
			},
			"gitlab.ncci.com:2222/alpha/repo": {
				URL: "ssh://git@gitlab.ncci.com:2222/alpha/repo.git",
				Secret: &domain.Secret{
					Source: "gitlab.ncci.com:2222/alpha/repo.env.sops",
					Target: ".env",
				},
			},
		},
		Secrets: domain.SecretsConfig{Provider: "sops", Path: "/tmp/secrets"},
	}

	if err := store.Save(context.Background(), cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "\"gitlab.ncci.com:2222/alpha/repo\":") {
		t.Fatalf("expected quoted repo key, got:\n%s", content)
	}
	if !strings.Contains(content, "\"gitlab.ncci.com:2222/zeta/repo\":") {
		t.Fatalf("expected quoted repo key, got:\n%s", content)
	}
	if !strings.Contains(content, "\"alpha\":") {
		t.Fatalf("expected quoted workspace key, got:\n%s", content)
	}
	if !strings.Contains(content, "\"zeta\":") {
		t.Fatalf("expected quoted workspace key, got:\n%s", content)
	}

	if strings.Contains(content, "\n?") {
		t.Fatalf("expected no explicit key syntax, got:\n%s", content)
	}

	repoAlphaIdx := strings.Index(content, "\"gitlab.ncci.com:2222/alpha/repo\":")
	repoZetaIdx := strings.Index(content, "\"gitlab.ncci.com:2222/zeta/repo\":")
	if repoAlphaIdx == -1 || repoZetaIdx == -1 || repoAlphaIdx >= repoZetaIdx {
		t.Fatalf("expected sorted repo keys, got:\n%s", content)
	}

	wsAlphaIdx := strings.Index(content, "\"alpha\":")
	wsZetaIdx := strings.Index(content, "\"zeta\":")
	if wsAlphaIdx == -1 || wsZetaIdx == -1 || wsAlphaIdx >= wsZetaIdx {
		t.Fatalf("expected sorted workspace keys, got:\n%s", content)
	}

	if !strings.Contains(content, "repos:\n      - a\n      - b") {
		t.Fatalf("expected sorted workspace repo list, got:\n%s", content)
	}
}
