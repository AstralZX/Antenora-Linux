# Antenora Linux — "Inferno" 1.0.0

> *Abandon all hope, ye who enter here.*
>
> The Ninth Circle of Speed. An operating system forged from raw silicon, in
> contempt of bloat, in worship of latency budgets. s6 for init. glibc for
> compatibility. XFS for warriors. A package manager that walks you through
> Hell.

<p align="center">
  <img src="https://github.com/AstralZX/Antenora-Linux/blob/main/theme/grub-theme/antenna-logo.png" alt="Antenora crest" width="420" />
</p>

[![ci](https://github.com/AstralZX/Antenora-Linux/actions/workflows/ci.yml/badge.svg)](https://github.com/AstralZX/Antenora-Linux/actions/workflows/ci.yml)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)

---

## What is Antenora?

Antenora is a **source-based Linux distribution** that rejects every
compromise. It is the operating system for people who want their machine to be
a *machine*, not a parking lot for daemons. It is opinionated to the point of
dogma, fast to the point of disbelief.

## The Ten Commandments

| # | Law | Verdict |
|---|-----|---------|
| I | `systemd` | **FORBIDDEN** — s6 + s6-rc only, boot measured in milliseconds |
| II | `musl` | Rejected — **glibc** always (Steam, NVIDIA, proprietary CAD) |
| III | `ext4`/`btrfs`/`zfs` | Rejected — **XFS** (`ftype=1 inode64`) |
| IV | Package manager | **Dante**, written in Go |
| V | Package language | **Hell** (`.hell`), 50-line max (`BLASPHEMY_ERROR`) |
| VI | Kernel | **linux-cachyos-bore** — 1000Hz, preempt-dynamic, BORE, `-O3 -march=native` |
| VII | Bootloader | **GRUB 2**, dark red/black, 3s timeout |
| VIII | Install | **Manual** — the [Antenora Handbook](install-guide.md), by hand |
| IX | Power user | Zsh + Powerlevel10k, ZRAM `lz4`, nftables, sysctl |
| X | Purity | No bloat. No placeholders. No mercy. |

---

## The Pillars

### s6 — init without the tumor

Every service is a trivial script in `/etc/s6-rc/source`. `s6-rc` supervises,
restarts, and logs. Boot reaches a login prompt in the blink of an eye, not a
coffee break.

### Dante — the package manager

Written in **Go**. Source-based like Gentoo, with **signed binary fallbacks**.

```sh
dante sync                  # update repos + DUR
dante install base-system   # resolve deps, compile, install
dante install -A pkgname    # install from the DUR
dante remove pkgname        # uninstall + reverse-dep warnings
dante search term           # search official + DUR
dante info pkgname          # metadata + binary availability
dante update                # upgrade (version-aware)
dante clean                 # purge caches
dante toolchain             # bootstrap the build toolchain
dante mirror                # test + rank binary mirrors
dante compile in.hell out.mk # compile a recipe to a Makefile
```

- **Dependency resolution** is a topological sort with full cycle reporting.
- **Binary packages** are SHA-256 checked **and** GPG-verified with the
  Antenora maintainer key; any failure triggers automatic source fallback.
- **Mirrors** are ordered and fail over automatically.
- **Rollback-friendly**: every installed file is tracked in
  `/var/lib/dante/db`.

### Hell — the package language

A minimalistic cross between Makefile and Lua. Impossible to write poorly:
exceed 50 lines and the parser throws `BLASPHEMY_ERROR`.

```hell
package "curl" version "8.8.0" {
    source "https://github.com/curl/curl/releases/download/8.8.0/curl-8.8.0.tar.xz"
    depends "openssl" "zlib" "ca-certificates"

    arch x86_64 { cflags "-march=native -O3 -pipe" }

    build    { run "./configure --prefix=/usr --with-openssl"
               run "make -j$HELL_JOBS" }
    install  { run "make install DESTDIR=$HELL_ROOT" }

    binary "https://cdn.antenora.org/packages/curl-8.8.0-x86_64.db" {
        sha256 "a1b2c3d4e5f6..."
        size "2.4MB"
    }
}
```

**Built-ins:** `run`, `patch`, `mkdir`, `cp`, `rm`, `ln`, `var`.
**Variables:** `$HELL_ROOT`, `$HELL_SRC_DIR`, `$HELL_ARCH`, `$HELL_CFLAGS`,
`$HELL_LDFLAGS`, `$HELL_JOBS`.
**Control flow:** `if arch_x86_64` / `if file_exists "/x"` / `if env "VAR"`.

### The Toolchain

`dante toolchain` (or `scripts/toolchain.sh`) bootstraps the full build
toolchain from nothing: `linux-api-headers` → `gmp`/`mpfr`/`mpc`/`isl` →
`binutils` → `gcc` → `make`/autotools → `cmake`/`meson`/`ninja`.

---

## The Repositories

| Repository | Purpose |
|------------|---------|
| **Antenora-Linux** | The OS: install handbook, ISO profile, kernel config, themes, docs |
| **antenora-packages** | The central package repository (`recipes/` + `manifest.yaml`) |
| **antenora-dur** | The **Dante User Repository** — public, anyone can commit |
| **dante** | The package manager alone (`cmd/` + `pkg/`) |

Split the monorepo into its satellites with `scripts/split-repos.sh`.

### The DUR

The DUR is Antenora's AUR: a public repository **anyone can commit to**.
Submit a recipe and install it:

```sh
scripts/dur-submit.sh my-package.hell   # contribute
dante install -A my-package             # install
```

---

## Building

Antenora is built **from source, entirely by Dante**. There is no dependency on
Arch, mkarchiso, pacman, or any other distribution's tooling. The only thing
required from the host is a working C compiler and a handful of standard GNU
tools; everything else is bootstrapped.

### Prerequisites (host)
- `gcc`, `make`, `tar`, `xz`, `cpio`, `gzip`, `git`, `go`
- `xorriso`, `grub-mkrescue`, `mksquashfs` (ISO assembly)

### Build Dante

```sh
make dante          # -> dist/dante
```

### Run the tests

```sh
make test
```

### Bootstrap the toolchain

```sh
make toolchain      # gcc/glibc/binutils built from source into dist/sysroot
```

### Build the Stage3 bootstrap tarball

```sh
make stage3         # -> dist/stage3-1.0.0-x86_64.tar.xz
```

### Build the live ISO

```sh
make iso            # -> dist/iso/antenora-*.iso
```

This compiles the entire base system (glibc, coreutils, bash, util-linux, s6,
s6-linux-init, the kernel, ...) from source, wraps it in a busybox initramfs
and a squashfs, and assembles a hybrid BIOS/UEFI ISO with `grub-mkrescue`.

### Everything

```sh
make all
```

### Prebuilt releases

Tagged releases are built by CI and ship a ready-to-boot ISO:

```sh
# manual
make iso
# CI (on GitHub)
git tag v1.0.0 && git push origin v1.0.0
```

---

## Installing

Antenora is installed **by hand**. There is no guided installer — you follow the
[Antenora Handbook](install-guide.md), a step-by-step descent:

1. partition (`cfdisk`)
2. format root as XFS (`ftype=1 inode64`), EFI as FAT32
3. mount + extract the Stage3 tarball
4. `chroot`
5. configure (`/etc/fstab`, hostname, locale, timezone)
6. `dante install base-system linux-cachyos-bore linux-firmware` (+ a desktop)
7. initramfs + s6-linux-init (`mkinitramfs.sh`, `s6-init-setup.sh`)
8. `grub-install` + `grub-mkconfig`
9. user + `wheel` + sudo + SSH keys
10. reboot into the promised land

ZRAM (`lz4`) is automatic. The `nftables` firewall blocks everything inbound
except SSH and ping. First boot generates SSH keys, enables ZRAM, and loads the
firewall.

---

## Performance

`/etc/sysctl.d/99-antenora-performance.conf` ships `vm.vfs_cache_pressure=50`,
`vm.swappiness=10`, `kernel.numa_balancing=0`, BBR congestion control, and
larger network buffers. The kernel runs at 1000Hz with the BORE scheduler tuned
for minimal input latency and maximum FPS.

---

## License

**GNU Affero General Public License v3.0** ([LICENSE](LICENSE)).

Antenora is free software: you may copy, modify, and redistribute it under the
terms of the AGPL-3.0. If you run a modified Antenora on a network server, you
must offer its users the source. The trident crest is drawn in blood. Use it
well.
