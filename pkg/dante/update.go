// Copyright (C) 2026 Antenora Linux contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package dante

import (
	"fmt"
	"os"
	"path/filepath"
)

// Update syncs repositories and upgrades every installed package whose
// repository version is newer than the installed version.
func (d *Dante) Update() error {
	if err := d.Sync(); err != nil {
		return err
	}
	if err := d.BuildIndex(); err != nil {
		return err
	}
	installed, err := d.InstalledPackages()
	if err != nil {
		return err
	}
	updated := 0
	for _, name := range installed {
		pi, _, ok := d.Lookup(name, true)
		if !ok {
			fmt.Printf("!! %s is installed but missing from repositories\n", name)
			continue
		}
		if pi.InstalledVersion == "" {
			continue
		}
		if CompareVersions(pi.Version, pi.InstalledVersion) > 0 {
			fmt.Printf(":: Updating %s %s -> %s\n", name, pi.InstalledVersion, pi.Version)
			if err := d.installOne(name, pi.Source == SourceDUR); err != nil {
				fmt.Printf("!! Failed to update %s: %v\n", name, err)
				continue
			}
			updated++
		}
	}
	if updated == 0 {
		fmt.Println(":: Nothing to do, system is current")
	} else {
		fmt.Printf(":: Updated %d package(s)\n", updated)
	}
	return nil
}

// Clean removes cached source tarballs, downloaded binary archives and stale
// build directories.
func (d *Dante) Clean() error {
	targets := []string{
		filepath.Join(d.Config.CacheDir, "sources"),
		filepath.Join(d.Config.CacheDir, "packages"),
		d.Config.BuildDir,
	}
	for _, t := range targets {
		if err := os.RemoveAll(t); err != nil {
			return err
		}
		fmt.Printf(":: Cleaned %s\n", t)
	}
	return nil
}
