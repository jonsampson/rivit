package domain

type ValidationIssue struct {
	Scope   string
	Code    string
	Message string
}

type WorkspaceValidationInput struct {
	WorkspaceName   string
	WorkspacePath   string
	WorkspaceExists bool
	Repositories    []RepositoryValidationInput
}

type RepositoryValidationInput struct {
	RepositoryID       string
	ExpectedPath       string
	PathExists         bool
	ExpectedRemoteURL  string
	ActualRemoteURL    string
	RemoteLookupFailed bool
	HasSecret          bool
	SecretSourcePath   string
	SecretSourceExists bool
	EnvTargetPath      string
	EnvTargetExists    bool
}

func ValidateWorkspace(input WorkspaceValidationInput) []ValidationIssue {
	issues := []ValidationIssue{}

	if !input.WorkspaceExists {
		issues = append(issues, ValidationIssue{
			Scope:   input.WorkspaceName,
			Code:    "workspace_path_missing",
			Message: "workspace path does not exist: " + input.WorkspacePath,
		})
	}

	for _, repo := range input.Repositories {
		issues = append(issues, ValidateRepository(repo)...)
	}

	return issues
}

func ValidateRepository(input RepositoryValidationInput) []ValidationIssue {
	issues := []ValidationIssue{}

	if !input.PathExists {
		issues = append(issues, ValidationIssue{
			Scope:   input.RepositoryID,
			Code:    "repo_path_missing",
			Message: "repository path does not exist: " + input.ExpectedPath,
		})
	}

	if input.RemoteLookupFailed {
		issues = append(issues, ValidationIssue{
			Scope:   input.RepositoryID,
			Code:    "repo_remote_unreadable",
			Message: "repository origin remote could not be read",
		})
	} else if input.PathExists && input.ExpectedRemoteURL != "" && input.ExpectedRemoteURL != input.ActualRemoteURL {
		issues = append(issues, ValidationIssue{
			Scope:   input.RepositoryID,
			Code:    "repo_remote_mismatch",
			Message: "repository origin remote mismatch",
		})
	}

	if input.HasSecret {
		if !input.SecretSourceExists {
			issues = append(issues, ValidationIssue{
				Scope:   input.RepositoryID,
				Code:    "secret_source_missing",
				Message: "secret source file does not exist: " + input.SecretSourcePath,
			})
		}
		if !input.EnvTargetExists {
			issues = append(issues, ValidationIssue{
				Scope:   input.RepositoryID,
				Code:    "env_target_missing",
				Message: "materialized env file does not exist: " + input.EnvTargetPath,
			})
		}
	}

	return issues
}
