package dante

import (
	"fmt"
	"os"
	"path/filepath"
)

// Update upgrades every installed package whose repository version is newer
// than the installed version.
func (d *Dante) Update() error {
	if err := d.BuildIndex(); err != nil {
		return err
	}
	installed, err := d.InstalledPackages()
	if err != nil {
		return err
	}
	updated := 0
	for _, name := range installed {
		pi, ok := d.Index[name]
		if !ok {
			fmt.Printf("!! %s is installed but missing from repository\n", name)
			continue
		}
		if pi.InstalledVersion != pi.Version {
			fmt.Printf(":: Updating %s %s -> %s\n", name, pi.InstalledVersion, pi.Version)
			if err := d.installOne(name); err != nil {
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
