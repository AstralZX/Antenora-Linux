# Antenora Handbook

> *Nel mezzo del cammin di nostra vita — in the middle of the journey of our life…*

Welcome to the Antenora Handbook. It will take you from a blank disk to a
fully installed, self-compiled Antenora system — the whole journey, one page,
no hand-holding, no surprises.

Antenora is installed **by hand**, like Gentoo but a little easier. There is no
guided installer. Every command here is one you type, and therefore one you
understand. This is the entire guide.

---

## Part I — Installing Antenora

### 1. Introduction & requirements

Antenora is a source-based distribution. The base system is compiled from
source by **Dante**, its package manager. The init system is **s6 + s6-rc**
(systemd is forbidden), the libc is **glibc**, the root filesystem is **XFS**.

**What you need:**

- An `x86_64` machine, UEFI or BIOS.
- A live medium: the Antenora ISO, or any Linux live system.
- Network access.
- At least 4 GB RAM and ~20 GB of disk.
- A root shell, and the will.

**Conventions:** `/dev/sda` is the disk. `/dev/sda1` is the EFI partition
(UEFI). `/dev/sda2` is root. **Replace them with your own.**

### 2. Choosing the installation medium

Three ways in:

1. **The Antenora live ISO** — boot it, open a terminal. This is the blessed path.
2. **Another Linux live system** — any Arch/Gentoo/Debian live ISO works; you
   only need `tar`, `xfsprogs`, and `chroot`.
3. **An existing Linux install** — you can install Antenora onto a spare
   partition or a chroot from a running system.

Build the ISO yourself if you prefer:

```sh
make iso        # in the Antenora source tree
```

### 3. Configuring the network

The live environment uses NetworkManager.

```sh
nmtui                       # text UI — easiest
# or, for Wi-Fi:
nmcli device wifi connect "SSID" password "pass"
# or wired:
dhclient
```

Verify:

```sh
ping -c3 example.com
```

### 4. Preparing the disks

**Verify boot mode:**

```sh
ls /sys/firmware/efi/efivars   # exists → UEFI; missing → BIOS
```

**Partition with cfdisk:**

```sh
cfdisk /dev/sda
```

- **UEFI:** an EFI System Partition (≥ 300 MiB, type *EFI System*) + a root
  partition. (GRUB can also use a `/boot` on the root filesystem.)
- **BIOS:** a root partition. Optionally a 1 MiB BIOS boot partition
  (type *BIOS boot*, no filesystem) for GPT disks.

**Partitioning schemes:**

| Scheme | Partitions |
|--------|-----------|
| UEFI + GPT | `/dev/sda1` EFI (FAT32, 300M) · `/dev/sda2` root (XFS) |
| BIOS + MBR | `/dev/sda1` root (XFS) |
| BIOS + GPT | `/dev/sda1` BIOS boot (1M, unformatted) · `/dev/sda2` root (XFS) |

### 5. Formatting the filesystems

The Third Commandment: root is **XFS**.

```sh
# root — XFS, ftype=1, inode64
mkfs.xfs -f -n ftype=1 -d inode64 /dev/sda2

# EFI (UEFI) — FAT32
mkfs.fat -F32 /dev/sda1
```

**Optional: full-disk encryption (LUKS).**

```sh
cryptsetup luksFormat /dev/sda2
cryptsetup open /dev/sda2 root
mkfs.xfs -f -n ftype=1 -d inode64 /dev/mapper/root
```

Then use `/dev/mapper/root` wherever the guide says `/dev/sda2`.

### 6. Mounting

```sh
mount /dev/sda2 /mnt
mkdir -p /mnt/boot
mount /dev/sda1 /mnt/boot    # UEFI only
```

### 7. Installing the Stage3

The Stage3 is the self-hosting base: s6, glibc, and the Dante package manager
solved the chicken-and-egg problem for you.

```sh
wget https://github.com/AstralZX/Antenora-Linux/releases/latest/download/stage3-1.0.0-x86_64.tar.xz
tar -xJpf stage3-1.0.0-x86_64.tar.xz -C /mnt --numeric-owner
```

Or build it yourself: `make stage3`.

### 8. Chrooting

```sh
cp /etc/resolv.conf /mnt/etc/resolv.conf
mount --bind /dev /mnt/dev
mount --bind /dev/pts /mnt/dev/pts
mount --bind /proc /mnt/proc
mount --bind /sys /mnt/sys
mount --bind /run /mnt/run
chroot /mnt /bin/bash
```

You are now inside the new system.

### 9. Configuring the system

```sh
# identity
echo "antenora" > /etc/hostname

# filesystem table
cat > /etc/fstab <<EOF
/dev/sda2 /     xfs  defaults,noatime 0 1
/dev/sda1 /boot vfat defaults         0 2
EOF

# timezone
ln -sf /usr/share/zoneinfo/Europe/Berlin /etc/localtime

# locale
nano /etc/locale.gen      # uncomment en_US.UTF-8 UTF-8
locale-gen
echo 'LANG=en_US.UTF-8' > /etc/locale.conf
```

### 10. Installing packages

```sh
dante sync
dante install base-system          # glibc, s6, s6-linux-init, coreutils, bash, util-linux, busybox
dante install linux-cachyos-bore   # the kernel (1000 Hz, BORE scheduler)
dante install linux-firmware
```

Then a desktop — or none:

```sh
dante install plasma    # KDE
dante install gnome     # GNOME
dante install sway      # SwayWM
# nothing → headless warrior mode
```

### 11. Configuring the bootloader

GRUB 2 is the Gatekeeper. It generates the initramfs and installs the boot
scripts first:

```sh
/usr/lib/antenora/s6-init-setup.sh   # s6-linux-init as PID 1 + s6-rc database
/usr/lib/antenora/mkinitramfs.sh / /boot/initramfs-antenora.img boot/disk-boot/init
```

Then:

```sh
# UEFI
grub-install --target=x86_64-efi --efi-directory=/boot --bootloader-id=Antenora
# BIOS
grub-install --target=i386-pc /dev/sda

grub-mkconfig -o /boot/grub/grub.cfg
```

A 3-second timeout, the dark red/black theme, the pitchfork crest.

### 12. Finalizing

```sh
# users
useradd -m -G wheel -s /usr/bin/zsh you
passwd you
passwd root
echo '%wheel ALL=(ALL:ALL) ALL' > /etc/sudoers.d/10-wheel

# your SSH key
su - you
ssh-keygen -t ed25519 -C 'you@antenora'
exit
```

### 13. Rebooting

```sh
exit                    # leave the chroot
umount -R /mnt
reboot
```

On first boot, `/usr/lib/antenora/firstboot.sh` enables ZRAM (`lz4`), generates
missing SSH keys, and loads the nftables firewall (SSH and ping only). A winged
lion appears. Antenora rises from the ashes of bloat.

---

## Part II — Working with Dante

### 14. Package management

Dante is source-based with signed binary fallbacks.

```sh
dante sync                 # update repos + DUR
dante install <pkg>        # resolve deps, compile, install
dante install -A <pkg>     # from the DUR
dante remove <pkg>         # uninstall + reverse-dep warnings
dante remove --orphans     # drop now-unneeded dependencies
dante search <term>        # search names + descriptions
dante info <pkg>           # metadata + binary availability
dante update               # version-aware upgrade
dante clean                # purge caches
dante toolchain            # bootstrap the build toolchain
dante mirror               # test + rank binary mirrors
dante key                  # import the maintainer GPG key
dante dur                  # list DUR packages
dante dur-search <term>    # search the DUR
dante compile in.hell out.mk   # compile a recipe to a Makefile
```

### 15. The Hell language

Packages are written in **Hell** (`.hell`). See the full [Hell language
guide](hell.html) — a recipe is ~20 keywords, a 50-line cap, and impossible to
write poorly.

### 16. The DUR

The **Dante User Repository** is a public repo anyone can commit to. Submit a
recipe, then install it:

```sh
scripts/dur-submit.sh my-package.hell   # contribute
dante install -A my-package             # install
```

### 17. Configuration & mirrors

`/etc/dante/dante.conf`:

```conf
BINARY="NO"        # "YES" prefers signed binary packages
MAKEFLAGS="-j4"    # parallel jobs (raise only with more RAM)
PARALLEL="4"
REPO_URL="https://github.com/AstralZX/antenora-packages.git"
DUR_URL="https://github.com/AstralZX/antenora-dur.git"
BINARY_MIRROR="https://cdn.antenora.org/packages"
MIRROR_FILE="/etc/dante/mirrors.conf"
CLEAN_SOURCE="YES"
KEEP_DEPS="NO"
```

Every key is overridable per-command with a `DANTE_*` env var:

```sh
DANTE_BINARY=YES dante install linux-cachyos-bore
```

Mirrors (`/etc/dante/mirrors.conf`) are one URL per line, ordered by
preference. `dante mirror` ranks them by latency.

### 18. The toolchain

To build everything from nothing:

```sh
dante toolchain
# or, for a sysroot:
scripts/toolchain.sh
```

It bootstraps `linux-api-headers → gmp/mpfr/mpc/isl → binutils → gcc → make →
autotools → cmake/meson/ninja`.

### 19. Binary packages

Binary packages are SHA-256 checksummed **and** GPG-signed. Dante downloads,
verifies the checksum, verifies the signature, then extracts. Any failure
falls back to compiling from source automatically.

---

## Part III — Working with Antenora

### 20. s6 & service management

Every service is a script in `/etc/s6-rc/source/`. The supervisor is `s6-rc`.

```sh
s6-rc -u change <service>     # start
s6-rc -d change <service>     # stop
s6-rc -a list                 # list all
s6-rc -a list <service>       # inspect

# aliases (in /etc/profile.d/antenora-aliases.sh)
s6-start <service>
s6-stop  <service>
s6-status
```

To enable a service at boot, compile the database and it is supervised:

```sh
/usr/lib/antenora/s6-init-setup.sh
```

Long-running services (`agetty`, `dbus`, `NetworkManager`, `pipewire`,
`wireplumber`, `sddm`/`gdm`/`greetd`) ship in the base. Each is a trivial
script you can read and edit.

### 21. Logging

s6 logs are managed with `s6-log`; service logs land in `/var/log/`. Inspect:

```sh
tail -f /var/log/<service>/current
```

### 22. Users & permissions

```sh
useradd -m -G wheel -s /usr/bin/zsh name
passwd name
usermod -aG wheel name     # grant sudo
```

`%wheel ALL=(ALL:ALL) ALL` is in `/etc/sudoers.d/10-wheel`.

### 23. Networking

NetworkManager is the default:

```sh
nmtui                       # text UI
nmcli device wifi connect "SSID" password "pass"
nmcli device status
```

The nftables firewall (`/etc/nftables.conf`) blocks all inbound except SSH and
ping, statefully.

### 24. Time & localization

```sh
timedatectl set-timezone Europe/Berlin
timedatectl set-ntp true
locale-gen
```

### 25. Performance

`/etc/sysctl.d/99-antenora-performance.conf` ships the canonical tweaks:

```ini
vm.vfs_cache_pressure = 50
vm.swappiness = 10
kernel.numa_balancing = 0
```

ZRAM (`lz4`) is automatic. The kernel runs at 1000 Hz with the BORE scheduler.

### 26. Upgrading

```sh
dante sync
dante update
```

Rebuild the initramfs after a kernel change:

```sh
/usr/lib/antenora/mkinitramfs.sh / /boot/initramfs-antenora.img boot/disk-boot/init
grub-mkconfig -o /boot/grub/grub.cfg
```

### 27. Troubleshooting

**Boot fails — dropped to busybox shell.** The initramfs couldn't mount the
root. Check the `root=` kernel parameter, or `fsck` the XFS filesystem:
`xfs_repair /dev/sda2`.

**A package fails to build.** The recipe is over 50 lines, or a dependency is
missing. Read the error, `dante info <pkg>`, and file a bug.

**A service won't start.** `s6-rc -a list <service>` shows its state; read its
`run` script and the log.

**Recompiled the world and something broke.** `dante clean` then reinstall.

---

*A winged lion appears. You have built Antenora from nothing. The Ninth Circle
of Speed is yours.*
