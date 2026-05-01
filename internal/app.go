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
	cli                 adapter.CLI
	addWorkspaceUse     usecase.AddWorkspace
	addRepositoryUse    usecase.AddRepository
	listWorkspaceUse    usecase.ListWorkspace
	listRepositoryUse   usecase.ListRepository
	removeWorkspaceUse  usecase.RemoveWorkspace
	removeRepositoryUse usecase.RemoveRepository
	scanUse             usecase.Scan
	out                 io.Writer
	errOut              io.Writer
}

func NewApp(out io.Writer, errOut io.Writer) (App, error) {
	configPath, err := defaultConfigPath()
	if err != nil {
		return App{}, err
	}

	store := adapter.NewConfigFileStore(configPath)
	gitDiscoverer := adapter.NewGitDiscoverer()

	return App{
		cli:                 adapter.NewCLI(out),
		addWorkspaceUse:     usecase.NewAddWorkspace(store),
		addRepositoryUse:    usecase.NewAddRepository(store),
		listWorkspaceUse:    usecase.NewListWorkspace(store),
		listRepositoryUse:   usecase.NewListRepository(store),
		removeWorkspaceUse:  usecase.NewRemoveWorkspace(store),
		removeRepositoryUse: usecase.NewRemoveRepository(store),
		scanUse:             usecase.NewScan(store, gitDiscoverer),
		out:                 out,
		errOut:              errOut,
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
	case "repo.list":
		items, err := a.listRepositoryUse.Execute(ctx)
		if err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 1
		}
		for _, item := range items {
			fmt.Fprintf(a.out, "%s\t%s\n", item.ID, item.URL)
		}
		return 0
	case "repo.remove":
		if err := a.removeRepositoryUse.Execute(ctx, usecase.RemoveRepositoryInput{ID: cmd.Args[0]}); err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(a.out, "removed repo %q\n", cmd.Args[0])
		return 0
	case "scan":
		dryRun := len(cmd.Args) > 2 && cmd.Args[2] == "dry-run"
		result, err := a.scanUse.Execute(ctx, usecase.ScanInput{Path: cmd.Args[0], Workspace: cmd.Args[1], DryRun: dryRun})
		if err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(a.out, "scan complete: discovered=%d added=%d skipped=%d", result.Discovered, result.Added, result.Skipped)
		if dryRun {
			fmt.Fprintf(a.out, " (dry-run)")
		}
		fmt.Fprintln(a.out)
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
