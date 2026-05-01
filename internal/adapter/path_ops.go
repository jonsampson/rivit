package adapter

import (
	"context"
	"errors"
	"fmt"
	"os"
)

type PathOps struct{}

func NewPathOps() PathOps {
	return PathOps{}
}

func (p PathOps) PathExists(_ context.Context, path string) (bool, error) {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat path %s: %w", path, err)
	}
	return true, nil
}

func (p PathOps) MkdirAll(_ context.Context, path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}
	return nil
}
