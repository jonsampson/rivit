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
