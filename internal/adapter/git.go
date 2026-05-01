package adapter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jonsampson/rivit/internal/domain"
)

type GitDiscoverer struct{}

type gitDiscoveredRepository struct {
	path   string
	remote string
}

func NewGitDiscoverer() GitDiscoverer {
	return GitDiscoverer{}
}

func (d GitDiscoverer) Discover(ctx context.Context, root string) ([]domain.Repository, error) {
	dtos := []gitDiscoveredRepository{}

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
			dtos = append(dtos, gitDiscoveredRepository{path: repoPath, remote: strings.TrimSpace(remote)})
		}

		return filepath.SkipDir
	})
	if err != nil {
		return nil, fmt.Errorf("walk directory %s: %w", root, err)
	}

	repos := make([]domain.Repository, 0, len(dtos))
	for _, dto := range dtos {
		repos = append(repos, domain.Repository{URL: dto.remote})
	}

	return repos, nil
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
