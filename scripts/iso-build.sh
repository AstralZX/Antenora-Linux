#!/usr/bin/env bash
# =============================================================================
# iso-build.sh - build the Antenora live ISO with mkarchiso
# =============================================================================
# Wraps mkarchiso with the Antenora overlay in iso-build/archiso so the live
# environment boots s6, carries the Dante package manager and launches the
# Gates of Hell installer.
# =============================================================================
set -euo pipefail

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PROFILE_DIR="$REPO_DIR/iso-build/archiso"
OUT_DIR="$REPO_DIR/dist/iso"
VERSION="${1:-1.0.0}"

have() { command -v "$1" >/dev/null 2>&1; }

have mkarchiso || { echo "error: mkarchiso not found (install archiso)"; exit 1; }

# ensure the live root contains Dante
echo ":: Building Dante for the live image..."
CGO_ENABLED=0 go -C "$REPO_DIR" build -trimpath -ldflags '-s -w' \
    -o "$PROFILE_DIR/airootfs/usr/bin/dante" ./cmd/dante

echo ":: Running mkarchiso..."
mkdir -p "$OUT_DIR"
mkarchiso -v -w /tmp/antenora-archiso -o "$OUT_DIR" "$PROFILE_DIR"

echo ":: ISO written to $OUT_DIR"
