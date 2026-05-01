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
	case "help", "--help", "-h":
		return Command{}, ErrHelp
	default:
		return Command{}, fmt.Errorf("unknown command: %s", args[0])
	}
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
}
