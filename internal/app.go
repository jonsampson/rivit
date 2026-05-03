package internal

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jonsampson/rivit/internal/adapter"
	"github.com/jonsampson/rivit/internal/domain"
	"github.com/jonsampson/rivit/internal/usecase"
)

type App struct {
	cli        adapter.CLI
	configPath string
	in         io.Reader
	out        io.Writer
	errOut     io.Writer
}

func NewApp(out io.Writer, errOut io.Writer) (App, error) {
	configPath, err := defaultConfigPath()
	if err != nil {
		return App{}, err
	}

	return App{
		cli:        adapter.NewCLI(out),
		configPath: configPath,
		in:         os.Stdin,
		out:        out,
		errOut:     errOut,
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

	store := adapter.NewConfigFileStore(a.configPath)
	if cmd.ConfigPath != "" {
		store = adapter.NewConfigFileStore(cmd.ConfigPath)
	}
	gitDiscoverer := adapter.NewGitDiscoverer()
	validateProbe := adapter.NewValidateProbe()
	pathOps := adapter.NewPathOps()
	sops := adapter.NewSOPS()

	initUse := usecase.NewInit(store)
	addWorkspaceUse := usecase.NewAddWorkspace(store)
	addRepositoryUse := usecase.NewAddRepository(store)
	listWorkspaceUse := usecase.NewListWorkspace(store)
	listRepositoryUse := usecase.NewListRepository(store)
	removeWorkspaceUse := usecase.NewRemoveWorkspace(store)
	removeRepositoryUse := usecase.NewRemoveRepository(store)
	scanUse := usecase.NewScan(store, gitDiscoverer, pathOps, sops)
	validateWorkspaceUse := usecase.NewValidateWorkspace(store, validateProbe)
	validateRepositoryUse := usecase.NewValidateRepository(store, validateProbe)
	hydrateUse := usecase.NewHydrate(store, pathOps, gitDiscoverer, sops)
	absorbUse := usecase.NewAbsorb(store, pathOps, sops)

	switch cmd.Name {
	case "init":
		secretsPath, err := defaultSecretsPath()
		if err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 2
		}
		if err := initUse.Execute(ctx, usecase.InitInput{SecretsPath: secretsPath}); err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 2
		}
		fmt.Fprintln(a.out, "initialized rivit config")
		return 0
	case "workspace.add":
		if err := addWorkspaceUse.Execute(ctx, usecase.AddWorkspaceInput{Name: cmd.Args[0], Path: cmd.Args[1]}); err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(a.out, "added workspace %q\n", cmd.Args[0])
		return 0
	case "workspace.list":
		items, err := listWorkspaceUse.Execute(ctx)
		if err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 1
		}
		for _, item := range items {
			fmt.Fprintf(a.out, "%s\t%s\n", item.Name, item.Path)
		}
		return 0
	case "workspace.remove":
		if err := removeWorkspaceUse.Execute(ctx, usecase.RemoveWorkspaceInput{Name: cmd.Args[0]}); err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(a.out, "removed workspace %q\n", cmd.Args[0])
		return 0
	case "repo.add":
		repoURL, err := addRepositoryUse.Execute(ctx, usecase.AddRepositoryInput{URL: cmd.Args[0], Workspace: cmd.Args[1]})
		if err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(a.out, "added repo %q to workspace %q\n", repoURL, cmd.Args[1])
		return 0
	case "repo.list":
		items, err := listRepositoryUse.Execute(ctx)
		if err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 1
		}
		for _, item := range items {
			fmt.Fprintf(a.out, "%s\t%s\n", item.Workspace, item.URL)
		}
		return 0
	case "repo.remove":
		if err := removeRepositoryUse.Execute(ctx, usecase.RemoveRepositoryInput{ID: cmd.Args[0]}); err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(a.out, "removed repo %q\n", cmd.Args[0])
		return 0
	case "scan":
		dryRun := len(cmd.Args) > 2 && cmd.Args[2] == "dry-run"
		result, err := scanUse.Execute(ctx, usecase.ScanInput{Path: cmd.Args[0], Workspace: cmd.Args[1], DryRun: dryRun})
		if err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(a.out, "scan complete: discovered=%d added=%d absorbed=%d skipped=%d", result.Discovered, result.Added, result.Absorbed, result.Skipped)
		if dryRun {
			fmt.Fprintf(a.out, " (dry-run)")
		}
		fmt.Fprintln(a.out)
		printSkipReasons(a.out, result.SkipReasons)
		if len(result.Failures) > 0 {
			for _, failure := range result.Failures {
				fmt.Fprintf(a.errOut, "scan warning: repo=%s step=%s error=%s\n", failure.RepositoryURL, failure.Step, failure.Message)
			}
			return 1
		}
		return 0
	case "validate":
		return a.runValidate(ctx, cmd.Args, store, validateWorkspaceUse, validateRepositoryUse)
	case "hydrate":
		return a.runHydrate(ctx, cmd.Args, hydrateUse)
	case "absorb":
		return a.runAbsorb(ctx, cmd.Args, absorbUse)
	default:
		fmt.Fprintf(a.errOut, "error: unsupported command %q\n", cmd.Name)
		return 2
	}
}

func (a App) runAbsorb(ctx context.Context, args []string, absorbUse usecase.Absorb) int {
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

	if !input.DryRun && !input.Yes {
		confirmed, err := a.confirmAbsorb()
		if err != nil {
			fmt.Fprintf(a.errOut, "error: %v\n", err)
			return 2
		}
		if !confirmed {
			fmt.Fprintln(a.out, "absorb cancelled")
			return 0
		}
		input.Yes = true
	}

	out, err := absorbUse.Execute(ctx, input)
	if err != nil {
		fmt.Fprintf(a.errOut, "error: %v\n", err)
		return 2
	}

	fmt.Fprintf(a.out, "absorb complete: updated=%d skipped=%d", out.Updated, out.Skipped)
	if input.DryRun {
		fmt.Fprintf(a.out, " (dry-run)")
	}
	fmt.Fprintln(a.out)
	printSkipReasons(a.out, out.SkipReasons)
	if len(out.Failures) > 0 {
		for _, failure := range out.Failures {
			fmt.Fprintf(a.errOut, "absorb warning: repo=%s step=%s error=%s\n", failure.RepositoryURL, failure.Step, failure.Message)
		}
		return 1
	}
	return 0
}

func (a App) confirmAbsorb() (bool, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false, fmt.Errorf("check stdin: %w", err)
	}
	if (info.Mode() & os.ModeCharDevice) == 0 {
		return false, usecase.ErrAbsorbConfirmationRequired
	}

	fmt.Fprint(a.out, "absorb will overwrite encrypted secret files from local .env files. Continue? [y/N]: ")
	reader := bufio.NewReader(a.in)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func (a App) runHydrate(ctx context.Context, args []string, hydrateUse usecase.Hydrate) int {
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

	input.Progress = func(p usecase.HydrateProgress) {
		fmt.Fprintf(a.out, "hydrate progress: %d/%d repo=%s stage=%s\n", p.Current, p.Total, p.RepositoryURL, p.Stage)
	}

	out, err := hydrateUse.Execute(ctx, input)
	if err != nil {
		fmt.Fprintf(a.errOut, "error: %v\n", err)
		return 2
	}

	fmt.Fprintf(a.out, "hydrate complete: dirs=%d repos=%d secrets=%d skipped=%d", out.DirectoriesCreated, out.ReposCloned, out.SecretsMaterialized, out.Skipped)
	if input.DryRun {
		fmt.Fprintf(a.out, " (dry-run)")
	}
	fmt.Fprintln(a.out)
	printSkipReasons(a.out, out.SkipReasons)
	if len(out.Failures) > 0 {
		for _, failure := range out.Failures {
			fmt.Fprintf(a.errOut, "hydrate warning: repo=%s step=%s error=%s\n", failure.RepositoryURL, failure.Step, failure.Message)
		}
		return 1
	}
	return 0
}

func printSkipReasons(out io.Writer, reasons map[string]int) {
	if len(reasons) == 0 {
		return
	}
	keys := make([]string, 0, len(reasons))
	for key := range reasons {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(out, "skip: reason=%s count=%d\n", key, reasons[key])
	}
}

func (a App) runValidate(ctx context.Context, args []string, store adapter.ConfigFileStore, validateWorkspaceUse usecase.ValidateWorkspace, validateRepositoryUse usecase.ValidateRepository) int {
	issues := []string{}

	if len(args) == 0 {
		cfg, err := store.Load(ctx)
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
			wsIssues, err := validateWorkspaceUse.Execute(ctx, usecase.ValidateWorkspaceInput{WorkspaceName: name})
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
		cfg, err := store.Load(ctx)
		if err != nil {
			fmt.Fprintf(a.errOut, "error: load config: %v\n", err)
			return 2
		}

		if _, ok := cfg.Workspaces[target]; ok {
			wsIssues, err := validateWorkspaceUse.Execute(ctx, usecase.ValidateWorkspaceInput{WorkspaceName: target})
			if err != nil {
				fmt.Fprintf(a.errOut, "error: %v\n", err)
				return 2
			}
			for _, issue := range wsIssues {
				issues = append(issues, fmt.Sprintf("%s\t%s\t%s", issue.Scope, issue.Code, issue.Message))
			}
		} else if repositoryExistsInAnyWorkspace(cfg, target) {
			repoIssues, err := validateRepositoryUse.Execute(ctx, usecase.ValidateRepositoryInput{RepositoryID: target})
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

func repositoryExistsInAnyWorkspace(cfg domain.Config, repoURL string) bool {
	for _, ws := range cfg.Workspaces {
		for _, repo := range ws.Repos {
			if repo.URL == repoURL {
				return true
			}
		}
	}

	return false
}

func defaultSecretsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config directory: %w", err)
	}
	return filepath.Join(dir, "rivit", "secrets"), nil
}
