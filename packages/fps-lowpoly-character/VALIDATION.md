# Package validation — 2026-10-29 JST

This candidate promotes the existing four-character assets and animation selector
into an independent public-repository package. Engine rendering support still
needs to be added to the released SDK.

## Environment and commands

From the `karty-packages` repository root, all commands below used:

```sh
export PATH=/home/kef/.local/share/mise/installs/go/1.27.3/bin:/home/kef/.local/share/mise/installs/node/26.12.0/bin:/home/kef/.local/bin:/usr/bin:/bin
export GOCACHE=/tmp/karty-engine-go-cache
export GOMODCACHE=/tmp/karty-character-module-cache
export GOLANGCI_LINT_CACHE=/tmp/karty-engine-lint-cache
```

The existing cache was writable before Go workflows. Explicit PATH order avoids
the moved workspace's Nix Go 1.27.0 shadowing the pinned Go 1.27.3. Mise selects
the repository's `GOWORK=off`, `GOMAXPROCS=2` and `GOEXPERIMENT=simd`.

Passed:

```sh
mise exec -- go -C packages/fps-lowpoly-character test -count=1 -timeout=9s ./...
mise run test
mise run build
mise exec -- go -C packages/fps-lowpoly-character build ./...
mise run lint
```

The full test task covers root archive/catalog/site tooling and every independent
package module. Lint includes formatting, TOML/YAML/Markdown/JSON checks and
native `go vet` across the modules. No test deadlines or assertions were relaxed.

New asset checks verify complete sizes/SHA-256, self-contained GLB buffers/images,
model-only geometry, triangle/joint counts, animation target-name availability
on all four models, all 20 clips and their declared time bounds. These are
structural checks, not decoded Meshopt geometry or a new skinning comparison.
The latest scientist GLB is pinned to its corrected hash. Controller coverage
preserves the prior movement/stance/action/death tests and adds independent NPC
state, immutable metadata copies, documentation execution and transactional
rejection of invalid action/death values. Warmed native Go steps allocate zero.

## Initial failures and corrections

- Mise initially needed access to its trust/tool state. Required access prepared
  the pinned tools. The first full task was blocked downloading public modules
  by sandbox DNS/socket denial; the authorized rerun downloaded them normally.
- The catalog test still expected seven packages. Its explicit name expectation
  now includes this eighth package, preserving its unreleased-version checks.
  The subsequent full test task passed.
- The original asset metadata reported 2,182 assault triangles. Inspecting the
  actual compressed GLB found 2,181. The catalog/manifest now distinguish delivered
  geometry from that authoring-source count. Asset bytes were not changed.

## Artifact and integration limits

The `.kpkg` is exported locally with `mise run package -- fps-lowpoly-character`;
its exact size, SHA-256 and source inventory are emitted beside the archive in
`dist/releases/`. Those files are ignored local output. The archive includes
portable GLBs, packed editable sources, credits, tests and the controller code.

The source bank, scientist correction and original policeman are preserved.
No new shader, source-animation edit, retargeting, root-motion change or graphics
execution was performed. The current workspace exposes no `/dev/dri`; no
software-renderer substitute was used.

Native tests/builds do not establish actual WASM gameplay, hardware rendering,
physical-device FPS, fixed strafe contact or resolved browser startup stalls.
The stock SDK still needs a supported character resource/playback API for a
normal playable sample. The inherited art/motion limits and asset-redistribution
licensing discrepancy are documented in README and CREDITS. No commit, release
publication or deployment was performed.

## README and catalog preview — 2026-10-29 JST

Shortened the README and moved asset/runtime details and limitations to
INTEGRATION.md. Added the existing all-four cast screenshot as `previews/cast.png`
and selected it in the root catalog. Its original hardware-renderer identity and
image hash are recorded in `previews/README.md`; no new graphics execution was
performed. The screenshot was opened and inspected before reuse.

With the environment above and `GOPROXY=off`, these passed from the repository root:

```sh
mise exec -- go test -count=1 -timeout=9s ./internal/catalog ./website
mise exec -- go run ./cmd/karty-catalog --repository karty-game/karty-packages --releases tmp/fps-lowpoly-readme-review/releases.json --output tmp/fps-lowpoly-readme-review/catalog
mise exec -- go run ./cmd/karty-site --master tmp/fps-lowpoly-readme-review/catalog/master.json --output tmp/fps-lowpoly-readme-review/site
```

The offline release fixture contains `[]`. Generated master.json, catalog-card
HTML, package-detail HTML and package Markdown all contain the expected raw
GitHub preview URL. Formatting checks passed. These are site-generation checks,
not a new browser rendering review or public website deployment.

The updated local candidate is exported into `dist/readme-preview/` to preserve
the earlier immutable archive in `dist/releases/`.

## Credits consolidation — 2026-10-30 JST

Removed the redundant `licenses/` directory at the user's request. CREDITS.md
now retains the source archive hashes, attribution and CC0 license references.
Updated the asset inventory and its directory coverage.

With the environment above and `GOPROXY=off`, the focused package tests passed:

```sh
mise exec -- go -C packages/fps-lowpoly-character test -count=1 -timeout=9s ./...
```

Formatting passed for the credits, integration guide and asset inventory.

## Standard asset layout — 2026-10-30 JST

Removed the unsupported binary cast payload and its directory. The package now
contains GLB models/animations, editable sources and the Go controller. Updated
the inventory, test directory coverage, README, integration guide, credits and
catalog description. The documentation states that released-SDK rendering
support is still missing without advertising an alternate feature tier.

With the environment above and `GOPROXY=off`, the focused package tests passed:

```sh
mise exec -- go -C packages/fps-lowpoly-character test -count=1 -timeout=9s ./...
```

Formatting passed for the affected Markdown/JSON. No new graphics execution or
rendering support is claimed.
