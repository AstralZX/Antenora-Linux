# Dante User Repository (DUR)

The **DUR** is Antenora's community package repository. It is a *public repo
anyone can commit to*. Any user may submit a `.hell` recipe; maintainers review
submissions and may promote stable packages into the official repositories.

This mirrors the AUR model: the DUR holds build recipes, not binaries. Trust
nothing, review everything.

## Contributing

1. Write a `.hell` recipe for your package (see the [Hell language spec]).
2. Validate it:
   ```sh
   dante compile your-package.hell -
   ```
3. Submit it:
   ```sh
   scripts/dur-submit.sh your-package.hell
   ```
   or open a pull request adding your recipe under `recipes/`.

## Rules

- A recipe must be **50 lines or fewer** (`BLASPHEMY_ERROR` otherwise).
- Recipes must build from source; no prebuilt blobs unless a signed binary
  fallback is declared with a SHA-256 and a `binary` block.
- Do not submit malware, miners, or anything that would embarrass the ninth
  circle.

## Installing from the DUR

```sh
dante sync          # pulls official repos + the DUR
dante install -A pkgname   # install from the DUR
dante dur           # list DUR packages
dante dur-search term
```

[Hell language spec]: https://github.com/AstralZX/Antenora-Linux
