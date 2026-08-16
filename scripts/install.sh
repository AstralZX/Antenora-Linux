#!/usr/bin/env bash
# =============================================================================
# THE GATES OF HELL INSTALLER
# Antenora Linux - "Inferno" 1.0.0
# =============================================================================
# A TTY installer that formats, seeds and boots the Ninth Circle of Speed.
# s6 for init. XFS for root. glibc for compatibility. No systemd. No mercy.
# =============================================================================
set -euo pipefail

VERSION="1.0.0"
CODENAME="Inferno"
STAGE3_URL="https://cdn.antenora.org/releases/stage3-${VERSION}-x86_64.tar.xz"
STAGE3_TARBALL="/root/stage3-antenora.tar.xz"
STAGE3_ROOT="/mnt/antenora"

# ---------------------------------------------------------------------------
# helpers
# ---------------------------------------------------------------------------
red()   { printf '\033[1;31m%b\033[0m\n' "$*"; }
dim()   { printf '\033[1;30m%b\033[0m\n' "$*"; }
bold()  { printf '\033[1m%b\033[0m\n' "$*"; }

have() { command -v "$1" >/dev/null 2>&1; }

banner() {
    red '================================================================'
    red '             T H E   G A T E S   O F   H E L L'
    red "      Antenora Linux ${VERSION} \"${CODENAME}\" Installer"
    red '================================================================'
    echo
}

crest() {
    cat <<'EOF'
                          \  |  /
                        -  ANTENORA  -
                        /  |  \  |  /
            A winged lion, forged in the ninth circle.
EOF
    echo
}

die() { red "FATAL: $*"; exit 1; }

# pick a tool for the progress gauge; prefer dialog, then whiptail, then plain
GAUGE=""
if have dialog; then
    GAUGE="dialog"
elif have whiptail; then
    GAUGE="whiptail"
fi

gauge() {
    local title="$1" msg="$2" pct="$3"
    case "$GAUGE" in
        dialog)  dialog --gauge "$msg" 8 70 "$pct" --title "$title" 2>/dev/null || true ;;
        whiptail) whiptail --gauge "$msg" 8 70 "$pct" --title "$title" 2>/dev/null || true ;;
        *)        echo "[$pct%] $msg" ;;
    esac
}

menu() {
    local title="$1" prompt="$2"; shift 2
    local opts=() i
    for i in "$@"; do opts+=("$i" ""); done
    if [ "$GAUGE" = "dialog" ]; then
        dialog --title "$title" --menu "$prompt" 20 70 8 "${opts[@]}" 2>&1 >/dev/tty
    elif [ "$GAUGE" = "whiptail" ]; then
        whiptail --title "$title" --menu "$prompt" 20 70 8 "${opts[@]}" 2>&1 >/dev/tty
    else
        read -r -p "$prompt (default 1): " ans
        echo "${ans:-1}"
    fi
}

ask() {
    local prompt="$1" default="$2"
    if [ "$GAUGE" = "dialog" ] || [ "$GAUGE" = "whiptail" ]; then
        read -r -p "$prompt [$default]: " ans
        echo "${ans:-$default}"
    else
        read -r -p "$prompt [$default]: " ans
        echo "${ans:-$default}"
    fi
}

# ---------------------------------------------------------------------------
# 0. preamble
# ---------------------------------------------------------------------------
[ "$(id -u)" -eq 0 ] || die "run as root, ye poor soul"
banner

if [ ! -f "$STAGE3_TARBALL" ]; then
    bold ":: Downloading Stage3 tarball..."
    wget -q --show-progress -O "$STAGE3_TARBALL" "$STAGE3_URL" || die "failed to fetch Stage3"
fi

# ---------------------------------------------------------------------------
# 1. hardware detection
# ---------------------------------------------------------------------------
bold ":: Detecting hardware..."
have lspci && lspci > /var/log/antenora-lspci.log 2>/dev/null || true
have lsusb && lsusb > /var/log/antenora-lsusb.log 2>/dev/null || true

FIRMWARE="linux-firmware"
have lspci && lspci | grep -qi "NVIDIA" && FIRMWARE="$FIRMWARE nvidia-firmware"
have lspci && lspci | grep -qi "AMD"    && FIRMWARE="$FIRMWARE amd-ucode"
have lspci && lspci | grep -qi "Intel"  && FIRMWARE="$FIRMWARE intel-ucode"
dim ":: Firmware set: $FIRMWARE"

# ---------------------------------------------------------------------------
# 2. partitioning
# ---------------------------------------------------------------------------
bold ":: Choose the disk to be sacrificed. It WILL be wiped."
DISK=$(lsblk -d -n -p -o NAME,SIZE,MODEL | sed 's/  */ /g')
echo "$DISK"
TARGET=$(ask "Target disk (e.g. /dev/sda)" "/dev/sda")

bold ":: Launching cfdisk on $TARGET — create at least a root partition"
bold ":: (and a FAT32 EFI partition if this is a UEFI system)."
cfdisk "$TARGET"

# detect EFI
EFI_PART=""
if [ -d /sys/firmware/efi ]; then
    EFI_PART=$(lsblk -n -r -p -o NAME,PARTTYPE "$TARGET" | grep -i 'c12a7328-f81f-11d2' | awk '{print $1}' | head -n1)
fi
ROOT_PART=$(ask "Root partition (e.g. ${TARGET}2)" "${TARGET}2")

if [ -z "$ROOT_PART" ]; then die "no root partition specified"; fi

# ---------------------------------------------------------------------------
# 3. format: XFS for root, FAT32 for EFI
# ---------------------------------------------------------------------------
bold ":: Formatting $ROOT_PART as XFS (ftype=1 inode64)..."
mkfs.xfs -f -m reflink=0,rmapbt=0 -i sparse=0 -n ftype=1 -d inode64 "$ROOT_PART"

if [ -n "$EFI_PART" ]; then
    bold ":: Formatting $EFI_PART as FAT32 (EFI)..."
    mkfs.fat -F32 "$EFI_PART"
fi

# ---------------------------------------------------------------------------
# 4. mount + extract Stage3
# ---------------------------------------------------------------------------
mkdir -p "$STAGE3_ROOT"
mount "$ROOT_PART" "$STAGE3_ROOT"
if [ -n "$EFI_PART" ]; then
    mkdir -p "$STAGE3_ROOT/boot"
    mount "$EFI_PART" "$STAGE3_ROOT/boot"
fi

bold ":: Extracting Stage3 tarball..."
gauge "Antenora" "Extracting base system..." 10
tar -xJpf "$STAGE3_TARBALL" -C "$STAGE3_ROOT" --numeric-owner

# seed fstab (XFS root, optional EFI)
cat > "$STAGE3_ROOT/etc/fstab" <<EOF
$ROOT_PART / xfs defaults,noatime 0 1
$([ -n "$EFI_PART" ] && echo "$EFI_PART /boot vfat defaults 0 2")
EOF

# ---------------------------------------------------------------------------
# 5. chroot setup
# ---------------------------------------------------------------------------
cp /etc/resolv.conf "$STAGE3_ROOT/etc/resolv.conf" 2>/dev/null || true
mount --bind /dev "$STAGE3_ROOT/dev"
mount --bind /dev/pts "$STAGE3_ROOT/dev/pts"
mount --bind /proc "$STAGE3_ROOT/proc"
mount --bind /sys "$STAGE3_ROOT/sys"
mount --bind /run "$STAGE3_ROOT/run" 2>/dev/null || mkdir -p "$STAGE3_ROOT/run"

HOSTNAME=$(ask "Hostname" "antenora")
echo "$HOSTNAME" > "$STAGE3_ROOT/etc/hostname"

chroot_run() { chroot "$STAGE3_ROOT" /bin/bash -c "$1"; }

# ---------------------------------------------------------------------------
# 6. kernel + base system + DE
# ---------------------------------------------------------------------------
KERNEL_MODE=$(menu "Kernel" "Kernel strategy:" \
    "binary" "prebuilt cachyos-bore" "source" "compile now (masochist)")
if [ "$KERNEL_MODE" = "source" ]; then
    bold ":: Compiling linux-cachyos-bore from source. Brew coffee."
    gauge "Antenora" "Compiling kernel..." 30
    chroot_run "DANTE_BINARY=NO dante install linux-cachyos-bore"
else
    chroot_run "DANTE_BINARY=YES dante install linux-cachyos-bore"
fi

bold ":: Installing base system + firmware..."
gauge "Antenora" "Installing base system..." 55
chroot_run "dante install base-system"
chroot_run "dante install $FIRMWARE"

DE_CHOICE=$(menu "Desktop" "Choose your desktop environment:" \
    "plasma" "KDE Plasma (full)" \
    "gnome"  "GNOME (minimal)" \
    "sway"   "SwayWM (rice-ready)" \
    "none"   "None (headless warrior mode)")
case "$DE_CHOICE" in
    plasma) chroot_run "dante install plasma" ;;
    gnome)  chroot_run "dante install gnome" ;;
    sway)   chroot_run "dante install sway" ;;
    none)   dim ":: Headless warrior mode. Respect." ;;
esac

# ---------------------------------------------------------------------------
# 7. user, sudo, ssh keys
# ---------------------------------------------------------------------------
USERNAME=$(ask "Username" "user")
chroot_run "useradd -m -G wheel -s /usr/bin/zsh $USERNAME"
bold ":: Set a password for $USERNAME:"
chroot_run "passwd $USERNAME"
chroot_run "passwd root"
chroot_run "echo '%wheel ALL=(ALL:ALL) ALL' > /etc/sudoers.d/10-wheel"

bold ":: Generating SSH keys for $USERNAME..."
chroot_run "sudo -u $USERNAME ssh-keygen -t ed25519 -f /home/$USERNAME/.ssh/id_ed25519 -N '' -C '$USERNAME@$HOSTNAME' || true"

# zram
bold ":: Enabling ZRAM (lz4, no swap partition)..."
chroot_run "zramctl --find --size 50% --algorithm lz4 || true"
chroot_run "mkswap /dev/zram0 && swapon /dev/zram0 || true"

# firewall
chroot_run "nft -f /etc/nftables.conf || true"

# sysctl
chroot_run "sysctl --system || true"

# ---------------------------------------------------------------------------
# 7b. initramfs (busybox, no dracut/systemd) + s6-linux-init
# ---------------------------------------------------------------------------
bold ":: Generating initramfs and installing s6-linux-init..."
chroot_run "/usr/lib/antenora/s6-init-setup.sh"
chroot_run "/usr/lib/antenora/mkinitramfs.sh / /boot/initramfs-antenora.img boot/disk-boot/init"

# ---------------------------------------------------------------------------
# 8. bootloader
# ---------------------------------------------------------------------------
bold ":: Installing GRUB and generating config..."
if [ -n "$EFI_PART" ]; then
    chroot_run "grub-install --target=x86_64-efi --efi-directory=/boot --bootloader-id=Antenora"
else
    chroot_run "grub-install --target=i386-pc $TARGET"
fi
chroot_run "grub-mkconfig -o /boot/grub/grub.cfg"

# ---------------------------------------------------------------------------
# 9. finalize
# ---------------------------------------------------------------------------
bold ":: Unmounting..."
umount -R "$STAGE3_ROOT" 2>/dev/null || true

# ---------------------------------------------------------------------------
# 10. the winged lion
# ---------------------------------------------------------------------------
clear
red '============================================================'
red '        A N T E N O R A   R I S E S   F R O M   T H E'
red '            A S H E S   O F   B L O A T'
red '============================================================'
crest
echo
bold "Installation complete. Reboot into the promised land."
if [ "$GAUGE" != "" ]; then
    if dialog --yesno "Reboot now?" 6 40 2>/dev/null; then reboot; fi
else
    read -r -p "Reboot now? [y/N]: " ans
    [ "$ans" = "y" ] && reboot
fi
