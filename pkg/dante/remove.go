// Copyright (C) 2026 Antenora Linux contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package dante

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Remove uninstalls a package, warning about any installed packages that still
// depend on it. Files are removed deepest-first so directories left empty are
// pruned by their parents.
func (d *Dante) Remove(name string) error {
	rec, err := d.loadRecord(name)
	if err != nil {
		return fmt.Errorf("package %q is not installed", name)
	}

	installed, _ := d.InstalledPackages()
	var dependents []string
	for _, p := range installed {
		if p == name {
			continue
		}
		r, err := d.loadRecord(p)
		if err != nil {
			continue
		}
		for _, dep := range r.Depends {
			if dep == name {
				dependents = append(dependents, p)
				break
			}
		}
	}
	if len(dependents) > 0 {
		fmt.Printf("!! Warning: %s is still required by: %s\n", name, strings.Join(dependents, ", "))
	}

	removed := 0
	sort.SliceStable(rec.Files, func(i, j int) bool {
		return strings.Count(rec.Files[i], "/") > strings.Count(rec.Files[j], "/")
	})
	for _, rel := range rec.Files {
		target := filepath.Join(d.Config.Root, rel)
		info, err := os.Lstat(target)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		if err := os.Remove(target); err != nil {
			fmt.Printf("!! Failed to remove %s: %v\n", target, err)
			continue
		}
		removed++
	}

	if err := os.Remove(d.recordPath(name)); err != nil {
		return err
	}
	fmt.Printf(":: Removed %s %s (%d files)\n", name, rec.Version, removed)

	if !d.Config.KeepDepsEnabled() {
		fmt.Println(":: Run 'dante remove --orphans' to clean now-unused dependencies")
	}
	return nil
}

// RemoveOrphans removes packages that are no longer required by anything.
func (d *Dante) RemoveOrphans() error {
	installed, err := d.InstalledPackages()
	if err != nil {
		return err
	}
	required := map[string]bool{}
	for _, p := range installed {
		rec, err := d.loadRecord(p)
		if err != nil {
			continue
		}
		for _, dep := range rec.Depends {
			required[dep] = true
		}
	}
	for _, p := range installed {
		if !required[p] {
			if err := d.Remove(p); err != nil {
				return err
			}
		}
	}
	return nil
}
