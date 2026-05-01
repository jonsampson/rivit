package domain

type Repository struct {
	URL    string  `yaml:"url"`
	Secret *Secret `yaml:"secret,omitempty"`
}
