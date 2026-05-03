package adapter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitDiscoverer struct{}

func NewGitDiscoverer() GitDiscoverer {
	return GitDiscoverer{}
}

func (d GitDiscoverer) Discover(ctx context.Context, root string, visit func(repoPath string, remoteURL string) error) error {
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		if entry.Name() != ".git" {
			return nil
		}

		repoPath := filepath.Dir(path)
		remote, remoteErr := gitOriginRemote(ctx, repoPath)
		if remoteErr == nil && strings.TrimSpace(remote) != "" {
			if err := visit(repoPath, strings.TrimSpace(remote)); err != nil {
				return err
			}
		}

		return filepath.SkipDir
	})
	if err != nil {
		return fmt.Errorf("walk directory %s: %w", root, err)
	}

	return nil
}

func gitOriginRemote(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("read origin remote for %s: %w", repoPath, err)
	}
	return string(output), nil
}

func (d GitDiscoverer) Clone(ctx context.Context, remoteURL string, path string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", remoteURL, path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
