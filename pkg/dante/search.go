// Copyright (C) 2026 Antenora Linux contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package dante

import (
	"fmt"
	"sort"
	"strings"
)

// Search scans official and DUR package names and descriptions for term and
// prints matches, tagging their source.
func (d *Dante) Search(term string) error {
	if err := d.BuildIndex(); err != nil {
		return err
	}
	term = strings.ToLower(term)
	type hit struct {
		pi     *PackageInfo
		source Source
	}
	var hits []hit
	for n, pi := range d.Index {
		if match(term, n, pi.Description) {
			hits = append(hits, hit{pi, SourceOfficial})
		}
	}
	for n, pi := range d.Dur {
		if match(term, n, pi.Description) {
			hits = append(hits, hit{pi, SourceDUR})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		return hits[i].pi.Name < hits[j].pi.Name
	})
	for _, h := range hits {
		mark := ""
		if h.pi.Installed {
			mark = " [installed]"
		}
		tag := ""
		if h.source == SourceDUR {
			tag = " (DUR)"
		}
		fmt.Printf("%s/%s %s%s%s\n", h.pi.Name, h.pi.Version, h.pi.Description, tag, mark)
	}
	if len(hits) == 0 {
		fmt.Printf("No packages match %q\n", term)
	}
	return nil
}

func match(term, name, desc string) bool {
	return strings.Contains(strings.ToLower(name), term) ||
		strings.Contains(strings.ToLower(desc), term)
}

// Info prints full metadata, dependencies and binary availability for a
// package, searching the DUR when the package is only present there.
func (d *Dante) Info(name string) error {
	if err := d.BuildIndex(); err != nil {
		return err
	}
	pi, src, ok := d.Lookup(name, true)
	if !ok {
		return fmt.Errorf("package %q not found", name)
	}
	fmt.Printf("Name        : %s\n", pi.Name)
	fmt.Printf("Version     : %s\n", pi.Version)
	fmt.Printf("Source      : %s\n", src)
	fmt.Printf("Description : %s\n", pi.Description)
	fmt.Printf("Recipe      : %s\n", pi.RecipePath)
	if len(pi.Depends) > 0 {
		fmt.Printf("Depends     : %s\n", strings.Join(pi.Depends, " "))
	} else {
		fmt.Println("Depends     : (none)")
	}
	if pi.SourceURL != "" {
		fmt.Printf("Upstream    : %s\n", pi.SourceURL)
	}
	if pi.BinaryURL != "" {
		fmt.Printf("Binary      : %s\n", pi.BinaryURL)
		fmt.Printf("Binary SHA  : %s\n", pi.BinarySHA256)
		fmt.Printf("Binary size : %s\n", pi.BinarySize)
	} else {
		fmt.Println("Binary      : (source only)")
	}
	if pi.Installed {
		fmt.Printf("Installed   : %s\n", pi.InstalledVersion)
	} else {
		fmt.Println("Installed   : no")
	}
	return nil
}
