package adapter

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

var ErrHelp = errors.New("help requested")

type Command struct {
	Name string
	Args []string
}

type CLI struct {
	out io.Writer
}

func NewCLI(out io.Writer) CLI {
	return CLI{out: out}
}

func (c CLI) Parse(args []string) (Command, error) {
	if len(args) == 0 {
		return Command{}, ErrHelp
	}

	switch args[0] {
	case "workspace":
		return c.parseWorkspace(args[1:])
	case "repo":
		return c.parseRepo(args[1:])
	case "scan":
		return c.parseScan(args[1:])
	case "validate":
		return c.parseValidate(args[1:])
	case "hydrate":
		return c.parseHydrate(args[1:])
	case "absorb":
		return c.parseAbsorb(args[1:])
	case "help", "--help", "-h":
		return Command{}, ErrHelp
	default:
		return Command{}, fmt.Errorf("unknown command: %s", args[0])
	}
}

func (c CLI) parseAbsorb(args []string) (Command, error) {
	target := ""
	dryRun := false
	yes := false

	for i := 0; i < len(args); i++ {
		tok := args[i]
		switch tok {
		case "--dry-run":
			dryRun = true
		case "--yes":
			yes = true
		default:
			if strings.HasPrefix(tok, "--") {
				return Command{}, fmt.Errorf("usage: rivit absorb [workspace-or-repo] [--dry-run] [--yes]")
			}
			if target != "" {
				return Command{}, fmt.Errorf("usage: rivit absorb [workspace-or-repo] [--dry-run] [--yes]")
			}
			target = tok
		}
	}

	cmd := Command{Name: "absorb"}
	if target != "" {
		cmd.Args = append(cmd.Args, target)
	}
	if dryRun {
		cmd.Args = append(cmd.Args, "dry-run")
	}
	if yes {
		cmd.Args = append(cmd.Args, "yes")
	}
	return cmd, nil
}

func (c CLI) parseHydrate(args []string) (Command, error) {
	target := ""
	dryRun := false
	reposOnly := false
	secretsOnly := false
	forceEnv := false

	for i := 0; i < len(args); i++ {
		tok := args[i]
		switch tok {
		case "--dry-run":
			dryRun = true
		case "--repos-only":
			reposOnly = true
		case "--secrets-only":
			secretsOnly = true
		case "--force-env":
			forceEnv = true
		default:
			if strings.HasPrefix(tok, "--") {
				return Command{}, fmt.Errorf("usage: rivit hydrate [workspace-or-repo] [--dry-run] [--repos-only] [--secrets-only] [--force-env]")
			}
			if target != "" {
				return Command{}, fmt.Errorf("usage: rivit hydrate [workspace-or-repo] [--dry-run] [--repos-only] [--secrets-only] [--force-env]")
			}
			target = tok
		}
	}

	if reposOnly && secretsOnly {
		return Command{}, fmt.Errorf("usage: rivit hydrate [workspace-or-repo] [--dry-run] [--repos-only] [--secrets-only] [--force-env]")
	}

	cmd := Command{Name: "hydrate"}
	if target != "" {
		cmd.Args = append(cmd.Args, target)
	}
	if dryRun {
		cmd.Args = append(cmd.Args, "dry-run")
	}
	if reposOnly {
		cmd.Args = append(cmd.Args, "repos-only")
	}
	if secretsOnly {
		cmd.Args = append(cmd.Args, "secrets-only")
	}
	if forceEnv {
		cmd.Args = append(cmd.Args, "force-env")
	}

	return cmd, nil
}

func (c CLI) parseValidate(args []string) (Command, error) {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return Command{}, err
	}

	parsed := fs.Args()
	if len(parsed) > 1 {
		return Command{}, fmt.Errorf("usage: rivit validate [workspace-or-repo]")
	}

	if len(parsed) == 0 {
		return Command{Name: "validate"}, nil
	}

	return Command{Name: "validate", Args: parsed}, nil
}

func (c CLI) parseScan(args []string) (Command, error) {
	var path string
	var workspace string
	dryRun := false

	for i := 0; i < len(args); i++ {
		tok := args[i]
		switch tok {
		case "--workspace":
			if i+1 >= len(args) {
				return Command{}, fmt.Errorf("usage: rivit scan <path> --workspace <name> [--dry-run]")
			}
			workspace = args[i+1]
			i++
		case "--dry-run":
			dryRun = true
		default:
			if strings.HasPrefix(tok, "--") {
				return Command{}, fmt.Errorf("usage: rivit scan <path> --workspace <name> [--dry-run]")
			}
			if path != "" {
				return Command{}, fmt.Errorf("usage: rivit scan <path> --workspace <name> [--dry-run]")
			}
			path = tok
		}
	}

	if path == "" || workspace == "" {
		return Command{}, fmt.Errorf("usage: rivit scan <path> --workspace <name> [--dry-run]")
	}

	cmd := Command{Name: "scan", Args: []string{path, workspace}}
	if dryRun {
		cmd.Args = append(cmd.Args, "dry-run")
	}

	return cmd, nil
}

func (c CLI) parseRepo(args []string) (Command, error) {
	if len(args) == 0 {
		return Command{}, fmt.Errorf("repo requires a subcommand")
	}

	switch args[0] {
	case "add":
		var repoURL string
		var workspace string
		tokens := args[1:]
		for i := 0; i < len(tokens); i++ {
			tok := tokens[i]
			if tok == "--workspace" {
				if i+1 >= len(tokens) {
					return Command{}, fmt.Errorf("usage: rivit repo add <url> --workspace <name>")
				}
				workspace = tokens[i+1]
				i++
				continue
			}
			if strings.HasPrefix(tok, "--") {
				return Command{}, fmt.Errorf("usage: rivit repo add <url> --workspace <name>")
			}
			if repoURL != "" {
				return Command{}, fmt.Errorf("usage: rivit repo add <url> --workspace <name>")
			}
			repoURL = tok
		}

		if repoURL == "" || workspace == "" {
			return Command{}, fmt.Errorf("usage: rivit repo add <url> --workspace <name>")
		}
		return Command{Name: "repo.add", Args: []string{repoURL, workspace}}, nil
	case "list":
		fs := flag.NewFlagSet("repo list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		if err := fs.Parse(args[1:]); err != nil {
			return Command{}, err
		}
		if len(fs.Args()) != 0 {
			return Command{}, fmt.Errorf("usage: rivit repo list")
		}
		return Command{Name: "repo.list"}, nil
	case "remove":
		fs := flag.NewFlagSet("repo remove", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		if err := fs.Parse(args[1:]); err != nil {
			return Command{}, err
		}
		parsed := fs.Args()
		if len(parsed) != 1 {
			return Command{}, fmt.Errorf("usage: rivit repo remove <repo-id>")
		}
		return Command{Name: "repo.remove", Args: parsed}, nil
	default:
		return Command{}, fmt.Errorf("unknown repo subcommand: %s", args[0])
	}
}

func (c CLI) parseWorkspace(args []string) (Command, error) {
	if len(args) == 0 {
		return Command{}, fmt.Errorf("workspace requires a subcommand")
	}

	sub := args[0]
	switch sub {
	case "add":
		fs := flag.NewFlagSet("workspace add", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		if err := fs.Parse(args[1:]); err != nil {
			return Command{}, err
		}
		parsed := fs.Args()
		if len(parsed) != 2 {
			return Command{}, fmt.Errorf("usage: rivit workspace add <name> <path>")
		}
		return Command{Name: "workspace.add", Args: parsed}, nil
	case "list":
		fs := flag.NewFlagSet("workspace list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		if err := fs.Parse(args[1:]); err != nil {
			return Command{}, err
		}
		if len(fs.Args()) != 0 {
			return Command{}, fmt.Errorf("usage: rivit workspace list")
		}
		return Command{Name: "workspace.list"}, nil
	case "remove":
		fs := flag.NewFlagSet("workspace remove", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		if err := fs.Parse(args[1:]); err != nil {
			return Command{}, err
		}
		parsed := fs.Args()
		if len(parsed) != 1 {
			return Command{}, fmt.Errorf("usage: rivit workspace remove <name>")
		}
		return Command{Name: "workspace.remove", Args: parsed}, nil
	default:
		return Command{}, fmt.Errorf("unknown workspace subcommand: %s", sub)
	}
}

func (c CLI) PrintHelp() {
	fmt.Fprintln(c.out, "rivit workspace add <name> <path>")
	fmt.Fprintln(c.out, "rivit workspace list")
	fmt.Fprintln(c.out, "rivit workspace remove <name>")
	fmt.Fprintln(c.out, "rivit repo add <url> --workspace <name>")
	fmt.Fprintln(c.out, "rivit repo list")
	fmt.Fprintln(c.out, "rivit repo remove <repo-id>")
	fmt.Fprintln(c.out, "rivit scan <path> --workspace <name> [--dry-run]")
	fmt.Fprintln(c.out, "rivit validate [workspace-or-repo]")
	fmt.Fprintln(c.out, "rivit hydrate [workspace-or-repo] [--dry-run] [--repos-only] [--secrets-only] [--force-env]")
	fmt.Fprintln(c.out, "rivit absorb [workspace-or-repo] [--dry-run] [--yes]")
}
