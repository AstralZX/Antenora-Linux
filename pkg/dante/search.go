package dante

import (
	"fmt"
	"sort"
	"strings"
)

// Search scans package names and descriptions for term and prints matches.
func (d *Dante) Search(term string) error {
	if err := d.BuildIndex(); err != nil {
		return err
	}
	term = strings.ToLower(term)
	names := make([]string, 0, len(d.Index))
	for n := range d.Index {
		names = append(names, n)
	}
	sort.Strings(names)

	found := 0
	for _, n := range names {
		pi := d.Index[n]
		if strings.Contains(strings.ToLower(n), term) ||
			strings.Contains(strings.ToLower(pi.Description), term) {
			mark := ""
			if pi.Installed {
				mark = " [installed]"
			}
			fmt.Printf("%s/%s %s%s\n", pi.Name, pi.Version, pi.Description, mark)
			found++
		}
	}
	if found == 0 {
		fmt.Printf("No packages match %q\n", term)
	}
	return nil
}

// Info prints full metadata, dependencies and binary availability for a package.
func (d *Dante) Info(name string) error {
	if err := d.BuildIndex(); err != nil {
		return err
	}
	pi, ok := d.Index[name]
	if !ok {
		return fmt.Errorf("package %q not found", name)
	}
	fmt.Printf("Name        : %s\n", pi.Name)
	fmt.Printf("Version     : %s\n", pi.Version)
	fmt.Printf("Description : %s\n", pi.Description)
	fmt.Printf("Recipe      : %s\n", pi.RecipePath)
	if len(pi.Depends) > 0 {
		fmt.Printf("Depends     : %s\n", strings.Join(pi.Depends, " "))
	} else {
		fmt.Println("Depends     : (none)")
	}
	if pi.Source != "" {
		fmt.Printf("Source      : %s\n", pi.Source)
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
