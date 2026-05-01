package adapter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

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

	data, err := marshalConfigYAML(fromDomainConfig(cfg))
	if err != nil {
		return fmt.Errorf("encode config yaml: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

func marshalConfigYAML(cfg fileConfig) ([]byte, error) {
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}

	if len(doc.Content) > 0 {
		root := doc.Content[0]
		quoteAndSortMapField(root, "workspaces")
		quoteAndSortMapField(root, "repos")
	}

	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func quoteAndSortMapField(root *yaml.Node, field string) {
	if root == nil || root.Kind != yaml.MappingNode {
		return
	}

	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i]
		value := root.Content[i+1]
		if key.Value != field || value.Kind != yaml.MappingNode {
			continue
		}

		sortMappingNode(value)
		for j := 0; j+1 < len(value.Content); j += 2 {
			value.Content[j].Style = yaml.DoubleQuotedStyle
		}
		return
	}
}

func sortMappingNode(node *yaml.Node) {
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}

	type pair struct {
		key   *yaml.Node
		value *yaml.Node
	}

	pairs := make([]pair, 0, len(node.Content)/2)
	for i := 0; i+1 < len(node.Content); i += 2 {
		pairs = append(pairs, pair{key: node.Content[i], value: node.Content[i+1]})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].key.Value < pairs[j].key.Value
	})

	node.Content = node.Content[:0]
	for _, p := range pairs {
		node.Content = append(node.Content, p.key, p.value)
	}
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
			result.Workspaces[name] = fileWorkspace{Path: ws.Path, Repos: sortedStringsCopy(ws.Repos)}
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

func sortedStringsCopy(values []string) []string {
	if len(values) == 0 {
		return values
	}

	result := make([]string, len(values))
	copy(result, values)
	sort.Strings(result)
	return result
}
