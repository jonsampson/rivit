package adapter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type ValidateProbe struct{}

func NewValidateProbe() ValidateProbe {
	return ValidateProbe{}
}

func (p ValidateProbe) PathExists(_ context.Context, path string) (bool, error) {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat path %s: %w", path, err)
	}
	return true, nil
}

func (p ValidateProbe) OriginRemote(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("read origin remote for %s: %w", repoPath, err)
	}
	return strings.TrimSpace(string(output)), nil
}
