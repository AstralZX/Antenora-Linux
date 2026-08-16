# Antenora Linux — "Inferno" 1.0.0

> *Abandon all hope, ye who enter here.*
>
> The Ninth Circle of Speed. An operating system forged from the raw silicon
> up, in contempt of bloat. s6 for init. glibc for compatibility. XFS for
> warriors. A package manager that walks you through Hell.

---

## The Ten Commandments (abridged)

| # | Law | Verdict |
|---|-----|---------|
| I | systemd | **FORBIDDEN** — s6 + s6-rc only |
| II | musl | Rejected — **glibc** always |
| III | ext4/btrfs/zfs | Rejected — **XFS** (`ftype=1 inode64`) |
| IV | package manager | **Dante**, written in Go |
| V | package language | **Hell** (`.hell`) |
| VI | kernel | **linux-cachyos-bore**, 1000Hz, BORE |
| VII | bootloader | **GRUB 2**, dark red/black, 3s timeout |
| VIII | installer | **The Gates of Hell** (TTY Bash) |
| IX | power user | Zsh + p10k, ZRAM, nftables, sysctl |

---

## Repository layout

```
antenora/
├── cmd/dante/            Dante CLI entry point
├── pkg/dante/            Package manager (Go): repo, install, build, binary…
├── pkg/hell/             Hell language: lexer, parser, interpreter, builtins
├── scripts/              install.sh, stage3-build.sh, iso-build.sh
├── s6-services/          s6-rc run scripts (agetty, dbus, pipewire, …)
├── recipes/              .hell package recipes
├── kernel-config/config  linux-cachyos-bore baseline .config
├── iso-build/            mkarchiso profile + live rootfs overlay
├── theme/                GRUB + Plymouth themes (Antenora crest)
├── sysctl/               99-antenora-performance.conf
├── manifest.yaml         Central repo manifest (sample)
└── docs/                 This document
```

---

## Building

### Prerequisites
- Go 1.22+
- `git`, `wget`, `tar`, `xz`
- `mkarchiso` (for the ISO), `dialog` (installer gauge)

### 1. Build Dante

```sh
make dante            # -> dist/dante
# or
go build ./cmd/dante
```

### 2. Run the tests

```sh
make test
```

### 3. Build the Stage3 bootstrap tarball

```sh
make stage3           # -> dist/stage3-1.0.0-x86_64.tar.xz
```

The Stage3 contains the s6 service set, base configuration and the Dante
binary, solving the bootstrap chicken-and-egg problem.

### 4. Build the live ISO

```sh
make iso              # -> dist/iso/antenora-*.iso
```

### 5. Build everything

```sh
make all
```

---

## Dante — the package manager

Dante is a source-based (with binary fallback) package manager. Every package
is described by a **Hell** recipe.

```sh
dante sync                    # update the central repo (git pull)
dante install base-system     # resolve deps (topological sort), build/install
dante install plasma          # KDE Plasma
dante remove plasma           # remove + reverse-dependency warnings
dante search pipewire         # search names/descriptions
dante info sway               # metadata + binary availability
dante update                  # upgrade installed packages
dante clean                   # purge caches
dante compile foo.hell foo.mk # compile a recipe to a Makefile
```

### Configuration — `/etc/dante/dante.conf`

```conf
BINARY="NO"                 # "YES" prefers signed binary packages
MAKEFLAGS="-j$(nproc)"
REPO_URL="https://github.com/antenora/package-repo.git"
BINARY_MIRROR="https://cdn.antenora.org/packages"
CLEAN_SOURCE="YES"
KEEP_DEPS="NO"
```

### Binary packages

Binary archives are **GPG-signed** with the Antenora maintainer key and
carry a SHA-256 checksum. Dante:

1. downloads the archive,
2. verifies the SHA-256 checksum,
3. verifies the detached GPG signature,
4. extracts it.

On **any** verification failure it refuses the binary and falls back to source
compilation. If source compilation then fails, it logs the error and tells you
to file a bug.

---

## The Hell language

A recipe is at most **50 lines** — longer and the parser raises a
`BLASPHEMY_ERROR`.

```hell
package "curl" version "8.5.0" {
    source "https://github.com/curl/curl/releases/download/8.5.0/curl-8.5.0.tar.xz"
    depends "openssl" "zlib" "ca-certificates"

    arch x86_64 {
        cflags "-march=native -O3 -pipe"
    }

    build {
        run "./configure --prefix=/usr --with-openssl"
        run "make -j$HELL_JOBS"
    }

    install {
        run "make install DESTDIR=$HELL_ROOT"
    }

    post_install {
        run "update-ca-certificates"
    }

    binary "https://cdn.antenora.org/packages/curl-8.5.0-x86_64.db" {
        sha256 "a1b2c3d4e5f6..."
        size "2.4MB"
    }
}
```

### Built-ins
`run`, `patch`, `mkdir`, `cp`, `rm`, `ln`, `var`

### Predefined variables
`$HELL_ROOT`, `$HELL_SRC_DIR`, `$HELL_ARCH`, `$HELL_CFLAGS`, `$HELL_LDFLAGS`,
`$HELL_JOBS`

### Control flow
```hell
if arch_x86_64 { run "..." } else { run "..." }
if file_exists "/some/path" { run "..." }
if env "SOME_VAR" { run "..." }
```

---

## Installing Antenora

Boot the live ISO and run the installer:

```sh
install-antenora
```

The **Gates of Hell** installer:

1. runs `cfdisk` for partitioning,
2. formats root as XFS (`ftype=1 inode64`), EFI as FAT32,
3. extracts the Stage3 tarball,
4. chroots into the new system,
5. installs the kernel (binary default, source if you are a masochist),
6. installs the base system and your chosen desktop (KDE / GNOME / Sway / none),
7. creates your user, adds you to `wheel`, configures sudo, generates SSH keys,
8. runs `grub-mkconfig`,
9. prints a winged lion,
10. reboots.

ZRAM (`lz4`) is enabled automatically — no swap partition. The firewall
(`nftables`) blocks everything inbound except SSH and ping. On first boot,
`/usr/lib/antenora/firstboot.sh` sets up ZRAM, generates SSH keys and loads
the firewall.

---

## Performance

`/etc/sysctl.d/99-antenora-performance.conf` ships:

```
vm.vfs_cache_pressure = 50
vm.swappiness         = 10
kernel.numa_balancing = 0
```

plus BBR congestion control, larger network buffers and aggressive dirty-page
flushing.

---

## License

MIT. The trident crest is drawn in blood. Use it well.
