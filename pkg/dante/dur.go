// Copyright (C) 2026 Antenora Linux contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package dante

import (
	"fmt"
	"sort"
	"strings"
)

// DurList lists every package in the Dante User Repository.
func (d *Dante) DurList() error {
	if err := d.BuildIndex(); err != nil {
		return err
	}
	names := make([]string, 0, len(d.Dur))
	for n := range d.Dur {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		pi := d.Dur[n]
		mark := ""
		if pi.Installed {
			mark = " [installed]"
		}
		fmt.Printf("%s/%s %s%s\n", pi.Name, pi.Version, pi.Description, mark)
	}
	fmt.Printf("\n%d package(s) in the DUR\n", len(names))
	return nil
}

// DurSearch searches DUR package names and descriptions.
func (d *Dante) DurSearch(term string) error {
	if err := d.BuildIndex(); err != nil {
		return err
	}
	term = strings.ToLower(term)
	names := make([]string, 0, len(d.Dur))
	for n := range d.Dur {
		names = append(names, n)
	}
	sort.Strings(names)
	found := 0
	for _, n := range names {
		pi := d.Dur[n]
		if strings.Contains(strings.ToLower(n), term) ||
			strings.Contains(strings.ToLower(pi.Description), term) {
			fmt.Printf("%s/%s %s\n", pi.Name, pi.Version, pi.Description)
			found++
		}
	}
	if found == 0 {
		fmt.Printf("No DUR packages match %q\n", term)
	}
	return nil
}
