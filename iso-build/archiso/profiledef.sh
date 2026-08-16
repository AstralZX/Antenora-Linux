#!/usr/bin/env bash
# Antenora mkarchiso profile definition.
# Boots s6, ships Dante, drops you into the Gates of Hell installer.

iso_name="antenora"
iso_label="ANTENORA_$(date +%Y%m)"
iso_publisher="Antenora Project <https://antenora.org>"
iso_application="Antenora Linux Live/Rescue CD — The Ninth Circle of Speed"
iso_version="${1:-1.0.0}"
install_dir="antenora"
buildmodes=('iso')
bootmodes=('bios.syslinux.mbr' 'bios.syslinux.eltorito'
           'uefi-x64.grub.esp' 'uefi-x64.grub.eltorito')
arch="x86_64"
pacman_conf="pacman.conf"
airootfs_image_type="squashfs"
airootfs_image_tool_options=('-comp' 'xz' '-Xbcj' 'x86' '-b' '1M' '-Xdict-size' '1M')
bootstrap_tarball_compression=(gzip -cn9)
file_permissions=(
  ["/etc/shadow"]="0:0:0400"
  ["/etc/gshadow"]="0:0:0400"
  ["/usr/bin/dante"]="0:0:0755"
  ["/usr/local/bin/install-antenora"]="0:0:0755"
)
