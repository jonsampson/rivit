package domain

type Workspace struct {
	Path  string   `yaml:"path"`
	Repos []string `yaml:"repos"`
}
