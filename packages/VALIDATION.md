# Controller split and dialogue preview validation

Validated on 2026-10-22 from the `karty-packages` repository root, using pinned
mise tools, `GOCACHE=/tmp/karty-engine-go-cache`, `GOMAXPROCS=2` and the repository's
`GOWORK=off` configuration. Lint also used
`GOLANGCI_LINT_CACHE=/tmp/karty-engine-lint-cache`.

## Checks

- `GOCACHE=/tmp/karty-engine-go-cache mise run test` passed all root tooling and
  independent package modules, including all three split controllers.
- `GOCACHE=/tmp/karty-engine-go-cache GOLANGCI_LINT_CACHE=/tmp/karty-engine-lint-cache mise run check`
  passed formatting, lint, tests, generated FPS Kit checks, every package manifest
  and tooling builds. This full gate preceded the additive dialogue preview.
- `GOCACHE=/tmp/karty-engine-go-cache mise exec -- sh -eu -c 'for name in controllers-3d controllers-isometric controllers-topdown; do go -C "packages/$name" build ./...; done'`
  passed standalone compilation of each new module.
- `GOCACHE=/tmp/karty-engine-go-cache mise run package -- controllers-3d`,
  `GOCACHE=/tmp/karty-engine-go-cache mise run package -- controllers-isometric`
  and `GOCACHE=/tmp/karty-engine-go-cache mise run package -- controllers-topdown`
  exported independent `0.0.1` archives and SHA-256 sidecars in `dist/releases/`.
- `GOCACHE=/tmp/karty-engine-go-cache KARTY_SITE_BROWSER=/root/.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell mise run check-site-web`
  passed with loopback access, including controller and dialogue images on cards
  and detail pages, search/filtering, version pages, Markdown, clipboard fallback
  and mobile overflow. The fixture maps public preview URLs to checked-in assets
  so it remains offline. Production links retain HTTPS URLs.
- After adding the dialogue preview,
  `GOCACHE=/tmp/karty-engine-go-cache mise exec -- go test -count=1 -timeout=9s ./internal/catalog ./website`
  passed. Focused Prettier checks and `golangci-lint fmt --diff` passed for the
  final metadata, documentation and test changes.
- From `dist/releases/`,
  `sha256sum --check controllers-3d-0.0.1.kpkg.sha256 controllers-isometric-0.0.1.kpkg.sha256 controllers-topdown-0.0.1.kpkg.sha256`
  verified all three archive checksums.

## Failures investigated

The first focused package run exposed an isometric camera test migration error:
splitting a combined test left its dead zone at the default instead of zero.
Restoring the original test setup preserved its assertions; the subsequent
package run and full gate passed.

The first browser invocation failed because the sandbox denied its loopback
listener (`listen tcp6 [::1]:0: socket: operation not permitted`). The check was
rerun with the required access. That exposed a fixture synchronization issue:
`Image.decode()` could outlive Chromium's virtual-time DOM capture. The fixture
now loads preview images eagerly before the frame load event and asserts their
completion and dimensions. Existing production lazy loading, assertions and
deadlines remain intact. The controller-only and final dialogue-inclusive
browser runs passed.

## Evidence limits

Controller behavior and allocation evidence comes from native Go tests; these
checks did not execute the new modules in WASM or run sample gameplay. Browser
evidence covers the package website, not engine graphics or device FPS.

The controller JPEGs are byte-identical copies of existing Crate Room, Vigil and
Willowbrook sample previews. They were visually inspected; they are not fresh
runtime captures. The dialogue PNG is a generated illustration of Yarn script
authoring, with its prompt saved beside it. Its text and layout were inspected.
No .NET compiler restore, package publication, commit or deployment was performed.
