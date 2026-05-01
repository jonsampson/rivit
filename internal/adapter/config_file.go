package adapter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jonsampson/rivit/internal/domain"
	"gopkg.in/yaml.v3"
)

type fileConfig struct {
	Version    int                       `yaml:"version"`
	Workspaces map[string]fileWorkspace  `yaml:"workspaces"`
	Repos      map[string]fileRepository `yaml:"repos"`
	Secrets    fileSecretsConfiguration  `yaml:"secrets"`
}

type fileWorkspace struct {
	Path  string   `yaml:"path"`
	Repos []string `yaml:"repos"`
}

type fileRepository struct {
	URL    string      `yaml:"url"`
	Secret *fileSecret `yaml:"secret,omitempty"`
}

type fileSecret struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

type fileSecretsConfiguration struct {
	Provider string `yaml:"provider"`
	Path     string `yaml:"path"`
}

type ConfigFileStore struct {
	path string
}

func NewConfigFileStore(path string) ConfigFileStore {
	return ConfigFileStore{path: path}
}

func (s ConfigFileStore) Exists(_ context.Context) (bool, error) {
	_, err := os.Stat(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat config file: %w", err)
	}
	return true, nil
}

func (s ConfigFileStore) Load(_ context.Context) (domain.Config, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return domain.Config{Version: 1}, nil
	}
	if err != nil {
		return domain.Config{}, fmt.Errorf("read config file: %w", err)
	}

	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return domain.Config{}, fmt.Errorf("decode config yaml: %w", err)
	}

	return toDomainConfig(cfg), nil
}

func (s ConfigFileStore) Save(_ context.Context, cfg domain.Config) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := yaml.Marshal(fromDomainConfig(cfg))
	if err != nil {
		return fmt.Errorf("encode config yaml: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

func toDomainConfig(cfg fileConfig) domain.Config {
	if cfg.Version == 0 {
		cfg.Version = 1
	}

	result := domain.Config{
		Version: cfg.Version,
		Secrets: domain.SecretsConfig{
			Provider: cfg.Secrets.Provider,
			Path:     cfg.Secrets.Path,
		},
	}

	if len(cfg.Workspaces) > 0 {
		result.Workspaces = make(map[string]domain.Workspace, len(cfg.Workspaces))
		for name, ws := range cfg.Workspaces {
			result.Workspaces[name] = domain.Workspace{Path: ws.Path, Repos: ws.Repos}
		}
	}

	if len(cfg.Repos) > 0 {
		result.Repos = make(map[string]domain.Repository, len(cfg.Repos))
		for id, repo := range cfg.Repos {
			mapped := domain.Repository{URL: repo.URL}
			if repo.Secret != nil {
				mapped.Secret = &domain.Secret{Source: repo.Secret.Source, Target: repo.Secret.Target}
			}
			result.Repos[id] = mapped
		}
	}

	return result
}

func fromDomainConfig(cfg domain.Config) fileConfig {
	result := fileConfig{
		Version: cfg.Version,
		Secrets: fileSecretsConfiguration{Provider: cfg.Secrets.Provider, Path: cfg.Secrets.Path},
	}

	if result.Version == 0 {
		result.Version = 1
	}

	if len(cfg.Workspaces) > 0 {
		result.Workspaces = make(map[string]fileWorkspace, len(cfg.Workspaces))
		for name, ws := range cfg.Workspaces {
			result.Workspaces[name] = fileWorkspace{Path: ws.Path, Repos: ws.Repos}
		}
	}

	if len(cfg.Repos) > 0 {
		result.Repos = make(map[string]fileRepository, len(cfg.Repos))
		for id, repo := range cfg.Repos {
			mapped := fileRepository{URL: repo.URL}
			if repo.Secret != nil {
				mapped.Secret = &fileSecret{Source: repo.Secret.Source, Target: repo.Secret.Target}
			}
			result.Repos[id] = mapped
		}
	}

	return result
}
