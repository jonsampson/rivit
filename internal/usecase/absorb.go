package usecase

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jonsampson/rivit/internal/domain"
)

var (
	ErrAbsorbTargetNotFound       = errors.New("absorb target not found")
	ErrAbsorbConfirmationRequired = errors.New("absorb requires confirmation; pass --yes")
)

type absorbConfigReader interface {
	Load(context.Context) (domain.Config, error)
}

type absorbPathOps interface {
	PathExists(context.Context, string) (bool, error)
}

type absorbSecretOps interface {
	EncryptFile(context.Context, string, string) error
}

type AbsorbInput struct {
	Target string
	DryRun bool
	Yes    bool
}

type AbsorbOutput struct {
	Updated int
	Skipped int
	SkipReasons map[string]int
	Failures []AbsorbFailure
}

type AbsorbFailure struct {
	RepositoryURL string
	Step          string
	Message       string
}

type Absorb struct {
	store   absorbConfigReader
	paths   absorbPathOps
	secrets absorbSecretOps
}

func NewAbsorb(store absorbConfigReader, paths absorbPathOps, secrets absorbSecretOps) Absorb {
	return Absorb{store: store, paths: paths, secrets: secrets}
}

func (u Absorb) Execute(ctx context.Context, input AbsorbInput) (AbsorbOutput, error) {
	if !input.DryRun && !input.Yes {
		return AbsorbOutput{}, ErrAbsorbConfirmationRequired
	}

	cfg, err := u.store.Load(ctx)
	if err != nil {
		return AbsorbOutput{}, fmt.Errorf("load config: %w", err)
	}

	refs, err := resolveHydrateTargets(cfg, strings.TrimSpace(input.Target))
	if err != nil {
		if errors.Is(err, ErrHydrateTargetNotFound) {
			return AbsorbOutput{}, fmt.Errorf("%w: %s", ErrAbsorbTargetNotFound, strings.TrimSpace(input.Target))
		}
		return AbsorbOutput{}, err
	}

	out := AbsorbOutput{SkipReasons: map[string]int{}}
	for _, ref := range refs {
		if ref.Repository.Secret == nil {
			out.Skipped++
			out.SkipReasons["no_secret_config"]++
			continue
		}

		repoPath := filepath.Join(ref.WorkspacePath, ref.RepositoryID)
		envPath := filepath.Join(repoPath, ref.Repository.Secret.Target)
		exists, err := u.paths.PathExists(ctx, envPath)
		if err != nil {
			return AbsorbOutput{}, fmt.Errorf("check env file: %w", err)
		}
		if !exists {
			out.Skipped++
			out.SkipReasons["env_missing"]++
			continue
		}

		secretPath := filepath.Join(cfg.Secrets.Path, ref.Repository.Secret.Source)
		if input.DryRun {
			out.Updated++
			continue
		}

		if err := u.secrets.EncryptFile(ctx, envPath, secretPath); err != nil {
			out.Skipped++
			out.SkipReasons["encrypt_failed"]++
			out.Failures = append(out.Failures, AbsorbFailure{RepositoryURL: ref.Repository.URL, Step: "encrypt", Message: err.Error()})
			continue
		}
		out.Updated++
	}

	return out, nil
}
