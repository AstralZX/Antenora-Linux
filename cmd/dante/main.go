// Copyright (C) 2026 Antenora Linux contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

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

Repository:
  sync                    Update all repositories (official + DUR)
  search <term>           Search official and DUR package names/descriptions
  info <pkg>              Show package metadata
  dur                     List Dante User Repository packages
  dur-search <term>       Search the DUR only
  key                     Import the Antenora maintainer GPG key
  mirror                  Test and rank binary mirrors

Packages:
  install <pkg>           Install a package and its dependencies
  install -A <pkg>        Install from the DUR (user repository)
  remove <pkg>            Remove a package
  remove --orphans        Remove packages no longer required
  update                  Upgrade all installed packages
  clean                   Clean cached sources and build directories

Toolchain:
  toolchain               Bootstrap the Antenora build toolchain
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

	preferDur := false
	rest := args[1:]
	if len(rest) > 0 && rest[0] == "-A" {
		preferDur = true
		rest = rest[1:]
	}

	switch args[0] {
	case "sync":
		err = d.Sync()
	case "install":
		err = requireArg(rest, 0)
		if err == nil {
			err = d.Install(rest[0], preferDur)
		}
	case "remove":
		if len(rest) > 0 && rest[0] == "--orphans" {
			err = d.RemoveOrphans()
		} else {
			err = requireArg(rest, 0)
			if err == nil {
				err = d.Remove(rest[0])
			}
		}
	case "search":
		err = requireArg(rest, 0)
		if err == nil {
			err = d.Search(rest[0])
		}
	case "info":
		err = requireArg(rest, 0)
		if err == nil {
			err = d.Info(rest[0])
		}
	case "dur":
		err = d.DurList()
	case "dur-search":
		err = requireArg(rest, 0)
		if err == nil {
			err = d.DurSearch(rest[0])
		}
	case "key":
		err = d.ImportKey()
	case "mirror":
		err = d.TestMirrors()
	case "toolchain":
		err = d.Toolchain()
	case "gen-manifest":
		err = requireArg(rest, 0)
		if err == nil {
			err = d.GenManifest(rest[0])
		}
	case "update":
		err = d.Update()
	case "clean":
		err = d.Clean()
	case "compile":
		if len(rest) != 2 {
			err = fmt.Errorf("usage: dante compile <in.hell> <out.mk>")
		} else {
			err = d.CompileToMakefile(rest[0], rest[1])
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
		return fmt.Errorf("missing required argument")
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "dante: error: %v\n", err)
	os.Exit(1)
}
