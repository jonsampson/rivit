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
			"zeta": {
				Path: "/tmp/zeta",
				Repos: []domain.Repository{
					{URL: "ssh://git@gitlab.ncci.com:2222/zeta/repo-b.git"},
					{URL: "ssh://git@gitlab.ncci.com:2222/zeta/repo-a.git"},
				},
			},
			"alpha": {Path: "/tmp/alpha", Repos: []domain.Repository{{URL: "ssh://git@gitlab.ncci.com:2222/alpha/repo.git"}}},
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

	if !strings.Contains(content, "\"alpha\":") {
		t.Fatalf("expected quoted workspace key, got:\n%s", content)
	}
	if !strings.Contains(content, "\"zeta\":") {
		t.Fatalf("expected quoted workspace key, got:\n%s", content)
	}

	if strings.Contains(content, "\n?") {
		t.Fatalf("expected no explicit key syntax, got:\n%s", content)
	}

	wsAlphaIdx := strings.Index(content, "\"alpha\":")
	wsZetaIdx := strings.Index(content, "\"zeta\":")
	if wsAlphaIdx == -1 || wsZetaIdx == -1 || wsAlphaIdx >= wsZetaIdx {
		t.Fatalf("expected sorted workspace keys, got:\n%s", content)
	}

	if !strings.Contains(content, "repos:\n      - url: ssh://git@gitlab.ncci.com:2222/zeta/repo-a.git\n      - url: ssh://git@gitlab.ncci.com:2222/zeta/repo-b.git") {
		t.Fatalf("expected sorted workspace repo list, got:\n%s", content)
	}
}
