// Copyright (C) 2026 Antenora Linux contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package hell

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// expandEnv substitutes $VAR and ${VAR} occurrences in s using env.
func expandEnv(s string, env map[string]string) string {
	for k, v := range env {
		s = strings.ReplaceAll(s, "$"+k, v)
		s = strings.ReplaceAll(s, "${"+k+"}", v)
	}
	return s
}

// NumJobs reports the number of parallel build jobs, defaulting to NumCPU.
func NumJobs() string {
	return fmt.Sprintf("%d", runtime.NumCPU())
}

func mapToEnv(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}

// runShell executes a command through /bin/sh in dir with env merged into the
// process environment. In dry-run mode it only echoes the command.
func runShell(dir string, env map[string]string, cmdline string, dryRun, verbose bool, out io.Writer) error {
	if out == nil {
		out = os.Stdout
	}
	if dryRun || verbose {
		fmt.Fprintf(out, "  + %s\n", cmdline)
	}
	if dryRun {
		return nil
	}
	cmd := exec.Command("/bin/sh", "-c", cmdline)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), mapToEnv(env)...)
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}

// applyPatch applies a unified diff to the source tree rooted at dir.
func applyPatch(dir, patchFile string) error {
	cmd := exec.Command("/bin/sh", "-c", "patch -p1 -N < \""+patchFile+"\"")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// copyPath copies a file or directory tree from src to dst.
func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyTree(src, dst)
	}
	return copyFile(src, dst, info.Mode())
}

func copyFile(src, dst string, mode os.FileMode) error {
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

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		return copyFile(path, target, info.Mode())
	})
}
