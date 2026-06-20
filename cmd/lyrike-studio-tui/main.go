package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/duckviet/lyrike-studio-tui/internal/version"
)

func main() {
	versionRequested := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *versionRequested {
		fmt.Println(version.Label())
		return
	}

	fmt.Fprintln(os.Stderr, "lyrike-studio-tui: TUI implementation has not started yet; use --version for the current smoke surface")
	os.Exit(2)
}
