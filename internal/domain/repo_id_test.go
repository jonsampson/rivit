package domain

import "testing"

func TestRepoIDFromRemoteURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "github ssh", in: "git@github.com:jonsampson/rivit.git", want: "github.com/jonsampson/rivit"},
		{name: "github https", in: "https://github.com/jonsampson/rivit.git", want: "github.com/jonsampson/rivit"},
		{name: "azure https", in: "https://dev.azure.com/org/project/_git/repo", want: "dev.azure.com/org/project/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RepoIDFromRemoteURL(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRepoIDFromRemoteURLInvalid(t *testing.T) {
	inputs := []string{
		"git@github.com",
		"://not-a-url",
		"https://github.com",
	}

	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			if _, err := RepoIDFromRemoteURL(in); err == nil {
				t.Fatalf("expected error for %q", in)
			}
		})
	}
}
