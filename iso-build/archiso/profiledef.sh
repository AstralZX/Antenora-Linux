#!/usr/bin/env bash
# Antenora mkarchiso profile definition.
# Boots a minimal live environment, ships Dante, drops you into the Gates of
# Hell installer. (systemd-boot here boots only the live media; the installed
# system is s6/s6-rc, systemd-free.)

iso_name="antenora"
iso_label="ANTENORA_$(date --date="@${SOURCE_DATE_EPOCH:-$(date +%s)}" +%Y%m)"
iso_publisher="Antenora Project <https://github.com/AstralZX/Antenora-Linux>"
iso_application="Antenora Linux Live/Rescue CD — The Ninth Circle of Speed"
iso_version="${1:-1.0.0}"
install_dir="antenora"
buildmodes=('iso')
bootmodes=('bios.syslinux'
           'uefi.systemd-boot')
arch="x86_64"
pacman_conf="pacman.conf"
airootfs_image_type="squashfs"
airootfs_image_tool_options=('-comp' 'xz' '-Xbcj' 'x86' '-b' '1M' '-Xdict-size' '1M')
bootstrap_tarball_compression=('zstd' '-c' '-T0' '--auto-threads=logical' '--long' '-19')
file_permissions=(
  ["/usr/bin/dante"]="0:0:0755"
  ["/usr/local/bin/install-antenora"]="0:0:0755"
  ["/usr/lib/antenora/firstboot.sh"]="0:0:0755"
)
