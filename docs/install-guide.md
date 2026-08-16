# The Antenora Handbook — Manual Installation

> *Nel mezzo del cammin di nostra vita…*

Antenora is installed **by hand**. There is no guided installer, no wizard, no
"express" mode — only you, a root shell, and the raw silicon. This is the whole
point. Every command below is one you understand, because you typed it.

The result is a machine you built from the bootloader up. Minimal. Yours.

---

## 0. What you need

- A bootable Antenora live medium (any Linux live system works too), or a
  running Linux to install *from*.
- An `x86_64` machine, UEFI or BIOS.
- Network access to fetch the Stage3 tarball.
- A cup of something strong.

Conventions used below: `/dev/sda` is the disk, `/dev/sda1` the EFI partition
(UEFI), `/dev/sda2` the root. **Substitute your own.**

---

## 1. Verify boot mode

```sh
ls /sys/firmware/efi/efivars
```

If the directory exists, you are in **UEFI** mode (make an EFI partition). If
not, you are in BIOS mode.

---

## 2. Partition the disk

```sh
cfdisk /dev/sda
```

- **UEFI:** an EFI System Partition (≥ 300 MiB, type *EFI System*), and a root
  partition.
- **BIOS:** a root partition. Optionally a small BIOS boot partition for GRUB.

---

## 3. Format the filesystems

The Third Commandment: root is **XFS**. No warnings, no confirmation.

```sh
# root — XFS, ftype=1, inode64
mkfs.xfs -f -n ftype=1 -d inode64 /dev/sda2

# EFI (UEFI only) — FAT32
mkfs.fat -F32 /dev/sda1
```

---

## 4. Mount the filesystems

```sh
mount /dev/sda2 /mnt
mkdir -p /mnt/boot
mount /dev/sda1 /mnt/boot    # UEFI only
```

---

## 5. Fetch and extract the Stage3

The Stage3 is the self-hosting base: s6, glibc, and the Dante package manager.
It solves the bootstrap problem — Dante installs itself because it ships inside.

```sh
wget https://github.com/AstralZX/Antenora-Linux/releases/latest/download/stage3-1.0.0-x86_64.tar.xz
tar -xJpf stage3-1.0.0-x86_64.tar.xz -C /mnt --numeric-owner
```

(Or build your own: `make stage3` in the source tree.)

---

## 6. Enter the new system (chroot)

```sh
cp /etc/resolv.conf /mnt/etc/resolv.conf
mount --bind /dev /mnt/dev
mount --bind /dev/pts /mnt/dev/pts
mount --bind /proc /mnt/proc
mount --bind /sys /mnt/sys
mount --bind /run /mnt/run
chroot /mnt /bin/bash
```

You are now inside Antenora, before it is Antenora.

---

## 7. Configure the system

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

# locale (uncomment your locale in /etc/locale.gen, then:)
locale-gen
echo 'LANG=en_US.UTF-8' > /etc/locale.conf
```

---

## 8. Sync and install the world

```sh
dante sync
dante install base-system          # glibc, s6, s6-linux-init, coreutils, bash, ...
dante install linux-cachyos-bore   # the kernel
dante install linux-firmware
```

Then a desktop — or none:

```sh
dante install plasma    # KDE
dante install gnome     # GNOME
dante install sway      # SwayWM
# or skip it entirely: headless warrior mode.
```

---

## 9. Initramfs and init

```sh
/usr/lib/antenora/s6-init-setup.sh
/usr/lib/antenora/mkinitramfs.sh / /boot/initramfs-antenora.img boot/disk-boot/init
```

The first installs **s6-linux-init** as PID 1 and compiles the s6-rc database;
the second builds a busybox initramfs. No dracut, no systemd.

---

## 10. Bootloader — the Gatekeeper

```sh
# UEFI
grub-install --target=x86_64-efi --efi-directory=/boot --bootloader-id=Antenora
# BIOS
grub-install --target=i386-pc /dev/sda

grub-mkconfig -o /boot/grub/grub.cfg
```

A 3-second timeout, the dark red/black theme, the pitchfork crest.

---

## 11. User, sudo, SSH

```sh
useradd -m -G wheel -s /usr/bin/zsh you
passwd you
passwd root

echo '%wheel ALL=(ALL:ALL) ALL' > /etc/sudoers.d/10-wheel

su - you
ssh-keygen -t ed25519 -C 'you@antenora'   # your own SSH key
exit
```

---

## 12. Leave and reboot

```sh
exit            # leave the chroot
umount -R /mnt
reboot
```

On first boot, `/usr/lib/antenora/firstboot.sh` enables ZRAM (`lz4`), generates
missing SSH keys, and loads the nftables firewall (SSH and ping only).

---

*A winged lion appears. Antenora rises from the ashes of bloat.*

Continue: [Package management](packages.html) · [The DUR](dur.html) · [The Nine Circles](circles.html)
