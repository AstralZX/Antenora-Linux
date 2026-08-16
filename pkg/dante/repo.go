// Copyright (C) 2026 Antenora Linux contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package dante

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manifest mirrors a package repository's manifest.yaml.
type Manifest struct {
	Packages []ManifestPackage `yaml:"packages"`
}

// ManifestPackage is a single entry in a repository manifest. The .hell
// recipe remains the source of truth for build instructions; the manifest
// carries descriptions and the authoritative binary metadata used by the
// binary fallback path.
type ManifestPackage struct {
	Name        string          `yaml:"name"`
	Version     string          `yaml:"version"`
	Description string          `yaml:"description"`
	Depends     []string        `yaml:"depends"`
	Binary      *ManifestBinary `yaml:"binary,omitempty"`
}

// ManifestBinary describes a signed prebuilt package archive.
type ManifestBinary struct {
	URL    string `yaml:"url"`
	SHA256 string `yaml:"sha256"`
	Size   string `yaml:"size"`
	SigURL string `yaml:"sig_url,omitempty"`
	Signer string `yaml:"signer,omitempty"`
}

// Source identifies where a package came from.
type Source string

const (
	SourceOfficial Source = "official"
	SourceDUR      Source = "dur"
)

// PackageInfo is the assembled metadata Dante operates on. It merges a parsed
// .hell recipe with its manifest entry.
type PackageInfo struct {
	Name             string
	Version          string
	Description      string
	Depends          []string
	RecipePath       string
	Source           Source
	SourceURL        string
	BinaryURL        string
	BinarySHA256     string
	BinarySize       string
	BinarySigURL     string
	Installed        bool
	InstalledVersion string
}

// Dante is the package manager state. It ties configuration to the repository
// cache, build area and installation database.
type Dante struct {
	Config *Config
	Index  map[string]*PackageInfo
	Dur    map[string]*PackageInfo
	// Arch is the target architecture, defaulting to runtime GOARCH.
	Arch string
}

// New builds a Dante instance for the given configuration.
func New(cfg *Config) *Dante {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Dante{Config: cfg, Arch: runtimeArch()}
}

func runtimeArch() string {
	return currentArch
}

// syncRepo clones or fast-forwards a single git repository.
func syncRepo(url, dir string) error {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		fmt.Printf(":: Cloning %s\n", url)
		cmd := exec.Command("git", "clone", "--depth", "1", url, dir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	fmt.Printf(":: Updating %s\n", url)
	cmd := exec.Command("git", "-C", dir, "pull", "--ff-only")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Sync clones or fast-forwards every configured repository, including the DUR.
func (d *Dante) Sync() error {
	if err := d.Config.EnsureDirs(); err != nil {
		return err
	}
	for i, url := range d.Config.Repos() {
		dir := d.Config.RepoDir
		if i > 0 {
			dir = filepath.Join(filepath.Dir(d.Config.RepoDir), fmt.Sprintf("repo-%d", i))
		}
		if err := syncRepo(url, dir); err != nil {
			return err
		}
	}
	if d.Config.DurEnabled() {
		if err := syncRepo(d.Config.DurURL, d.Config.DurRepoDir()); err != nil {
			fmt.Printf("!! DUR sync failed: %v\n", err)
		}
	}
	return nil
}

// ImportKey fetches and imports the Antenora maintainer GPG key used to sign
// binary packages.
func (d *Dante) ImportKey() error {
	fmt.Printf(":: Importing maintainer key %s\n", d.Config.GPGKeyID)
	cmd := exec.Command("gpg", "--recv-keys", d.Config.GPGKeyID)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// loadManifest reads manifest.yaml from a repository directory.
func loadManifest(dir string) (*Manifest, error) {
	path := filepath.Join(dir, "manifest.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest %s: %w", path, err)
	}
	return &m, nil
}

// recipesIn returns every .hell recipe under dir, sorted by name.
func recipesIn(dir string) ([]string, error) {
	var out []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".hell") {
			out = append(out, path)
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

// buildRepoIndex assembles a package index from a single repository directory.
func (d *Dante) buildRepoIndex(dir string, src Source) (map[string]*PackageInfo, error) {
	recipes, err := recipesIn(dir)
	if err != nil {
		return nil, err
	}
	manifest, _ := loadManifest(dir)
	meta := map[string]ManifestPackage{}
	if manifest != nil {
		for _, mp := range manifest.Packages {
			meta[mp.Name] = mp
		}
	}
	index := map[string]*PackageInfo{}
	for _, rp := range recipes {
		pkg, err := d.ParseRecipe(rp)
		if err != nil {
			return nil, fmt.Errorf("recipe %s: %w", rp, err)
		}
		pi := &PackageInfo{
			Name:        pkg.Name,
			Version:     pkg.Version,
			Description: pkg.Description,
			Depends:     pkg.Depends,
			RecipePath:  rp,
			Source:      src,
			SourceURL:   pkg.Source,
		}
		if pkg.Binary != nil {
			pi.BinaryURL = pkg.Binary.URL
			pi.BinarySHA256 = pkg.Binary.SHA256
			pi.BinarySize = pkg.Binary.Size
		}
		if mp, ok := meta[pkg.Name]; ok {
			if mp.Description != "" {
				pi.Description = mp.Description
			}
			if mp.Binary != nil {
				if mp.Binary.URL != "" {
					pi.BinaryURL = mp.Binary.URL
				}
				if mp.Binary.SHA256 != "" {
					pi.BinarySHA256 = mp.Binary.SHA256
				}
				if mp.Binary.Size != "" {
					pi.BinarySize = mp.Binary.Size
				}
				if mp.Binary.SigURL != "" {
					pi.BinarySigURL = mp.Binary.SigURL
				}
			}
		}
		if rec, err := d.loadRecord(pkg.Name); err == nil && rec != nil {
			pi.Installed = true
			pi.InstalledVersion = rec.Version
		}
		index[pkg.Name] = pi
	}
	return index, nil
}

// BuildIndex parses every configured repository into the official index and
// (if configured) the DUR index, marking installed packages from the local
// database.
func (d *Dante) BuildIndex() error {
	merged := map[string]*PackageInfo{}
	for i := range d.Config.Repos() {
		dir := d.Config.RepoDir
		if i > 0 {
			dir = filepath.Join(filepath.Dir(d.Config.RepoDir), fmt.Sprintf("repo-%d", i))
		}
		idx, err := d.buildRepoIndex(dir, SourceOfficial)
		if err != nil {
			// A secondary repo that has not been synced yet is not fatal.
			if i > 0 && os.IsNotExist(err) {
				continue
			}
			if i > 0 {
				continue
			}
			// Primary repo missing: fall back to an empty index rather than
			// hard-failing so `dante` can still bootstrap from the DUR.
			merged = map[string]*PackageInfo{}
			continue
		}
		for name, pi := range idx {
			merged[name] = pi
		}
	}
	d.Index = merged

	if d.Config.DurEnabled() {
		dur, err := d.buildRepoIndex(d.Config.DurRepoDir(), SourceDUR)
		if err != nil {
			d.Dur = map[string]*PackageInfo{}
		} else {
			d.Dur = dur
		}
	} else {
		d.Dur = map[string]*PackageInfo{}
	}
	return nil
}

// Lookup finds a package in the official index and (optionally) the DUR.
func (d *Dante) Lookup(name string, includeDur bool) (*PackageInfo, Source, bool) {
	if pi, ok := d.Index[name]; ok {
		return pi, SourceOfficial, true
	}
	if includeDur {
		if pi, ok := d.Dur[name]; ok {
			return pi, SourceDUR, true
		}
	}
	return nil, "", false
}

// Resolve performs a depth-first topological sort over the dependency graph
// rooted at name, returning package names in install order. When preferDur is
// set, DUR packages are chosen first; official packages always satisfy a
// dependency if no DUR equivalent exists. It reports the full cycle path if a
// dependency cycle is detected.
func (d *Dante) Resolve(name string, preferDur bool) ([]string, error) {
	if d.Index == nil {
		if err := d.BuildIndex(); err != nil {
			return nil, err
		}
	}
	state := map[string]int{}
	var order []string
	var stack []string

	lookup := func(n string) (*PackageInfo, bool) {
		if preferDur {
			if pi, ok := d.Dur[n]; ok {
				return pi, true
			}
		}
		if pi, ok := d.Index[n]; ok {
			return pi, true
		}
		if pi, ok := d.Dur[n]; ok {
			return pi, true
		}
		return nil, false
	}

	var visit func(string) error
	visit = func(n string) error {
		switch state[n] {
		case 1:
			start := 0
			for i, m := range stack {
				if m == n {
					start = i
					break
				}
			}
			cycle := append(append([]string{}, stack[start:]...), n)
			return fmt.Errorf("dependency cycle detected: %s", strings.Join(cycle, " -> "))
		case 2:
			return nil
		}
		pkg, ok := lookup(n)
		if !ok {
			if preferDur {
				return fmt.Errorf("package %q not found in repositories or DUR", n)
			}
			return fmt.Errorf("package %q not found in repositories (try 'dante install -A %s' for the DUR)", n, n)
		}
		state[n] = 1
		stack = append(stack, n)
		for _, dep := range pkg.Depends {
			if err := visit(dep); err != nil {
				return err
			}
		}
		stack = stack[:len(stack)-1]
		state[n] = 2
		order = append(order, n)
		return nil
	}
	if err := visit(name); err != nil {
		return nil, err
	}
	return order, nil
}

// GenManifest regenerates a repository's manifest.yaml from its .hell recipes.
// Descriptions, dependencies and binary metadata are sourced from the recipes.
func (d *Dante) GenManifest(dir string) error {
	recipes, err := recipesIn(dir)
	if err != nil {
		return err
	}
	var manifest Manifest
	for _, rp := range recipes {
		pkg, err := d.ParseRecipe(rp)
		if err != nil {
			return fmt.Errorf("recipe %s: %w", rp, err)
		}
		mp := ManifestPackage{
			Name:        pkg.Name,
			Version:     pkg.Version,
			Description: pkg.Description,
			Depends:     pkg.Depends,
		}
		if pkg.Binary != nil {
			mp.Binary = &ManifestBinary{
				URL:    pkg.Binary.URL,
				SHA256: pkg.Binary.SHA256,
				Size:   pkg.Binary.Size,
			}
		}
		manifest.Packages = append(manifest.Packages, mp)
	}
	data, err := yaml.Marshal(&manifest)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "manifest.yaml"), data, 0o644)
}

// InstalledPackages lists the names of all packages recorded in the database.
func (d *Dante) InstalledPackages() ([]string, error) {
	dir := filepath.Join(d.Config.DBDir, "installed")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			out = append(out, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	sort.Strings(out)
	return out, nil
}
