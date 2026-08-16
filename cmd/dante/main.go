// Command dante is the Antenora package manager. It manages packages described
// by Hell recipes, resolving dependencies, compiling from source or installing
// signed binary fallbacks, and tracking every file it installs.
package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/antenora/dante/pkg/dante"
)

const version = "1.0.0"

func usage() {
	fmt.Fprintf(os.Stderr, `Dante %s - the Antenora package manager
Abandon all hope, ye who enter here.

Usage: dante <command> [arguments]

Commands:
  sync                    Update the central package repository
  install <pkg>           Install a package and its dependencies
  remove <pkg>            Remove a package
  remove --orphans        Remove packages no longer required
  search <term>           Search package names and descriptions
  info <pkg>              Show package metadata
  update                  Upgrade all installed packages
  clean                   Clean cached sources and build directories
  compile <in.hell> <out> Compile a Hell recipe to a standalone Makefile

Options:
  --help                  Show this help
  --version               Show the version
`, version)
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	switch args[0] {
	case "--version", "-V":
		fmt.Printf("Dante %s (go %s)\n", version, runtime.Version())
		return
	case "--help", "-h", "help":
		usage()
		return
	}

	cfg, err := dante.LoadConfig("/etc/dante/dante.conf")
	if err != nil {
		fatal(err)
	}
	d := dante.New(cfg)

	switch args[0] {
	case "sync":
		err = d.Sync()
	case "install":
		err = requireArg(args, 1)
		if err == nil {
			err = d.Install(args[1])
		}
	case "remove":
		if len(args) > 1 && args[1] == "--orphans" {
			err = d.RemoveOrphans()
		} else {
			err = requireArg(args, 1)
			if err == nil {
				err = d.Remove(args[1])
			}
		}
	case "search":
		err = requireArg(args, 1)
		if err == nil {
			err = d.Search(args[1])
		}
	case "info":
		err = requireArg(args, 1)
		if err == nil {
			err = d.Info(args[1])
		}
	case "update":
		err = d.Update()
	case "clean":
		err = d.Clean()
	case "compile":
		if len(args) != 3 {
			err = fmt.Errorf("usage: dante compile <in.hell> <out.mk>")
		} else {
			err = d.CompileToMakefile(args[1], args[2])
		}
	default:
		err = fmt.Errorf("unknown command %q", args[0])
	}
	if err != nil {
		fatal(err)
	}
}

func requireArg(args []string, n int) error {
	if len(args) <= n {
		return fmt.Errorf("command %q requires an argument", args[0])
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "dante: error: %v\n", err)
	os.Exit(1)
}
