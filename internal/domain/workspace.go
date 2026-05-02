package domain

type Workspace struct {
	Path  string       `yaml:"path"`
	Repos []Repository `yaml:"repos"`
}
