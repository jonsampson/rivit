package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jonsampson/rivit/internal/adapter"
	"github.com/jonsampson/rivit/internal/usecase"
)

type App struct {
	cli                adapter.CLI
	addWorkspaceUse    usecase.AddWorkspace
	addRepositoryUse   usecase.AddRepository
	listWorkspaceUse   usecase.ListWorkspace
	removeWorkspaceUse usecase.RemoveWorkspace
	out                io.Writer
	errOut             io.Writer
}

func NewApp(out io.Writer, errOut io.Writer) (App, error) {
	configPath, err := defaultConfigPath()
	if err != nil {
		return App{}, err
	}

	store := adapter.NewConfigFileStore(configPath)

	return App{
		cli:                adapter.NewCLI(out),
		addWorkspaceUse:    usecase.NewAddWorkspace(store),
		addRepositoryUse:   usecase.NewAddRepository(store),
		listWorkspaceUse:   usecase.NewListWorkspace(store),
		removeWorkspaceUse: usecase.NewRemoveWorkspace(store),
		out:                out,
		errOut:             errOut,
	}, nil
}

func (a App) Run(args []string) int {
	ctx := context.Background()

	cmd, err := a.cli.Parse(args)
	if err != nil {
		if errors.Is(err, adapter.ErrHelp) {
			a.cli.PrintHelp()
			return 0
		}
		fmt.Fprintf(a.errOut, "error: %v\n", err)
		a.cli.PrintHelp()
		return 2
	}

	switch cmd.Name {
	case "workspace.add":
		if err := a.addWorkspaceUse.Execute(ctx, usecase.AddWorkspaceInput{Name: cmd.Args[0], Path: cmd.Args[1]}); err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(a.out, "added workspace %q\n", cmd.Args[0])
		return 0
	case "workspace.list":
		items, err := a.listWorkspaceUse.Execute(ctx)
		if err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 1
		}
		for _, item := range items {
			fmt.Fprintf(a.out, "%s\t%s\n", item.Name, item.Path)
		}
		return 0
	case "workspace.remove":
		if err := a.removeWorkspaceUse.Execute(ctx, usecase.RemoveWorkspaceInput{Name: cmd.Args[0]}); err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(a.out, "removed workspace %q\n", cmd.Args[0])
		return 0
	case "repo.add":
		repoID, err := a.addRepositoryUse.Execute(ctx, usecase.AddRepositoryInput{URL: cmd.Args[0], Workspace: cmd.Args[1]})
		if err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(a.out, "added repo %q to workspace %q\n", repoID, cmd.Args[1])
		return 0
	default:
		fmt.Fprintf(a.errOut, "error: unsupported command %q\n", cmd.Name)
		return 2
	}
}

func defaultConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config directory: %w", err)
	}
	return filepath.Join(dir, "rivit", "config.yaml"), nil
}
