package usecase

import (
	"context"

	"github.com/jonsampson/rivit/internal/domain"
)

type memoryConfigStore struct {
	config domain.Config
	err    error
}

func (m *memoryConfigStore) Load(context.Context) (domain.Config, error) {
	if m.err != nil {
		return domain.Config{}, m.err
	}
	return m.config, nil
}

func (m *memoryConfigStore) Save(_ context.Context, cfg domain.Config) error {
	if m.err != nil {
		return m.err
	}
	m.config = cfg
	return nil
}
