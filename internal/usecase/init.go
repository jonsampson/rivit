package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jonsampson/rivit/internal/domain"
)

var (
	ErrInitAlreadyInitialized  = errors.New("config already initialized")
	ErrInitSecretsPathRequired = errors.New("secrets path is required")
)

type initConfigStore interface {
	Exists(context.Context) (bool, error)
	Save(context.Context, domain.Config) error
}

type InitInput struct {
	SecretsPath string
}

type Init struct {
	store initConfigStore
}

func NewInit(store initConfigStore) Init {
	return Init{store: store}
}

func (u Init) Execute(ctx context.Context, input InitInput) error {
	secretsPath := strings.TrimSpace(input.SecretsPath)
	if secretsPath == "" {
		return ErrInitSecretsPathRequired
	}

	exists, err := u.store.Exists(ctx)
	if err != nil {
		return fmt.Errorf("check config existence: %w", err)
	}
	if exists {
		return ErrInitAlreadyInitialized
	}

	cfg := domain.Config{
		Version:    1,
		Workspaces: map[string]domain.Workspace{},
		Repos:      map[string]domain.Repository{},
		Secrets: domain.SecretsConfig{
			Provider: "sops",
			Path:     secretsPath,
		},
	}

	if err := u.store.Save(ctx, cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	return nil
}
