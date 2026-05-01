package adapter

import "testing"

func TestCLIParse(t *testing.T) {
	cli := NewCLI(nil)

	t.Run("workspace add", func(t *testing.T) {
		cmd, err := cli.Parse([]string{"workspace", "add", "personal", "~/Code"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd.Name != "workspace.add" {
			t.Fatalf("unexpected command name: %s", cmd.Name)
		}
	})

	t.Run("workspace list", func(t *testing.T) {
		cmd, err := cli.Parse([]string{"workspace", "list"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd.Name != "workspace.list" {
			t.Fatalf("unexpected command name: %s", cmd.Name)
		}
	})

	t.Run("workspace remove", func(t *testing.T) {
		cmd, err := cli.Parse([]string{"workspace", "remove", "personal"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd.Name != "workspace.remove" {
			t.Fatalf("unexpected command name: %s", cmd.Name)
		}
	})

	t.Run("repo add", func(t *testing.T) {
		cmd, err := cli.Parse([]string{"repo", "add", "git@github.com:jonsampson/rivit.git", "--workspace", "personal"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd.Name != "repo.add" {
			t.Fatalf("unexpected command name: %s", cmd.Name)
		}
		if len(cmd.Args) != 2 || cmd.Args[1] != "personal" {
			t.Fatalf("unexpected command args: %+v", cmd.Args)
		}
	})

	t.Run("repo list", func(t *testing.T) {
		cmd, err := cli.Parse([]string{"repo", "list"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd.Name != "repo.list" {
			t.Fatalf("unexpected command name: %s", cmd.Name)
		}
	})

	t.Run("repo remove", func(t *testing.T) {
		cmd, err := cli.Parse([]string{"repo", "remove", "github.com/org/repo"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd.Name != "repo.remove" {
			t.Fatalf("unexpected command name: %s", cmd.Name)
		}
		if len(cmd.Args) != 1 || cmd.Args[0] != "github.com/org/repo" {
			t.Fatalf("unexpected command args: %+v", cmd.Args)
		}
	})
}
