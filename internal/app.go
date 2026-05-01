package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/jonsampson/rivit/internal/adapter"
	"github.com/jonsampson/rivit/internal/usecase"
)

type App struct {
	cli                   adapter.CLI
	addWorkspaceUse       usecase.AddWorkspace
	initUse               usecase.Init
	addRepositoryUse      usecase.AddRepository
	listWorkspaceUse      usecase.ListWorkspace
	listRepositoryUse     usecase.ListRepository
	removeWorkspaceUse    usecase.RemoveWorkspace
	removeRepositoryUse   usecase.RemoveRepository
	scanUse               usecase.Scan
	validateWorkspaceUse  usecase.ValidateWorkspace
	validateRepositoryUse usecase.ValidateRepository
	hydrateUse            usecase.Hydrate
	absorbUse             usecase.Absorb
	configStore           adapter.ConfigFileStore
	out                   io.Writer
	errOut                io.Writer
}

func NewApp(out io.Writer, errOut io.Writer) (App, error) {
	configPath, err := defaultConfigPath()
	if err != nil {
		return App{}, err
	}

	store := adapter.NewConfigFileStore(configPath)
	gitDiscoverer := adapter.NewGitDiscoverer()
	validateProbe := adapter.NewValidateProbe()
	pathOps := adapter.NewPathOps()
	sops := adapter.NewSOPS()

	return App{
		cli:                   adapter.NewCLI(out),
		initUse:               usecase.NewInit(store),
		addWorkspaceUse:       usecase.NewAddWorkspace(store),
		addRepositoryUse:      usecase.NewAddRepository(store),
		listWorkspaceUse:      usecase.NewListWorkspace(store),
		listRepositoryUse:     usecase.NewListRepository(store),
		removeWorkspaceUse:    usecase.NewRemoveWorkspace(store),
		removeRepositoryUse:   usecase.NewRemoveRepository(store),
		scanUse:               usecase.NewScan(store, gitDiscoverer),
		validateWorkspaceUse:  usecase.NewValidateWorkspace(store, validateProbe),
		validateRepositoryUse: usecase.NewValidateRepository(store, validateProbe),
		hydrateUse:            usecase.NewHydrate(store, pathOps, gitDiscoverer, sops),
		absorbUse:             usecase.NewAbsorb(store, pathOps, sops),
		configStore:           store,
		out:                   out,
		errOut:                errOut,
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
	case "init":
		secretsPath, err := defaultSecretsPath()
		if err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 2
		}
		if err := a.initUse.Execute(ctx, usecase.InitInput{SecretsPath: secretsPath}); err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 2
		}
		fmt.Fprintln(a.out, "initialized rivit config")
		return 0
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
	case "validate":
		return a.runValidate(ctx, cmd.Args)
	case "hydrate":
		return a.runHydrate(ctx, cmd.Args)
	case "absorb":
		return a.runAbsorb(ctx, cmd.Args)
	default:
		fmt.Fprintf(a.errOut, "error: unsupported command %q\n", cmd.Name)
		return 2
	}
}

func (a App) runAbsorb(ctx context.Context, args []string) int {
	input := usecase.AbsorbInput{}
	for _, arg := range args {
		switch arg {
		case "dry-run":
			input.DryRun = true
		case "yes":
			input.Yes = true
		default:
			if input.Target == "" {
				input.Target = arg
			}
		}
	}

	out, err := a.absorbUse.Execute(ctx, input)
	if err != nil {
		fmt.Fprintf(a.errOut, "error: %v\n", err)
		return 2
	}

	fmt.Fprintf(a.out, "absorb complete: updated=%d skipped=%d", out.Updated, out.Skipped)
	if input.DryRun {
		fmt.Fprintf(a.out, " (dry-run)")
	}
	fmt.Fprintln(a.out)
	return 0
}

func (a App) runHydrate(ctx context.Context, args []string) int {
	input := usecase.HydrateInput{}
	for _, arg := range args {
		switch arg {
		case "dry-run":
			input.DryRun = true
		case "repos-only":
			input.ReposOnly = true
		case "secrets-only":
			input.SecretsOnly = true
		case "force-env":
			input.ForceEnv = true
		default:
			if input.Target == "" {
				input.Target = arg
			}
		}
	}

	out, err := a.hydrateUse.Execute(ctx, input)
	if err != nil {
		fmt.Fprintf(a.errOut, "error: %v\n", err)
		return 2
	}

	fmt.Fprintf(a.out, "hydrate complete: dirs=%d repos=%d secrets=%d skipped=%d", out.DirectoriesCreated, out.ReposCloned, out.SecretsMaterialized, out.Skipped)
	if input.DryRun {
		fmt.Fprintf(a.out, " (dry-run)")
	}
	fmt.Fprintln(a.out)
	return 0
}

func (a App) runValidate(ctx context.Context, args []string) int {
	issues := []string{}

	if len(args) == 0 {
		cfg, err := a.configStore.Load(ctx)
		if err != nil {
			fmt.Fprintf(a.errOut, "error: load config: %v\n", err)
			return 2
		}

		names := make([]string, 0, len(cfg.Workspaces))
		for name := range cfg.Workspaces {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			wsIssues, err := a.validateWorkspaceUse.Execute(ctx, usecase.ValidateWorkspaceInput{WorkspaceName: name})
			if err != nil {
				fmt.Fprintf(a.errOut, "error: %v\n", err)
				return 2
			}
			for _, issue := range wsIssues {
				issues = append(issues, fmt.Sprintf("%s\t%s\t%s", issue.Scope, issue.Code, issue.Message))
			}
		}
	} else {
		target := args[0]
		cfg, err := a.configStore.Load(ctx)
		if err != nil {
			fmt.Fprintf(a.errOut, "error: load config: %v\n", err)
			return 2
		}

		if _, ok := cfg.Workspaces[target]; ok {
			wsIssues, err := a.validateWorkspaceUse.Execute(ctx, usecase.ValidateWorkspaceInput{WorkspaceName: target})
			if err != nil {
				fmt.Fprintf(a.errOut, "error: %v\n", err)
				return 2
			}
			for _, issue := range wsIssues {
				issues = append(issues, fmt.Sprintf("%s\t%s\t%s", issue.Scope, issue.Code, issue.Message))
			}
		} else if _, ok := cfg.Repos[target]; ok {
			repoIssues, err := a.validateRepositoryUse.Execute(ctx, usecase.ValidateRepositoryInput{RepositoryID: target})
			if err != nil {
				fmt.Fprintf(a.errOut, "error: %v\n", err)
				return 2
			}
			for _, issue := range repoIssues {
				issues = append(issues, fmt.Sprintf("%s\t%s\t%s", issue.Scope, issue.Code, issue.Message))
			}
		} else {
			fmt.Fprintf(a.errOut, "error: target not found: %s\n", target)
			return 2
		}
	}

	if len(issues) == 0 {
		fmt.Fprintln(a.out, "valid")
		return 0
	}

	for _, line := range issues {
		fmt.Fprintln(a.out, line)
	}
	return 1
}

func defaultConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config directory: %w", err)
	}
	return filepath.Join(dir, "rivit", "config.yaml"), nil
}

func defaultSecretsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config directory: %w", err)
	}
	return filepath.Join(dir, "rivit", "secrets"), nil
}
