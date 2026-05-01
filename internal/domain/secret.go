package domain

type Secret struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}
