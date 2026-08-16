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

// Manifest mirrors the central repository's manifest.yaml.
type Manifest struct {
	Packages []ManifestPackage `yaml:"packages"`
}

// ManifestPackage is a single entry in the repository manifest. The .hell
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

// PackageInfo is the assembled metadata Dante operates on. It merges a parsed
// .hell recipe with its manifest entry.
type PackageInfo struct {
	Name             string
	Version          string
	Description      string
	Depends          []string
	RecipePath       string
	Source           string
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

// Sync clones or fast-forwards the central package repository.
func (d *Dante) Sync() error {
	if err := d.Config.EnsureDirs(); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(d.Config.RepoDir, ".git")); err != nil {
		fmt.Printf(":: Cloning repository %s\n", d.Config.RepoURL)
		cmd := exec.Command("git", "clone", "--depth", "1", d.Config.RepoURL, d.Config.RepoDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	fmt.Printf(":: Updating repository in %s\n", d.Config.RepoDir)
	cmd := exec.Command("git", "-C", d.Config.RepoDir, "pull", "--ff-only")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// LoadManifest reads manifest.yaml from the repository root.
func (d *Dante) LoadManifest() (*Manifest, error) {
	path := filepath.Join(d.Config.RepoDir, "manifest.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}
	return &m, nil
}

// Recipes returns every .hell recipe in the repository, sorted by name.
func (d *Dante) Recipes() ([]string, error) {
	var out []string
	err := filepath.Walk(d.Config.RepoDir, func(path string, info os.FileInfo, err error) error {
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

// BuildIndex parses every recipe and merges manifest metadata into an
// in-memory index, also marking installed packages from the local database.
func (d *Dante) BuildIndex() error {
	recipes, err := d.Recipes()
	if err != nil {
		return err
	}
	manifest, err := d.LoadManifest()
	if err != nil {
		manifest = &Manifest{}
	}
	meta := map[string]ManifestPackage{}
	for _, mp := range manifest.Packages {
		meta[mp.Name] = mp
	}
	index := map[string]*PackageInfo{}
	for _, rp := range recipes {
		pkg, err := d.ParseRecipe(rp)
		if err != nil {
			return fmt.Errorf("recipe %s: %w", rp, err)
		}
		pi := &PackageInfo{
			Name:        pkg.Name,
			Version:     pkg.Version,
			Description: pkg.Description,
			Depends:     pkg.Depends,
			RecipePath:  rp,
			Source:      pkg.Source,
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
	d.Index = index
	return nil
}

// Resolve performs a depth-first topological sort over the dependency graph
// rooted at name, returning packages in install order. It reports the full
// cycle path if a dependency cycle is detected.
func (d *Dante) Resolve(name string) ([]string, error) {
	if d.Index == nil {
		if err := d.BuildIndex(); err != nil {
			return nil, err
		}
	}
	state := map[string]int{}
	var order []string
	var stack []string

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
		pkg, ok := d.Index[n]
		if !ok {
			return fmt.Errorf("package %q not found in repository", n)
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
