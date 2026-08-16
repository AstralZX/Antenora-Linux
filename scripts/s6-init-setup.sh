#!/bin/sh
# s6-init-setup.sh — set up s6-linux-init as PID 1 inside the sysroot.
# Runs inside a chroot. Compiles the s6-rc service database and generates the
# boot scripts (init/shutdown/halt/reboot/poweroff) with s6-linux-init-maker.
# No systemd anywhere.

export PATH=/usr/bin:/bin:/usr/sbin:/sbin

echo ":: Compiling s6-rc service database..."
s6-rc-compile /etc/s6-rc/compiled /etc/s6-rc/source

echo ":: Generating s6-linux-init boot scripts..."
INITDIR="$(mktemp -d)"
s6-linux-init-maker -c /etc/s6-rc/compiled -b /usr/bin -d /sbin -p "/usr/bin:/bin:/usr/sbin:/sbin" "$INITDIR"

install -m755 "$INITDIR/init" /sbin/init
for f in shutdown halt poweroff reboot stage2-fatal; do
    [ -f "$INITDIR/$f" ] && install -m755 "$INITDIR/$f" /sbin/
done

# machine-id / etc fragments produced by the maker
[ -d "$INITDIR/etc" ] && cp -a "$INITDIR/etc/." /etc/

rm -rf "$INITDIR"
echo ":: s6-linux-init installed. PID 1 is now s6."
