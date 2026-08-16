# Writing Hell recipes

Hell is deliberately small. You can learn the whole language in five minutes,
and a real recipe usually fits in under 30 lines. If yours passes 50 lines, the
parser refuses to run it (`BLASPHEMY_ERROR`) — that's a feature, not a bug.

## A recipe in 30 seconds

```hell
# my-tool.hell
package "my-tool" version "1.0.0" {
    description "Does something useful"
    source "https://example.com/my-tool-1.0.0.tar.gz"

    build {
        run "./configure --prefix=/usr"
        run "make -j$HELL_JOBS"
    }

    install {
        run "make install DESTDIR=$HELL_ROOT"
    }
}
```

That's a complete, installable package. Four things:

1. `package "name" version "x"` — identity.
2. `source` — where the tarball comes from.
3. `build { run "..." }` — commands to compile (run in the extracted source dir).
4. `install { run "..." }` — commands to install (use `DESTDIR=$HELL_ROOT`).

Everything else is optional.

## The rules

- `#` starts a comment.
- Strings are `"double quoted"`.
- Indentation is just style; the parser ignores it.
- Every recipe lives in its own file named `<package>.hell`.

## All the keywords

| Directive | What it does |
|-----------|--------------|
| `package "name" version "x"` | Required. The package identity. |
| `description "..."` | One-line summary shown in `dante search`. |
| `source "url"` | Tarball to download (`.tar.gz`/`.tar.xz`/`.tar.bz2`). |
| `depends "a" "b"` | Dependencies (resolved automatically). |
| `build { ... }` | Compile steps. |
| `install { ... }` | Install steps. |
| `post_install { ... }` | Runs once after install (links, caches, config). |
| `binary "url" { sha256 "..." size "..." }` | Optional signed binary fallback. |
| `arch x86_64 { ... }` | Architecture-specific flags. |

### Commands inside `build` / `install`

| Command | Example |
|---------|---------|
| `run "..."` | `run "make -j$HELL_JOBS"` — any shell command |
| `mkdir "/path"` | `mkdir "/etc/foo"` — make a dir (staged) |
| `cp "src" "dst"` | `cp "README.md" "$HELL_ROOT/usr/share/doc/my-tool/README"` |
| `rm "/path"` | `rm "/usr/lib/foo.la"` — remove a staged file |
| `ln "target" "link"` | `ln "gcc" "/usr/bin/cc"` — symlink |
| `var "NAME" "value"` | `var "CFLAGS" "-O3"` — set an env var for the build |
| `patch "file" "patch"` | `patch "src.c" "fix.patch"` — apply a patch |

### Variables

`$HELL_ROOT` (staging dir for install), `$HELL_SRC_DIR` (extracted source),
`$HELL_ARCH` (`x86_64`/`aarch64`), `$HELL_CFLAGS`, `$HELL_LDFLAGS`,
`$HELL_JOBS` (parallel jobs).

### Conditions

```hell
if arch_x86_64 { run "..." } else { run "..." }
if binary_available { run "..." }
if file_exists "/etc/foo" { run "..." }
if env "SOME_VAR" { run "..." }
```

## Common build patterns

**Autotools** (configure/make):
```hell
build { run "./configure --prefix=/usr"; run "make -j$HELL_JOBS" }
install { run "make install DESTDIR=$HELL_ROOT" }
```

**Meson**:
```hell
build { run "meson setup build --prefix=/usr"; run "ninja -C build" }
install { run "ninja -C build install --destdir $HELL_ROOT" }
```

**CMake**:
```hell
build { run "cmake -B build -DCMAKE_INSTALL_PREFIX=/usr"; run "cmake --build build -j$HELL_JOBS" }
install { run "cmake --install build --prefix $HELL_ROOT/usr" }
```

**Go**:
```hell
build { run "go build -trimpath -o mytool ." }
install { run "install -Dm755 mytool $HELL_ROOT/usr/bin/mytool" }
```

**Rust**:
```hell
build { run "cargo build --release --locked" }
install { run "install -Dm755 target/release/mytool $HELL_ROOT/usr/bin/mytool" }
```

**Config-only** (no compile):
```hell
build { run "true" }
install { cp "config" "$HELL_ROOT/etc/mytool.conf" }
```

## Submitting to the DUR

1. Write `my-package.hell` (start from `docs/recipe-template.hell`).
2. Validate it parses: `dante compile my-package.hell -`
3. Ship it:
   ```sh
   scripts/dur-submit.sh my-package.hell
   ```
   or open a PR adding it under `dur/recipes/`.

Then anyone installs it with `dante install -A my-package`.
