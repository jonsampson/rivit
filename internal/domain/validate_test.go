package domain

import "testing"

func TestValidateRepository(t *testing.T) {
	issues := ValidateRepository(RepositoryValidationInput{
		RepositoryID:       "github.com/org/repo",
		ExpectedPath:       "/tmp/repo",
		PathExists:         false,
		ExpectedRemoteURL:  "git@github.com:org/repo.git",
		RemoteLookupFailed: true,
		HasSecret:          true,
		SecretSourcePath:   "/tmp/secrets/repo.env.sops",
		SecretSourceExists: false,
		EnvTargetPath:      "/tmp/repo/.env",
		EnvTargetExists:    false,
	})

	if len(issues) != 4 {
		t.Fatalf("expected 4 issues, got %d", len(issues))
	}
}

func TestValidateWorkspace(t *testing.T) {
	issues := ValidateWorkspace(WorkspaceValidationInput{
		WorkspaceName:   "personal",
		WorkspacePath:   "~/Code",
		WorkspaceExists: false,
		Repositories: []RepositoryValidationInput{{
			RepositoryID:      "github.com/org/repo",
			ExpectedPath:      "/tmp/repo",
			PathExists:        true,
			ExpectedRemoteURL: "git@github.com:org/repo.git",
			ActualRemoteURL:   "git@github.com:org/other.git",
		}},
	})

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
}
