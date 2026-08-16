#!/usr/bin/env bash
# =============================================================================
# split-repos.sh - split the Antenora monorepo into its satellite repositories
# =============================================================================
# Extracts and pushes standalone repositories from this monorepo:
#
#   dante              the package manager alone  (cmd/ pkg/ go.mod go.sum)
#   antenora-packages  the central package repo   (recipes/ + manifest.yaml)
#   antenora-dur       the Dante User Repository  (dur/)
#
# Requires: `gh` authenticated (to create the repos) or pre-existing remotes.
# =============================================================================
set -euo pipefail

OWNER="${GITHUB_OWNER:-AstralZX}"
BASE="git@github.com:${OWNER}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="${TMPDIR:-/tmp}/antenora-split"

LICENSE="$ROOT/LICENSE"

ensure_repo() {
    local name="$1"
    if ! gh repo view "$OWNER/$name" >/dev/null 2>&1; then
        gh repo create "$OWNER/$name" --public
    fi
}

commit_and_push() {
    local dir="$1" url="$2" msg="$3"
    cd "$dir"
    git init -b main -q
    git add -A
    git -c user.email='208268648+AstralZX@users.noreply.github.com' \
        -c user.name="$OWNER" commit -q -m "$msg" || true
    git remote add origin "$url"
    git push -u origin main
    cd - >/dev/null
}

rm -rf "$TMP"
mkdir -p "$TMP"

# ---------------------------------------------------------------------------
# dante — the package manager alone
# ---------------------------------------------------------------------------
echo ":: Preparing dante ..."
ensure_repo dante
D="$TMP/dante"
mkdir -p "$D"
git archive HEAD cmd pkg go.mod go.sum | tar -x -C "$D"
cp "$LICENSE" "$D/LICENSE"
cat > "$D/README.md" <<'EOF'
# Dante — the Antenora package manager

Dante is a source-based package manager with signed binary fallbacks, written
in Go. It manages packages described by the **Hell** recipe language and is
the package manager for [Antenora Linux](https://github.com/AstralZX/Antenora-Linux).

## Features

- **Hell** recipes (`.hell`) — a minimalistic Makefile/Lua hybrid, capped at
  50 lines (`BLASPHEMY_ERROR`).
- Topological dependency resolution with full cycle reporting.
- SHA-256 + GPG-verified binary packages with automatic source fallback.
- Multiple repositories + a public **Dante User Repository (DUR)**.
- Ordered mirror failover, version-aware upgrades, and a bootstrap toolchain.

## Build

```sh
go build ./cmd/dante
```

## Usage

```sh
dante sync
dante install base-system
dante install -A some-dur-package
dante update
```

## License

[AGPL-3.0](LICENSE)
EOF
commit_and_push "$D" "$BASE/dante.git" "Import Dante package manager"

# ---------------------------------------------------------------------------
# antenora-packages — the central package repository
# ---------------------------------------------------------------------------
echo ":: Preparing antenora-packages ..."
ensure_repo antenora-packages
P="$TMP/antenora-packages"
mkdir -p "$P"
git archive HEAD recipes | tar -x -C "$P"
mv "$P/recipes/manifest.yaml" "$P/manifest.yaml"
cp "$LICENSE" "$P/LICENSE"
cat > "$P/README.md" <<'EOF'
# Antenora Packages

The central package repository for [Antenora Linux](https://github.com/AstralZX/Antenora-Linux).
Every package is a **Hell** recipe (`.hell`). `manifest.yaml` carries
descriptions and signed binary metadata.

## Layout

```
recipes/          # .hell build recipes (source of truth)
manifest.yaml     # descriptions + binary hashes
```

## Contributing

1. Write a `.hell` recipe (≤ 50 lines, or the parser throws `BLASPHEMY_ERROR`).
2. Regenerate the manifest: `dante gen-manifest .`
3. Open a pull request.

## License

[AGPL-3.0](LICENSE)
EOF
commit_and_push "$P" "$BASE/antenora-packages.git" "Import Antenora package repository"

# ---------------------------------------------------------------------------
# antenora-dur — the Dante User Repository
# ---------------------------------------------------------------------------
echo ":: Preparing antenora-dur ..."
ensure_repo antenora-dur
U="$TMP/antenora-dur"
mkdir -p "$U"
git archive HEAD dur | tar -x -C "$U"
mv "$U/dur/recipes" "$U/recipes"
rm -rf "$U/dur"
cp "$LICENSE" "$U/LICENSE"
cat > "$U/README.md" <<'EOF'
# Dante User Repository (DUR)

The **DUR** is Antenora's community package repository — a *public repo anyone
can commit to*. Submit a `.hell` recipe and install it with:

```sh
dante install -A pkgname
```

## Contributing

1. Write a `.hell` recipe for your package (≤ 50 lines).
2. Validate it: `dante compile your-package.hell -`
3. Submit it: add the recipe under `recipes/` and open a pull request.

## Rules

- Build from source; no prebuilt blobs unless a signed binary fallback is
  declared with a SHA-256.
- No malware, miners, or anything that would embarrass the ninth circle.

## License

[AGPL-3.0](LICENSE)
EOF
commit_and_push "$U" "$BASE/antenora-dur.git" "Import Dante User Repository"

echo
echo ":: Split complete:"
echo "   https://github.com/$OWNER/dante"
echo "   https://github.com/$OWNER/antenora-packages"
echo "   https://github.com/$OWNER/antenora-dur"
