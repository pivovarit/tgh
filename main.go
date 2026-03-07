package main

import (
	"flag"
	"fmt"
	"os"

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

	p := tea.NewProgram(ui.New(owners))
	if _, err := p.Run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
