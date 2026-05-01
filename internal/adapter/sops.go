package adapter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type SOPS struct{}

func NewSOPS() SOPS {
	return SOPS{}
}

func (s SOPS) DecryptFile(ctx context.Context, sourcePath string, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}

	cmd := exec.CommandContext(ctx, "sops", "--decrypt", sourcePath)
	cmd.Dir = filepath.Dir(sourcePath)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("sops decrypt failed: %w", err)
	}

	if err := os.WriteFile(targetPath, output, 0o600); err != nil {
		return fmt.Errorf("write env target: %w", err)
	}

	return nil
}

func (s SOPS) EncryptFile(ctx context.Context, sourcePath string, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}

	cmd := exec.CommandContext(ctx, "sops", "--encrypt", "--filename-override", targetPath, sourcePath)
	cmd.Dir = filepath.Dir(targetPath)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("sops encrypt failed: %w", err)
	}

	if err := os.WriteFile(targetPath, output, 0o600); err != nil {
		return fmt.Errorf("write secret target: %w", err)
	}

	return nil
}
