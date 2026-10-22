# Karty Packages

Reusable code and assets for Karty games, versioned independently of the CLI
and SDK. Package releases contain portable `.kpkg` archives with SHA-256
checksums. This repository can build them without a released Karty CLI or SDK.

| Package                                       | Purpose                                                 |
| --------------------------------------------- | ------------------------------------------------------- |
| [Controllers](packages/controllers/README.md) | Movement and camera helpers                             |
| [Dialogue](packages/dialogue/README.md)       | Yarn dialogue runtime, compiler and UI                  |
| [FPS Kit](packages/fps-kit/README.md)         | 256px materials, crates, bumpers and placeholder decals |
| [Savegame](packages/savegame/README.md)       | Versioned saves and profile storage                     |
| [Tiled](packages/tiled/README.md)             | Tiled map import and runtime helpers                    |

## Develop and package locally

Install the tools pinned in `mise.toml`, then run:

```sh
mise install
mise run check
mise run package -- fps-kit
```

The last command creates `dist/releases/fps-kit-0.1.0.kpkg`, its `.sha256`
checksum and release metadata JSON. Code packages retain their own Go modules;
the root Go module contains only the release tooling. Resource packages such
as FPS Kit need no Go dependency. The checked-in FPS Kit generators live in
`scripts/fps-kit`; use `mise run generate-fps-kit` after editing their recipes.

Go tests use the committed dialogue fixtures. Building the dialogue compiler
also requires the .NET SDK pinned by `packages/dialogue/global.json`; CI
restores its locked NuGet dependencies and builds it before release.

## Release one package

Commit and push the package sources and workflows first. Set the package's
`karty-package.toml` version, then use **Actions → Release package → Run workflow**.
Choose the package and its version. Leave **publish** unchecked to validate
and download candidate artifacts without creating a release.

For example, publish FPS Kit after reviewing its candidate:

```sh
gh workflow run release.yml --repo karty-game/karty-packages \
  -f package=fps-kit -f version=0.1.0 -f publish=true
```

Alternatively, push a package-specific tag pointing at the committed sources:

```sh
git tag fps-kit-v0.1.0
git push origin fps-kit-v0.1.0
```

Both routes validate the package and require the tag to match its manifest
version. The release contains only that package's archive, checksum and JSON
metadata. A tag that already points at another commit is rejected, and an
existing release is never overwritten. Increment the package version to make
another release. Other packages and the CLI keep their own release schedules.

These tags release Karty archives; they do not publish the code packages as
independent Go modules or change their existing import paths.

## Package catalog

`catalog.json` describes the known packages. `cmd/karty-catalog` combines
that metadata with published stable releases to produce `master.json`.
Unreleased packages remain discoverable with no installable versions; draft and
prerelease assets never become a package's latest stable version.

The **Build package catalog** workflow saves the generated JSON as an artifact.
Every successful package publication runs it explicitly, so catalog refreshes
also work for releases created with GitHub's workflow token.

To publish the index at
`https://karty-game.github.io/karty-packages/master.json`:

1. In repository **Settings → Pages**, choose **GitHub Actions** as the source.
2. Run **Build package catalog** with **deploy** checked.
3. Set the repository Actions variable `PACKAGE_CATALOG_PAGES` to `true` to
   deploy automatically after future package releases or catalog metadata edits.

The index is a discovery API that a catalog UI can consume. Projects pin the
chosen archive URL and SHA-256 in `karty.toml`; restoring an existing project
does not require the catalog to be online. The CLI and SDK can adopt a package
release after it is published, without embedding its source or artwork.
