package internal

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunInitUsesConfigOverridePath(t *testing.T) {
	xdgHome := t.TempDir()
	overrideDir := t.TempDir()
	overridePath := filepath.Join(overrideDir, "rivit-test-config.yaml")

	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	var out bytes.Buffer
	var errOut bytes.Buffer
	app, err := NewApp(&out, &errOut)
	if err != nil {
		t.Fatalf("unexpected app construction error: %v", err)
	}

	exitCode := app.Run([]string{"--config", overridePath, "init"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", exitCode, errOut.String())
	}

	if _, err := os.Stat(overridePath); err != nil {
		t.Fatalf("expected override config file to exist: %v", err)
	}

	defaultPath := filepath.Join(xdgHome, "rivit", "config.yaml")
	if _, err := os.Stat(defaultPath); !os.IsNotExist(err) {
		t.Fatalf("expected default config path to remain untouched, stat err: %v", err)
	}
}
