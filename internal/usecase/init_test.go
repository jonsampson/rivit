package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/jonsampson/rivit/internal/domain"
)

type memoryInitStore struct {
	exists bool
	err    error
	saved  domain.Config
}

func (m *memoryInitStore) Exists(context.Context) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.exists, nil
}

func (m *memoryInitStore) Save(_ context.Context, cfg domain.Config) error {
	if m.err != nil {
		return m.err
	}
	m.saved = cfg
	return nil
}

func TestInitExecute(t *testing.T) {
	t.Run("initializes config", func(t *testing.T) {
		store := &memoryInitStore{}
		uc := NewInit(store)
		err := uc.Execute(context.Background(), InitInput{SecretsPath: "/cfg/rivit/secrets"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if store.saved.Version != 1 || store.saved.Secrets.Provider != "sops" {
			t.Fatalf("unexpected saved config: %+v", store.saved)
		}
	})

	t.Run("already initialized", func(t *testing.T) {
		store := &memoryInitStore{exists: true}
		uc := NewInit(store)
		err := uc.Execute(context.Background(), InitInput{SecretsPath: "/cfg/rivit/secrets"})
		if !errors.Is(err, ErrInitAlreadyInitialized) {
			t.Fatalf("expected ErrInitAlreadyInitialized, got %v", err)
		}
	})
}
