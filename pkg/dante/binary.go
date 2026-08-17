// Copyright (C) 2026 Antenora Linux contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package dante

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// BinaryError is returned when a binary package fails verification. The
// manager uses it to trigger an automatic fallback to source compilation.
type BinaryError struct {
	Reason string
}

func (e *BinaryError) Error() string { return "binary verification failed: " + e.Reason }

// sha256OfFile computes the hex-encoded SHA-256 digest of path.
func sha256OfFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// verifySHA256 checks the digest of path against expected.
func verifySHA256(path, expected string) error {
	if expected == "" {
		return nil
	}
	got, err := sha256OfFile(path)
	if err != nil {
		return err
	}
	if got != expected {
		return &BinaryError{Reason: fmt.Sprintf("sha256 mismatch: want %s got %s", expected, got)}
	}
	return nil
}

// verifySignature checks a detached GPG signature using the Antenora
// maintainer key. It shells out to gpg so the system trust store applies.
func (d *Dante) verifySignature(file, sigFile string) error {
	if _, err := os.Stat(sigFile); err != nil {
		return &BinaryError{Reason: "missing signature file"}
	}
	cmd := exec.Command("gpg", "--batch", "--verify", sigFile, file)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return &BinaryError{Reason: "gpg verification failed"}
	}
	return nil
}

// binaryURLs builds the ordered list of candidate URLs for a binary package,
// spanning the explicit URL (if any) and every configured mirror.
func (d *Dante) binaryURLs(pi *PackageInfo) []string {
	var out []string
	if pi.BinaryURL != "" {
		out = append(out, pi.BinaryURL)
	}
	base := pi.Name + "-" + pi.Version + "-" + d.Arch + ".tar.xz"
	for _, m := range d.Mirrors() {
		out = append(out, m+"/"+pi.Name+"/"+base)
	}
	return out
}

// InstallBinary downloads and verifies a binary package, then extracts it into
// the staging directory. It fails over across mirrors. It returns an error
// (BinaryError) on any verification failure so the caller can fall back to
// source.
func (d *Dante) InstallBinary(pi *PackageInfo, srcDir string) (string, error) {
	cfg := d.Config
	urls := d.binaryURLs(pi)
	if len(urls) == 0 {
		return "", fmt.Errorf("no binary url")
	}
	name := "binary.tar"
	if pi.BinaryURL != "" {
		if b := filepath.Base(pi.BinaryURL); b != "" && b != "." {
			name = b
		}
	} else {
		name = pi.Name + "-" + pi.Version + "-" + d.Arch + ".db"
	}
	archive := filepath.Join(cfg.CacheDir, "packages", name)
	if err := os.MkdirAll(filepath.Dir(archive), 0o755); err != nil {
		return "", err
	}

	var lastErr error
	downloaded := false
	for _, url := range urls {
		if err := d.Download(url, archive); err != nil {
			lastErr = err
			continue
		}
		downloaded = true
		break
	}
	if !downloaded {
		return "", fmt.Errorf("downloading binary: %w", lastErr)
	}

	if err := verifySHA256(archive, pi.BinarySHA256); err != nil {
		return "", err
	}

	if pi.BinarySigURL != "" {
		sigFile := archive + ".sig"
		if err := d.Download(pi.BinarySigURL, sigFile); err != nil {
			return "", err
		}
		if err := d.verifySignature(archive, sigFile); err != nil {
			return "", err
		}
	}

	unpack := filepath.Join(srcDir, pi.Name, "stage")
	if err := os.MkdirAll(unpack, 0o755); err != nil {
		return "", err
	}
	if _, err := d.extractArchive(archive, unpack); err != nil {
		return "", err
	}
	return applyStrip(unpack, pi.BinaryStrip)
}

// applyStrip returns the effective staging root after removing the top-level
// wrapper directory from a binary archive. A strip of zero (or a tarball that
// is already payload-rooted) leaves the tree untouched.
func applyStrip(dir string, strip int) (string, error) {
	if strip <= 0 {
		return dir, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		return dir, nil
	}
	inner := filepath.Join(dir, entries[0].Name())
	if strip == 1 {
		return inner, nil
	}
	return applyStrip(inner, strip-1)
}
