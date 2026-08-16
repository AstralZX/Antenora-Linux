// Copyright (C) 2026 Antenora Linux contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package hell

import (
	"strings"
	"testing"
)

func TestParseFullRecipe(t *testing.T) {
	src := `package "curl" version "8.5.0" {
    source "https://example.com/curl-8.5.0.tar.xz"
    depends "openssl" "zlib" "ca-certificates"

    arch x86_64 {
        cflags "-march=native -O3 -pipe"
    }
    arch aarch64 {
        cflags "-mcpu=native -O3 -pipe"
    }

    build {
        run "./configure --prefix=/usr --with-openssl"
        run "make -j$HELL_JOBS"
        if env "DEBUG" {
            run "echo debugging"
        } else {
            run "echo not debugging"
        }
    }

    install {
        run "make install DESTDIR=$HELL_ROOT"
    }

    binary "https://example.com/curl.db" {
        sha256 "a1b2c3"
        size "2.4MB"
    }

    post_install {
        run "update-ca-certificates"
    }
}`
	pkg, err := Parse(src, "curl.hell")
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Name != "curl" || pkg.Version != "8.5.0" {
		t.Fatalf("name/version wrong: %s %s", pkg.Name, pkg.Version)
	}
	if len(pkg.Depends) != 3 {
		t.Fatalf("depends = %v", pkg.Depends)
	}
	if pkg.Arches["x86_64"].CFlags != "-march=native -O3 -pipe" {
		t.Fatalf("x86_64 cflags wrong: %q", pkg.Arches["x86_64"].CFlags)
	}
	if len(pkg.Build) != 3 {
		t.Fatalf("build stmts = %d", len(pkg.Build))
	}
	if pkg.Binary == nil || pkg.Binary.SHA256 != "a1b2c3" {
		t.Fatalf("binary wrong: %+v", pkg.Binary)
	}
	if len(pkg.Install) != 1 || len(pkg.PostInstall) != 1 {
		t.Fatalf("install/post_install wrong")
	}
}

func TestBlasphemyError(t *testing.T) {
	var b strings.Builder
	b.WriteString(`package "x" version "1" {`)
	b.WriteString("\n")
	for i := 0; i < 55; i++ {
		b.WriteString("    # comment padding\n")
	}
	b.WriteString("}")
	_, err := Parse(b.String(), "x.hell")
	if err == nil {
		t.Fatal("expected BlasphemyError")
	}
	if _, ok := err.(BlasphemyError); !ok {
		t.Fatalf("expected BlasphemyError, got %T: %v", err, err)
	}
}

func TestParseConditionals(t *testing.T) {
	src := `package "x" version "1" {
    build {
        if arch_x86_64 { run "x" } else { run "y" }
        if file_exists "/etc/hosts" { run "z" }
    }
}`
	pkg, err := Parse(src, "x.hell")
	if err != nil {
		t.Fatal(err)
	}
	if len(pkg.Build) != 2 {
		t.Fatalf("build stmts = %d", len(pkg.Build))
	}
	first := pkg.Build[0]
	if first.Op != "if" || first.Cond != "arch_x86_64" {
		t.Fatalf("first stmt wrong: %+v", first)
	}
	if len(first.Then) != 1 || len(first.Else) != 1 {
		t.Fatalf("then/else wrong: %d/%d", len(first.Then), len(first.Else))
	}
}
