package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/pivovarit/tgh/internal/ui"
)

type multiFlag []string

func (f *multiFlag) String() string     { return fmt.Sprintf("%v", *f) }
func (f *multiFlag) Set(v string) error { *f = append(*f, v); return nil }

func main() {
	var owners multiFlag
	flag.Var(&owners, "owner", "limit to this owner/org (repeatable, e.g. -owner pivovarit -owner vavr-io)")
	flag.Parse()

	if err := checkGH(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.New(owners))
	if _, err := p.Run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func checkGH() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found. Install it from https://cli.github.com")
	}
	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		return fmt.Errorf("gh is not authenticated. Run 'gh auth login' first")
	}
	return nil
}
