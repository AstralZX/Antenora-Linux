// Copyright (C) 2026 Antenora Linux contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package dante

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// InstalledRecord is the on-disk record of an installed package.
type InstalledRecord struct {
	Name       string   `json:"name"`
	Version    string   `json:"version"`
	FromBinary bool     `json:"from_binary"`
	Files      []string `json:"files"`
	Depends    []string `json:"depends"`
	Timestamp  string   `json:"timestamp"`
}

func (d *Dante) recordPath(name string) string {
	return filepath.Join(d.Config.DBDir, "installed", name+".json")
}

func (d *Dante) saveRecord(rec *InstalledRecord) error {
	dir := filepath.Dir(d.recordPath(rec.Name))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(d.recordPath(rec.Name), data, 0o644)
}

func (d *Dante) loadRecord(name string) (*InstalledRecord, error) {
	data, err := os.ReadFile(d.recordPath(name))
	if err != nil {
		return nil, err
	}
	var rec InstalledRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// Install resolves and installs a package and all of its dependencies.
// When preferDur is true, DUR packages are preferred over official ones.
func (d *Dante) Install(name string, preferDur bool) error {
	if err := d.BuildIndex(); err != nil {
		return err
	}
	order, err := d.Resolve(name, preferDur)
	if err != nil {
		return err
	}
	fmt.Printf(":: Resolved %d package(s): %v\n", len(order), order)
	for _, n := range order {
		if err := d.installOne(n, preferDur); err != nil {
			return fmt.Errorf("installing %s: %w", n, err)
		}
	}
	fmt.Printf(":: Installed %s\n", name)
	return nil
}

func (d *Dante) installOne(name string, preferDur bool) error {
	pi, _, ok := d.Lookup(name, preferDur)
	if !ok {
		return fmt.Errorf("package %q not found", name)
	}
	if pi.Installed && pi.InstalledVersion == pi.Version {
		fmt.Printf(":: %s %s already installed, skipping\n", name, pi.Version)
		return nil
	}

	fmt.Printf(":: Building %s %s\n", name, pi.Version)
	srcDir := filepath.Join(d.Config.BuildDir, name)
	if err := os.RemoveAll(srcDir); err != nil {
		return err
	}
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return err
	}

	var stage string
	fromBinary := false
	if d.Config.BinaryEnabled() && pi.BinaryURL != "" {
		s, err := d.InstallBinary(pi, srcDir)
		if err != nil {
			fmt.Printf("!! Binary unavailable (%v), falling back to source\n", err)
			s, err = d.BuildFromSource(pi, srcDir)
			if err != nil {
				return err
			}
			stage = s
		} else {
			stage = s
			fromBinary = true
		}
	} else {
		s, err := d.BuildFromSource(pi, srcDir)
		if err != nil {
			return err
		}
		stage = s
	}

	files, err := mergeStaging(stage, d.Config.Root)
	if err != nil {
		return err
	}
	rec := &InstalledRecord{
		Name:       pi.Name,
		Version:    pi.Version,
		FromBinary: fromBinary,
		Files:      files,
		Depends:    pi.Depends,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	if err := d.saveRecord(rec); err != nil {
		return err
	}
	fmt.Printf(":: %s %s installed (%d files)\n", name, pi.Version, len(files))
	return nil
}

// mergeStaging copies a staged install tree into root, returning the list of
// installed relative paths (directories excluded).
func mergeStaging(stage, root string) ([]string, error) {
	var files []string
	err := filepath.Walk(stage, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(stage, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(root, rel)
		switch {
		case info.IsDir():
			return os.MkdirAll(target, info.Mode())
		case info.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			return os.Symlink(link, target)
		default:
			if err := copyFileContents(path, target, info.Mode()); err != nil {
				return err
			}
		}
		files = append(files, rel)
		return nil
	})
	return files, err
}

func copyFileContents(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
