package domain

type Config struct {
	Version    int                  `yaml:"version"`
	Workspaces map[string]Workspace `yaml:"workspaces"`
	Secrets    SecretsConfig        `yaml:"secrets"`
}

type SecretsConfig struct {
	Provider string `yaml:"provider"`
	Path     string `yaml:"path"`
}
