// Copyright (C) 2026 Antenora Linux contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package dante

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/antenora/dante/pkg/hell"
)

// currentArch maps Go's GOARCH to the Hell architecture identifier.
var currentArch = mapGOARCH(runtime.GOARCH)

func mapGOARCH(goarch string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	case "386":
		return "i686"
	default:
		return goarch
	}
}

// ParseRecipe reads and parses a .hell recipe file.
func (d *Dante) ParseRecipe(path string) (*hell.Package, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return hell.Parse(string(data), path)
}

// Download fetches url to dest, creating parent directories as needed.
func (d *Dante) Download(url, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp := dest + ".part"
	fmt.Printf(":: Fetching %s\n", url)
	cmd := exec.Command("wget", "-q", "--show-progress", "-O", tmp, url)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("downloading %s: %w", url, err)
	}
	if fi, err := os.Stat(tmp); err != nil || fi.Size() == 0 {
		os.Remove(tmp)
		if err == nil {
			err = fmt.Errorf("downloaded empty file")
		}
		return fmt.Errorf("downloading %s: %w", url, err)
	}
	return os.Rename(tmp, dest)
}

// extractArchive unpacks a (possibly gzip/bzip2-compressed) tar archive into
// destDir and returns the top-level directory created inside it.
func (d *Dante) extractArchive(archive, destDir string) (string, error) {
	f, err := os.Open(archive)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var r io.Reader = f
	switch {
	case strings.HasSuffix(archive, ".gz"), strings.HasSuffix(archive, ".tgz"):
		gz, err := gzip.NewReader(f)
		if err != nil {
			return "", err
		}
		defer gz.Close()
		r = gz
	case strings.HasSuffix(archive, ".bz2"):
		r = bzip2.NewReader(f)
	case strings.HasSuffix(archive, ".xz"):
		// External xz handles this via the shell path below.
		return d.extractViaTar(archive, destDir)
	}

	return d.extractTarStream(r, destDir)
}

func (d *Dante) extractViaTar(archive, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	cmd := exec.Command("tar", "-xf", archive, "-C", destDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return d.topLevelDir(destDir)
}

func (d *Dante) extractTarStream(r io.Reader, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		target := filepath.Join(destDir, hdr.Name)
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return "", fmt.Errorf("archive path escape: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return "", err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return "", err
			}
			out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return "", err
			}
			out.Close()
			if err := os.Chtimes(target, hdr.ModTime, hdr.ModTime); err != nil {
				return "", err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return "", err
			}
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return "", err
			}
		}
	}
	return d.topLevelDir(destDir)
}

func (d *Dante) topLevelDir(destDir string) (string, error) {
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return "", err
	}
	if len(entries) == 1 && entries[0].IsDir() {
		return filepath.Join(destDir, entries[0].Name()), nil
	}
	return destDir, nil
}

// BuildFromSource downloads, extracts and builds a package from source,
// returning the staging directory containing its installed files.
func (d *Dante) BuildFromSource(pi *PackageInfo, srcDir string) (string, error) {
	cfg := d.Config
	pkg, err := d.ParseRecipe(pi.RecipePath)
	if err != nil {
		return "", err
	}
	if pkg.Source == "" {
		stage := filepath.Join(srcDir, "stage")
		if err := os.MkdirAll(stage, 0o755); err != nil {
			return "", err
		}
		interp := hell.NewInterpreter(stage, "", d.Arch, d.Config.Root)
		interp.Verbose = true
		interp.BinaryAvailable = pi.BinaryURL != ""
		interp.Setup(pkg, cfg.Jobs())
		if err := interp.Exec(pkg.Install); err != nil {
			return "", fmt.Errorf("%s install: %w", pi.Name, err)
		}
		if err := interp.Exec(pkg.PostInstall); err != nil {
			return "", fmt.Errorf("%s post_install: %w", pi.Name, err)
		}
		return stage, nil
	}

	cacheDir := filepath.Join(cfg.CacheDir, "sources")
	archive := filepath.Join(cacheDir, filepath.Base(pkg.Source))
	if fi, err := os.Stat(archive); err != nil || fi.Size() == 0 {
		if err := d.Download(pkg.Source, archive); err != nil {
			return "", err
		}
	}

	workDir := filepath.Join(srcDir, pi.Name)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", err
	}
	extracted, err := d.extractArchive(archive, workDir)
	if err != nil {
		return "", err
	}

	stage := filepath.Join(workDir, "stage")
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return "", err
	}

	interp := hell.NewInterpreter(stage, extracted, d.Arch, d.Config.Root)
	interp.Verbose = true
	interp.BinaryAvailable = pi.BinaryURL != ""
	interp.Setup(pkg, cfg.Jobs())

	if err := interp.Exec(pkg.Build); err != nil {
		return "", fmt.Errorf("%s build: %w", pi.Name, err)
	}
	if err := interp.Exec(pkg.Install); err != nil {
		return "", fmt.Errorf("%s install: %w", pi.Name, err)
	}
	if err := interp.Exec(pkg.PostInstall); err != nil {
		return "", fmt.Errorf("%s post_install: %w", pi.Name, err)
	}

	if cfg.CleanSourceEnabled() {
		_ = os.Remove(archive)
		_ = os.RemoveAll(extracted)
	}
	return stage, nil
}

// CompileToMakefile renders a package recipe as a standalone Makefile, useful
// for debugging and for builds outside the Hell interpreter.
func (d *Dante) CompileToMakefile(path, out string) error {
	pkg, err := d.ParseRecipe(path)
	if err != nil {
		return err
	}
	gen := &makefileGen{}
	src := gen.generate(pkg, d.Arch)
	if out == "-" {
		fmt.Print(src)
		return nil
	}
	return os.WriteFile(out, []byte(src), 0o644)
}

// makefileGen translates a Hell AST into a standalone Makefile.
type makefileGen struct{}

func (g *makefileGen) generate(pkg *hell.Package, arch string) string {
	var b strings.Builder
	b.WriteString("# Generated by Dante from " + pkg.Path + "\n")
	b.WriteString("# Antenora Linux - Hell recipe compiled to Make\n\n")
	fmt.Fprintf(&b, "PACKAGE := %s\n", pkg.Name)
	fmt.Fprintf(&b, "VERSION := %s\n", pkg.Version)
	b.WriteString("HELL_ROOT ?= $(DESTDIR)\n")
	fmt.Fprintf(&b, "HELL_SRC_DIR := $(CURDIR)/src/%s-%s\n", pkg.Name, pkg.Version)
	fmt.Fprintf(&b, "HELL_ARCH := %s\n", arch)
	fmt.Fprintf(&b, "HELL_CFLAGS := %s\n", pkg.CFlags)
	fmt.Fprintf(&b, "HELL_LDFLAGS := %s\n", pkg.LDFlags)
	b.WriteString("HELL_JOBS ?= $(shell nproc)\n\n")
	if pkg.Source != "" {
		fmt.Fprintf(&b, "SOURCE_URL := %s\n", pkg.Source)
		b.WriteString("SOURCE_TARBALL := $(notdir $(SOURCE_URL))\n\n")
	}
	b.WriteString("all: fetch build install post_install\n\n")
	if pkg.Source != "" {
		b.WriteString("fetch:\n")
		fmt.Fprintf(&b, "\twget -c -O $(SOURCE_TARBALL) $(SOURCE_URL)\n")
		fmt.Fprintf(&b, "\tmkdir -p $(HELL_SRC_DIR)\n")
		fmt.Fprintf(&b, "\ttar -xf $(SOURCE_TARBALL) -C $(dir $(HELL_SRC_DIR)) --strip-components=1\n\n")
	} else {
		b.WriteString("fetch:\n\t@true\n\n")
	}
	b.WriteString("build:\n")
	for _, s := range pkg.Build {
		g.emitStmt(&b, s, "\t", arch)
	}
	b.WriteString("install:\n")
	for _, s := range pkg.Install {
		g.emitStmt(&b, s, "\t", arch)
	}
	b.WriteString("post_install:\n")
	for _, s := range pkg.PostInstall {
		g.emitStmt(&b, s, "\t", arch)
	}
	b.WriteString("\n.PHONY: all fetch build install post_install\n")
	return b.String()
}

func (g *makefileGen) emitStmts(b *strings.Builder, stmts []hell.Stmt, indent, arch string) {
	for _, s := range stmts {
		g.emitStmt(b, s, indent, arch)
	}
}

func (g *makefileGen) emitStmt(b *strings.Builder, s hell.Stmt, indent, arch string) {
	switch s.Op {
	case "if":
		cond := makeCondExpr(s.Cond, s.CondArg)
		fmt.Fprintf(b, "%sifeq ($(%s),1)\n", indent, cond)
		g.emitStmts(b, s.Then, indent+"\t", arch)
		if len(s.Else) > 0 {
			fmt.Fprintf(b, "%selse\n", indent)
			g.emitStmts(b, s.Else, indent+"\t", arch)
		}
		fmt.Fprintf(b, "%sendif\n", indent)
	case "run":
		fmt.Fprintf(b, "%scd $(HELL_SRC_DIR) && %s\n", indent, s.Args[0])
	case "patch":
		fmt.Fprintf(b, "%scd $(HELL_SRC_DIR) && patch -p1 < %s\n", indent, s.Args[1])
	case "mkdir":
		fmt.Fprintf(b, "%smkdir -p $(HELL_ROOT)%s\n", indent, s.Args[0])
	case "cp":
		fmt.Fprintf(b, "%scp -a $(HELL_ROOT)%s $(HELL_ROOT)%s\n", indent, s.Args[0], s.Args[1])
	case "rm":
		fmt.Fprintf(b, "%srm -rf $(HELL_ROOT)%s\n", indent, s.Args[0])
	case "ln":
		fmt.Fprintf(b, "%sln -s %s $(HELL_ROOT)%s\n", indent, s.Args[0], s.Args[1])
	case "var":
		fmt.Fprintf(b, "%sexport %s=%s\n", indent, s.Args[0], s.Args[1])
	}
}

func makeCondExpr(cond, arg string) string {
	switch cond {
	case "arch_x86_64":
		return "HELL_ARCH_X86_64"
	case "arch_aarch64":
		return "HELL_ARCH_AARCH64"
	case "binary_available":
		return "HELL_BINARY_AVAILABLE"
	case "file_exists":
		return "HELL_FILE_" + sanitizeIdent(arg)
	case "env":
		return "HELL_ENV_" + sanitizeIdent(arg)
	}
	return sanitizeIdent(cond)
}

func sanitizeIdent(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return strings.ToUpper(b.String())
}
