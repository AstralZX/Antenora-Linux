// Copyright (C) 2026 Antenora Linux contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package dante

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeRecipes(t *testing.T, dir string, recipes map[string]string) {
	t.Helper()
	for name, content := range recipes {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

const simple = `build { run "true" }
install { run "true" }`

func TestResolveTopologicalOrder(t *testing.T) {
	dir := t.TempDir()
	writeRecipes(t, dir, map[string]string{
		"a.hell": "package \"a\" version \"1\" {\n    depends \"b\" \"c\"\n    " + simple + "\n}",
		"b.hell": "package \"b\" version \"1\" {\n    depends \"c\"\n    " + simple + "\n}",
		"c.hell": "package \"c\" version \"1\" {\n    " + simple + "\n}",
		"z.hell": "package \"z\" version \"1\" {\n    " + simple + "\n}",
	})
	cfg := DefaultConfig()
	cfg.RepoDir = dir
	cfg.DurURL = ""
	d := New(cfg)
	order, err := d.Resolve("a", false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"c", "b", "a"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}

func TestResolveCycleDetection(t *testing.T) {
	dir := t.TempDir()
	writeRecipes(t, dir, map[string]string{
		"a.hell": "package \"a\" version \"1\" {\n    depends \"b\"\n    " + simple + "\n}",
		"b.hell": "package \"b\" version \"1\" {\n    depends \"c\"\n    " + simple + "\n}",
		"c.hell": "package \"c\" version \"1\" {\n    depends \"a\"\n    " + simple + "\n}",
	})
	cfg := DefaultConfig()
	cfg.RepoDir = dir
	cfg.DurURL = ""
	d := New(cfg)
	_, err := d.Resolve("a", false)
	if err == nil {
		t.Fatal("expected cycle error")
	}
	if !contains(err.Error(), "a", "b", "c") {
		t.Fatalf("cycle error does not list all packages: %v", err)
	}
}

func TestResolveMissingPackage(t *testing.T) {
	dir := t.TempDir()
	writeRecipes(t, dir, map[string]string{
		"a.hell": "package \"a\" version \"1\" {\n    depends \"nonexistent\"\n    " + simple + "\n}",
	})
	cfg := DefaultConfig()
	cfg.RepoDir = dir
	cfg.DurURL = ""
	d := New(cfg)
	_, err := d.Resolve("a", false)
	if err == nil {
		t.Fatal("expected missing package error")
	}
}

func contains(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
