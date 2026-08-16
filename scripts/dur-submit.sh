#!/usr/bin/env bash
# =============================================================================
# dur-submit.sh - submit a .hell recipe to the Dante User Repository (DUR)
# =============================================================================
# The DUR is a public repository anyone can commit to. This script clones it,
# adds your recipe, and pushes. A maintainer then reviews and may promote the
# package into the official repositories.
#
# Usage: dur-submit.sh path/to/my-package.hell
# =============================================================================
set -euo pipefail

DUR_URL="${DUR_URL:-https://github.com/AstralZX/antenora-dur.git}"
WORKDIR="${TMPDIR:-/tmp}/dur-submit"

[ $# -eq 1 ] || { echo "usage: $0 <package.hell>"; exit 1; }
RECIPE="$1"
[ -f "$RECIPE" ] || { echo "error: $RECIPE not found"; exit 1; }

NAME="$(basename "$RECIPE" .hell)"

echo ":: Cloning the DUR..."
rm -rf "$WORKDIR"
git clone --depth 1 "$DUR_URL" "$WORKDIR"

mkdir -p "$WORKDIR/recipes"
cp "$RECIPE" "$WORKDIR/recipes/"

# validate the recipe parses before pushing
if command -v dante >/dev/null 2>&1; then
    dante compile "$RECIPE" - >/dev/null || { echo "error: recipe failed to parse"; exit 1; }
fi

cd "$WORKDIR"
git add "recipes/$NAME.hell"
git commit -m "dur: add $NAME" || { echo "!! nothing to commit"; exit 0; }
git push origin HEAD

echo ":: Submitted $NAME to the DUR."
echo ":: Install it with: dante install -A $NAME"
