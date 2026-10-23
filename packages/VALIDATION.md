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

## Controllers 3D 0.0.2 and real TinyGo execution

On 2026-10-22, the package-owned allocation gate passed all 14 cases using the
frozen SDK 0.0.18 candidate and controllers-3d 0.0.2 candidate archives. These are
local candidate results; they do not establish public release availability.

```sh
XDG_CACHE_HOME=/tmp/karty-tinygo-xdg-cache \
GOCACHE=/tmp/karty-engine-go-cache GOMAXPROCS=2 \
  mise run check-controller -- \
  --sdk-archive /tmp/karty-controller-final-home/sdks/0.0.18/sdk.zip \
  --sdk-sha256 139f031b7c642f61a34dea4fae0270a00ca81992e55c48560058a8ef0596ceda \
  --package-archive /tmp/karty-controller-extraction/controllers-3d-0.0.2.kpkg \
  --package-sha256 f6f1869743270910ec365ab41a9e7ec8741ac7808c71081c7df7dff403a40946 \
  --go /root/.local/share/mise/installs/go/1.27.3/bin/go \
  --tinygo /tmp/karty-controller-final-home/tools/tinygo/0.43.0/linux-amd64/tinygo/bin/tinygo \
  --output /tmp/karty-controller-extraction/controller-allocations.json
```

Actual TinyGo 0.43.0 WASM ran in public Wazero v1.12.0. Every case measured 1,000
updates after 101 warm-up updates, with zero guest allocations and zero allocated
bytes. Both Hooks and manual clients exercised free and physics movement with
keyboard, captured mouse and dual touch; accepted-physics-pose runs covered both
client styles. The gate retained the original command counts, movement speed,
finite and bounded look, touch entity reuse, simultaneous walking/look, accepted
pose ordering and focus-reset assertions. Each case also confirmed that deliberate
heap growth changes both allocation counters. This establishes actual WASM
execution; it does not establish browser graphics or physical device performance.

Focused archive-authentication, unsafe-path, case-collision, complete command
parsing, archived-source staging and stale-success-report tests passed:

```sh
GOCACHE=/tmp/karty-engine-go-cache GOMAXPROCS=2 \
  mise exec -- go test -count=1 -timeout=9s \
  ./internal/controllercheck ./cmd/check-controller
```

The initial compiler invocation could not write the sandbox's default TinyGo cache.
The successful command selected an existing writable cache with `XDG_CACHE_HOME`.
An isolated temporary fixture also initially tried to inspect VCS state; the runner
now adds `GOFLAGS=-buildvcs=false` only to its compiler subprocess environment.
The repository's normal builds retain VCS stamping. The first unit invocation
reported a missing transitive checksum; `go mod tidy` pinned Wazero's public
`golang.org/x/sys v0.44.0` dependency before the passing run. Deadlines were not
extended. A failed run removes an existing success report before it starts.

The complete repository check passed after the final borrowed-event adapter was
frozen:

```sh
GOCACHE=/tmp/karty-engine-go-cache GOMAXPROCS=2 mise run check
GOCACHE=/tmp/karty-engine-go-cache GOMAXPROCS=2 mise run package -- controllers-3d
printf '%s  %s\n' \
  f6f1869743270910ec365ab41a9e7ec8741ac7808c71081c7df7dff403a40946 \
  dist/releases/controllers-3d-0.0.2.kpkg | sha256sum --check -
```

HK format/lint, root tooling tests (including the allocation checker), all six
code-package module suites, deterministic FPS generation, seven package source
checks and all four production tool builds passed. The full check log is
`/tmp/karty-packages-final-check.log`. These checks add native/tooling
evidence; the actual TinyGo/WASM allocation evidence is recorded above.

The normal release task exported a 146,032-byte controller archive with the
exact frozen SHA-256 shown above. The obsolete local candidate and its sidecars
were preserved in `/tmp/karty-controller-old-standard-archive/` before
exporting into `dist/releases/`; published archives were not touched.
No commit, tag or publication was performed.
