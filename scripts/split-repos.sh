#!/usr/bin/env bash
# =============================================================================
# split-repos.sh - split the Antenora monorepo into its satellite repositories
# =============================================================================
# Extracts and pushes four standalone repositories from this monorepo:
#
#   dante              the package manager alone  (cmd/ pkg/ go.mod go.sum)
#   antenora-packages  the central package repo   (recipes/ + manifest.yaml)
#   antenora-dur       the Dante User Repository  (dur/)
#   Antenora-Linux     the OS + installer + ISO   (everything else)
#
# Each target repo must already exist on GitHub. Push happens over SSH.
# =============================================================================
set -euo pipefail

OWNER="${GITHUB_OWNER:-AstralZX}"
BASE="git@github.com:${OWNER}"

TMP="${TMPDIR:-/tmp}/antenora-split"
rm -rf "$TMP"
mkdir -p "$TMP"

split() {
    local name="$1"; shift
    echo
    echo ":: Preparing $name ..."
    local dir="$TMP/$name"
    mkdir -p "$dir"
    git archive HEAD "$@" | tar -x -C "$dir"
}

# dante — the package manager
split dante cmd pkg go.mod go.sum
cp "$(git rev-parse --show-toplevel)/Makefile" "$TMP/dante/Makefile" 2>/dev/null || true

# antenora-packages — recipes + manifest
split antenora-packages recipes

# antenora-dur — the DUR
split antenora-dur dur

for repo in dante antenora-packages antenora-dur; do
    dir="$TMP/$repo"
    [ -d "$dir" ] || { echo "!! $dir missing, skipping"; continue; }
    cd "$dir"
    git init -b main -q
    git add -A
    git -c user.email='208268648+AstralZX@users.noreply.github.com' \
        -c user.name="$OWNER" commit -q -m "Split $repo from Antenora monorepo" || true
    echo ":: Pushing $repo -> $BASE/$repo.git"
    git remote add origin "$BASE/$repo.git"
    git push -u origin main || echo "!! push failed for $repo (does $BASE/$repo.git exist?)"
    cd - >/dev/null
done

echo
echo ":: Split complete. If pushes failed, create the repos first:"
echo "   gh repo create $OWNER/dante --public"
echo "   gh repo create $OWNER/antenora-packages --public"
echo "   gh repo create $OWNER/antenora-dur --public"
