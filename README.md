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

For FPS Kit, edit the version in `packages/fps-kit/karty-package.toml` directly.
The asset generator preserves that value; no generator version needs updating.

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

Packages can live in any public GitHub repository. Add an entry to `catalog.json`
with a repository override; its sources do not need to be copied here:

```json
{
  "name": "community-props",
  "repository": "someone/karty-props",
  "description": "Extra prototype props",
  "tags": ["3d", "prototyping"],
  "source": ".",
  "ref": "main",
  "readme": "README.md",
  "preview": "screenshots/props.png"
}
```

`repository` defaults to this catalog's repository. An explicit repository defaults
to sources at its root; otherwise sources default to `packages/<name>`. `source`
can select a subdirectory, `ref` defaults to `main`, and `readme` defaults to
`README.md` inside the source directory. README and preview paths are relative to
the repository root. Package names must be unique across the catalog.

External repositories use the same release convention: tag
`community-props-v1.2.3` with an asset `community-props-1.2.3.kpkg` and either
GitHub's SHA-256 asset digest or a matching `.kpkg.sha256` sidecar. The builder
checks each archive and release URL against that entry's repository. Generated
entries include their repository, archive URLs and SHA-256 checksums.

Run `mise run catalog` to discover releases from every listed repository. Set
`GH_TOKEN` for authenticated GitHub API requests. Offline builds can pass
`--releases releases.json` to `cmd/karty-catalog`, with an object mapping each
`owner/repository` to its GitHub releases array. A plain array still works for
catalogs containing only the default repository.

## Package website

The website is generated from `master.json`: searchable package cards, tag filters,
and a page for each package with its description, install command, available
versions and source links. Each page is also available as Markdown. Packages from
external repositories appear alongside packages maintained here.

Generate the catalog and preview the website locally:

```sh
mise run catalog
mise run preview-site
```

Open `http://127.0.0.1:8082`. To generate files without starting a server, run
`mise run site`; output is written to `dist/catalog/` alongside `master.json`.
The generator works offline with an existing catalog:

```sh
go run ./cmd/karty-site --master path/to/master.json --output dist/site
```

`mise run check` covers generation and input validation. To check search, tag
filters, version pages and mobile layout in a browser, set `KARTY_SITE_BROWSER`
to an installed Chromium executable and run `mise run check-site-web`. Browser
checks are written in Go and need no additional package manager.

The **Build package catalog** workflow generates the JSON, HTML and Markdown site
and saves it as an artifact. Every successful package publication runs it
explicitly, including releases created with GitHub's workflow token. A scheduled
refresh every six hours discovers releases from external repositories; run the
workflow manually for an immediate refresh.

To publish the website at `https://karty-game.github.io/karty-packages/`:

1. In repository **Settings → Pages**, choose **GitHub Actions** as the source.
2. Run **Build package catalog** with **deploy** checked.
3. Set the repository Actions variable `PACKAGE_CATALOG_PAGES` to `true` to
   deploy automatically after future package releases, metadata edits and scheduled
   refreshes.

The discovery API remains available at
`https://karty-game.github.io/karty-packages/master.json`. Projects pin the chosen
archive URL and SHA-256 in `karty.toml`; restoring an existing project does not
require the website or catalog to be online. The CLI and SDK can adopt a package
release after it is published, without embedding its source or artwork.
