# FPS Kit validation — 2026-10-22

Version 0.1.0 contains 49 opaque materials and 36 transparent decals, all
256×256. Canonical source is `packages/fps-kit`; tracked generators live in
`scripts/fps-kit`. SDK starter inputs are exported copies, verified byte for
byte by `export-fps-kit-starter --check`.

## Checks run

From the CLI repository root, using writable warmed
`GOCACHE=/tmp/karty-engine-go-cache` and `GOMAXPROCS=2`:

- `KARTY_HOME=/tmp/karty-fps-kit-complete-home mise run test`: passed, 51.61s.
- `mise run build`: passed, including the sprite-package resolver change.
- `mise run check-fps-kit`: passed deterministic generation, archive-only
  texture/geometry checks, all 36 transparent decals, unused-export pruning,
  malformed references and the complete decal gallery.
- `mise exec -- golangci-lint run ./internal/packages ./internal/scaffold`:
  passed with `GOLANGCI_LINT_CACHE=/tmp/karty-engine-lint-cache`.
- `mise run export-fps-kit-starter --
/home/kef/projects/karty-engine/core/templates/fps/packages/fps-kit --check`:
  passed.
- Prettier, YAML lint, Taplo syntax checks and scoped `git diff --check`: passed.

Engine root, with the same Go cache and worker setting:

- `mise run check-sdk-inputs`, `mise run test`, `mise run build` and
  `mise run bundle-sdk`: passed. The full tests completed in 52.03s. Native and
  browser host compilation used the prepared renderer workspace.
- `mise exec -- golangci-lint run --fix ./core/internal/sdk`: passed.

The local SDK 0.0.18 candidate selects template revision 0.0.9. Its archive
SHA-256 is `ef3752df0be19d7f1bf976ba87eb575ee67df3c9c3414f4613833dc4327480f6`.
This is local candidate evidence, not a published SDK/tag/checksum check.

## Actual execution and visual review

The common browser host SHA-256 is
`ec0f6a06f4bc22530897e842c50f7e3a0f984018bcba877fe48b3716d715932b`.
Cartridges ran in Chromium WebGL2 with SwiftShader and synthesized input.
Loopback fixture servers served immutable copies and checked artifact hashes.
Material builds used the pinned native Materialize tool under Xvfb with
`LIBGL_ALWAYS_SOFTWARE=1`; Go compilation alone is separate evidence.

- **Crates:** the archive-only gallery built and executed with all five sizes.
  Each moving crate fell and settled within 0.035m of its expected half-height.
  Reviewed front/side/top views show aligned geometric frames on static and
  dynamic crates. The seven solids share wood textures; each moving crate
  retains one box collider. An earlier painted frame wrapped incorrectly; that
  artwork was replaced by seamless wood and actual frame geometry.
- **FPS starter:** an untouched generated project built with its default TinyGo
  compiler. Its two crates settled and ordinary D/W input walked through the
  2m doorway, ending at approximately `(4.16, 10.72, 1.72)`. The final marker
  build passed with game SHA-256
  `0d1a2f044c7278123065b838301738ca3da38f79896e217a7152361c72238c85`.
  Reviewed room numbers, floor arrow, goal star, brick/concrete materials and
  crate frames. No browser errors.
- **Physics lab:** `mise run build-crate-room-web` and
  `KARTY_CRATE_ROOM_UI_OUTPUT=dist/validation/fps-kit/aligned-crate-room
PLAYWRIGHT_BROWSERS_PATH=/root/.cache/ms-playwright mise run check-crate-room-ui`
  passed. The existing desktop, landscape and portrait HiDPI checks retain
  held-fire release, movement/look/shoot, blast and reset assertions. Reviewed
  tumbling crates retain aligned frames. Game SHA-256:
  `7a7a8b90c64591e9d855354dbe1069ec751531b84b5a397d186d1b7a30166bac`.
- **Decals:** the complete wall/floor gallery built with the pinned Go 1.27.3
  compiler and executed in the browser. Reviewed all 36 symbols and a floor
  target; transparency, one-sided orientation and surface offsets display
  correctly. Game SHA-256:
  `43a475da80cfee101c9c25121dd99a3ff190ef56a0a4f915f8d0380ea514e56d`.

Browser reports and captures are under the CLI's ignored
`dist/validation/fps-kit/`, including `complete-starter-browser-report.json`,
`decals-browser-report.json`, `aligned-crate-room/report.json` and
`alignment-validation.txt`. One-off browser helpers live in `/tmp`; maintained
checks do not depend on those files.

## Initial failures and limits

The first CLI test run had no SDK 0.0.18 in its default home. Selecting an
isolated locally checksummed candidate resolved that. Sandboxed HTTP/SSE
fixtures then failed to bind loopback; the authorized run passed. One later
full CLI run exhausted the unchanged nine-second lightmap-packing deadline
under concurrent load; the inspected stack was in `packCompactRects`, and the
same task passed when idle. No deadlines were extended.

The first browser observer watched only planar body event 19; spatial bodies
also use authoritative event 11. Correcting the observer established actual
falling/settled poses. The initial full-palette room's heavily subdivided
boundary showed triangular rendering artifacts; a rectangular room with small
wall swatches renders cleanly. No renderer workaround was shipped.

This establishes compilation, actual WASM execution, software browser graphics
and screenshot review. It does not establish physical-device FPS, physical
input/audio behavior or public release availability. Material atlases/maps and
sprite pages have separate decoded costs; PNG source size is not GPU memory.

## README captures

Three clean 1280×720 PNGs were captured with the native screenshot command,
Xvfb and Mesa software OpenGL. They render the authored levels without game HUD
or browser controls; runtime-spawned dynamic bodies are not part of these stills.
All three were visually reviewed for framing, surface detail, decal transparency
and aligned crate frames. No image synthesis or image retouching was used.

Common command prefix:

```sh
KARTY_HOME=/tmp/karty-fps-kit-complete-home LIBGL_ALWAYS_SOFTWARE=1 \
  xvfb-run -a /home/kef/projects/karty-cli/dist/karty render screenshot \
  --width 1280 --height 720 \
  --host /home/kef/projects/karty-engine/dist/host/linux-amd64/karty-host
```

Each command adds the following camera/output flags and `levels.room`, from
the corresponding project under `dist/validation/fps-kit/`:

| Image                       | Project        | Position    | Yaw | Pitch | FOV |
| --------------------------- | -------------- | ----------- | --- | ----- | --- |
| `screenshots/materials.png` | `game`         | `2,2,2.1`   | 28  | -6    | 65  |
| `screenshots/starter.png`   | `fps-corridor` | `2,1.5,1.9` | 23  | -6    | 65  |
| `screenshots/decals.png`    | `fps-decals`   | `7,6,2.4`   | 0   | 0     | 48  |

For example, the materials flags are `--position 2,2,2.1 --yaw 28 --pitch -6
--fov 65 --output /home/kef/projects/karty-cli/packages/fps-kit/screenshots/materials.png
levels.room`. All three captures passed. The gallery and decal worlds match
the tracked `examples/room` and `examples/decals` files byte for byte. The starter
uses two rooms joined by a four-metre corridor, matching the revised starter
world. Native host SHA-256:
`46173cd74819616fa5ecfad36d9312e74d38e8c93045a4c4032cdc6b9beb0447`.

Documentation follow-up checks passed: Prettier, local Markdown link/image and
PNG-dimension checks, `mise run check-fps-kit`, SDK starter export byte comparison,
`mise run check-sdk-inputs` and scoped `git diff --check`. The package archive was
refreshed locally; no SDK artifact was republished. These captures establish
native software graphics and visual review, not device performance.

## Corridor layout follow-up

The FPS starter now separates its two 8×8m rooms by a 2×4m corridor. The passage
has two ordinary portal connections, a 2.6m ceiling below the rooms' 3m ceilings,
concrete walls and its own light. The workshop's props, light, room number,
dynamic crate and goal moved four metres north with the room. Source revision
0.0.9 remains folded into the unreleased SDK 0.0.18 candidate.

An independently scaffolded `fps-corridor-final` project matches the tracked
world byte for byte. It built with the default TinyGo compiler in 14.765s:

```sh
KARTY_HOME=/tmp/karty-fps-kit-corridor-home \
KARTY_GO=/root/.local/share/mise/installs/go/1.27.3/bin/go \
GOCACHE=/tmp/karty-engine-go-cache GOMAXPROCS=2 \
XDG_CACHE_HOME=/tmp/karty-tinygo-xdg-cache LIBGL_ALWAYS_SOFTWARE=1 \
xvfb-run -a /home/kef/projects/karty-cli/dist/karty build --target web \
  --host /home/kef/projects/karty-engine/dist/host/web/karty-host.wasm
```

`PLAYWRIGHT_BROWSERS_PATH=/root/.cache/ms-playwright node
/tmp/check-fps-kit-corridor.mjs` passed actual WASM/WebGL2 execution in Chromium
SwiftShader, using an immutable loopback distribution. The two dynamic crates
settled; keyboard movement reached `(4.04, 9.52, 1.72)` inside the passage,
the east wall stopped strafing at X=4.68, and the player crossed the second
portal to `(4.04, 12.96, 1.72)` in the workshop. No browser errors occurred.
Report/captures are in `dist/validation/fps-kit/corridor-browser/`.
Cartridge SHA-256:
`5bf017795bb1a85cac0e5c77dea81edc724436f7d0d00fe39100447b3a01fec2`.
Level SHA-256:
`6f57462f95f8ca48ce154dc0492b8e31d400489530d7dff8046a48848b18b13f`.
The browser host hash remains the one recorded above.

The starter screenshot was recaptured at the original camera angle, now showing
the corridor depth. Both palette sheets are again displayed inline beneath the
README's gallery links. A first starter export overlapped screenshot capture
and included a temporary `.capture` image; that generated temporary copy was
removed, and the exporter now selects the three named documentation screenshots.
The byte comparison passed after capture finished. Export mismatch messages
also avoid dumping base64 pixel data.

Engine `GOCACHE=/tmp/karty-engine-go-cache GOMAXPROCS=2 mise run test` passed in
51.77s; `mise run build` passed for native, check and browser hosts. The first
CLI test run caught an older 1280×720 physics-lab thumbnail; it was recaptured
from actual browser execution at the required 960×540 and visually reviewed.
The initial focused scaffold lint caught naming, spelling and spacing issues;
those were corrected, and `golangci-lint run ./internal/scaffold` passed with
zero issues. No test deadline or assertion was relaxed.

With the corrected thumbnail, the unchanged full CLI command
`KARTY_HOME=/tmp/karty-fps-kit-corridor-home GOCACHE=/tmp/karty-engine-go-cache
GOMAXPROCS=2 mise run test` passed in 48.44s; `mise run build` passed.
The scaffold regression now verifies separated room footprints, corridor
dimensions and both reversed coincident portal edges before compiling the world.

## Shared metal bumpers and authored slopes

The kit now exports `bumper-low`, `bumper-cover` and `bumper-tall`: static,
collision-enabled octagons 1.7m across and 0.75m, 1.25m and 2m high. Each uses
one solid, a floor-level origin and two new 256px materials, `bumper-red-metal`
and `bumper-metal-cap`. Both select the `metal` Materialize preset. The current
package contains 49 environment textures and 36 transparent decals.

Crate Room uses all three shared prefabs at its original bumper positions and
retains its authored sloped banks. The ramp prefab and gallery instance were
removed; slopes belong to the consuming level's sectors/geometry. The gallery
step side now shows `floor-dark-checker`, retaining all 49 materials without a
ramp. The gallery has 94 static solids. A first focused test found the missing
swatch after ramp removal; correcting the authored step material resolved it.

Optional `[textures.materialize]` settings now survive complete package
validation, archive import, selected texture resolution and level staging.
Tests cover unknown/invalid settings, environment-only profiles, unused-export
validation, archive portability and omitted versus explicit-zero overrides.
Existing bounds, confinement and pruning remain enforced. The versioned CLI
decision is `website/content/design/package-materials-v1.md`; no engine API or
wire change was introduced.

From the CLI root:

- `KARTY_HOME=/tmp/karty-fps-kit-bumpers-home GOCACHE=/tmp/karty-engine-go-cache
GOMAXPROCS=2 mise run test`: passed in 48.77s; `mise run build`: passed.
- `mise run check-fps-kit`: passed deterministic generation, archive-only
  materials, three bumper collision heights, removed ramp and decal coverage.
- `GOCACHE=/tmp/karty-engine-go-cache GOLANGCI_LINT_CACHE=/tmp/karty-engine-lint-cache
GOMAXPROCS=2 mise exec -- golangci-lint run ./website/internal/generator
./internal/packages ./internal/levelbuild ./internal/scaffold`: passed, zero issues.
- Prettier, repository-configured YAML lint, Taplo syntax, local README links
  and scoped `git diff --check`: passed.

The initial CLI run found the new design page missing from the website's
explicit page list. Registering it fixed the link, and the unchanged full task
passed. Ordinary naming/spacing lint findings were fixed. No assertions or
deadlines were relaxed.

With the local candidate, pinned Go 1.27.3 and Materialize under Xvfb,
`mise run build-crate-room-web` passed in 7.398s. Actual Chromium SwiftShader
execution passed:

```sh
KARTY_CRATE_ROOM_UI_OUTPUT=dist/validation/fps-kit/metal-bumpers \
PLAYWRIGHT_BROWSERS_PATH=/root/.cache/ms-playwright \
mise run check-crate-room-ui
```

Desktop, landscape and portrait HiDPI checks retained held-fire release,
simultaneous move/look/shoot, BLAST/RESET UI and keyboard assertions. No browser
errors occurred. The 960×540 thumbnail was recaptured from this build. The hero
was recaptured with its documented camera from
`dist/validation/fps-kit/bumper-gallery`, showing three bumper heights and no
ramp. Both images were visually reviewed.

Public SDK decoders validated the compiled level, all three collision heights
and reflectance companion/mips. Reconstructing real asset-cache recipe hashes
proved that both new materials used
`--only height,normal,ao,metallic,smoothness --normal-format opengl --format png
--preset metal`, without numeric overrides; default-preset hashes differed.
Painted red sides generated metallic 0 and smoothness 111–128; neutral caps
generated metallic 10–22 and smoothness 129–136. This records generated data
instead of inferring reflectance from albedo or screenshots.

Cartridge SHA-256:
`ec01144a8616b19f6736478f120ed78457c60c07878c503d59305f3b9bac5079`.
Level SHA-256:
`6ab6469a6cc279fe3214071a6172526e771170311bf410e3c11a01ba98b47af1`.
Reports/captures: `dist/validation/fps-kit/metal-bumpers/`; decoded material/cache
evidence: `dist/validation/fps-kit/bumper-materials-evidence.txt`. These establish
software graphics and synthesized input, not physical-device FPS.

Engine `GOCACHE=/tmp/karty-engine-go-cache GOMAXPROCS=2 mise run test` passed in
74.09s; `mise run build` passed for native, check and browser hosts. The updated
SDK starter includes all 85 texture exports and the revised prefab library.

## White-surface reflectance and gallery spawns

The live gallery exposed two authoring issues. Albedo-based inference classified
bright white masonry/paint as metal: white concrete's mean metallic channel was
233/255 versus blue concrete's 32/255, despite identical concrete normals.
White brick also inferred stronger normals from its greater paint/mortar
contrast. The generator now chooses explicit brick, concrete, wood, floor or
plaster presets and sets `metallic_scale = 0` for all 47 nonmetal materials.
White brick uses `normal_strength = 0.6`; the two bumper materials retain the
metal preset. These remain calibrated albedo-derived maps, not shared neutral
geometry maps across colors.

Public SDK decoding of the corrected live level
`61e16f73915bb22d4c7a60b199635888756e03d840590a80277addc65ad8d6c0`
verified zero metallic pixels in all 47 nonmetal tiles. White-brick XY normal
RMS is 0.240 versus blue brick's 0.225; white-concrete mean smoothness fell from
212/255 to 19/255. Before/after evidence is under
`dist/validation/fps-kit/white-material-{before,after}-evidence.txt`.

The live project had retained the two-room starter's dynamic crate coordinates:
one intersected a static gallery crate, and another was centered on the gallery's
Y=14 wall beside the step. It now calls the canonical five-size gallery helper.
The helper's tall crate moved from `(12,7)` to `(12,5)` to clear the new bumper.
README/USAGE now explain replacing starter spawning when switching galleries.
A regression reads the actual helper coordinates and checks clearance from
walls, props, the player and other crates.

Actual Chromium SwiftShader execution at the running dev URL passed with no
errors. All five crates fell and settled within 0.001m of their expected
half-heights: 0.25m, 0.5m, 1m, 0.5m and 1m. Report/captures:
`dist/validation/fps-kit/gallery-fixed/`. Native Xvfb/Mesa captures of the gallery
and north-wall white swatches were visually reviewed; the README hero was
refreshed. They used the same documented gallery camera and a close view at
`(3.8,11.8,2.15)`, yaw 30°, pitch 2°, FOV 65°.

`mise run check-fps-kit`, scoped package lint, Prettier and scoped diff checks
passed. A first full CLI run while two software-rendered browser sessions were
active exhausted unchanged nine-second timers in sample compilation, archive
compression, lightmap packing and schema fixtures. Inspected stacks showed
active work. After closing those validation browsers, the identical command
`KARTY_HOME=/tmp/karty-fps-kit-bumpers-home GOCACHE=/tmp/karty-engine-go-cache
GOMAXPROCS=2 mise run test` passed idle in 47.54s, and `mise run build` passed.
No deadlines or assertions changed. The user's dev server remains running on
loopback port 39231. This is software-graphics and synthesized-input evidence,
not physical-device performance.

## Generic per-texture geometry normalization

This supersedes the white-brick `normal_strength = 0.6` exception above, which
has been removed. Geometry generation now derives Rec.709 byte luminance from
each texture's own covered pixels. Symmetric 1% tail trimming defines the robust
range; normalization centers it at 128, targets a span of 64 and caps gain at 3.
Spans up to 8 receive no amplification; gain ramps smoothly across spans 8–16.
Transparent pixels do not affect statistics, alpha is preserved and no spatial
blur removes local detail.

Only normal, height and AO generation uses this input. Authored albedo remains
unchanged; metallic/smoothness generation retains the original RGB input and the
existing explicit nonmetal classifications. No shared source image, new API,
material-family knowledge or engine rendering change is involved. Rare tail
values can clip; painted patterns and shadows remain ambiguous height evidence.

The initial 49-material gallery build with native Materialize completed in
42.7s. Public SDK decoding measured natural brick's normal XY RMS increasing
from 0.137909 to 0.270944 and its height range from 23–84 to 69–197. The smooth
gain transition does not alter brick inputs, whose robust spans are at least 30.
Decoded evidence and native captures are retained under
`dist/validation/fps-kit/`: `normalized-material-evidence.txt`,
`brick-normalized-close.png` and `brick-normalized-comparison.png`.

From the CLI root:

- `KARTY_HOME=/tmp/karty-fps-kit-bumpers-home GOCACHE=/tmp/karty-engine-go-cache
GOMAXPROCS=2 mise run test`: passed in 51.60s; `mise run build`: passed.
- Focused asset-pipeline tests passed, including brightness/tint normalization,
  detail, bounded gain, coverage, cancellation, material caching and reflectance.
- Scoped `golangci-lint run ./internal/assetpipeline ./internal/levelbuild
./internal/packages`: passed with zero issues.

The refreshed dev gallery passed actual Chromium WebGL2/SwiftShader execution
with no errors. All five dynamic crates fell and settled within 0.001m of their
expected half-heights. The immutable served level has SHA-256
`03e19f4137b08d842d747b265da09ed121dfaaffe28379812abbcaa60d8f6c62`;
the browser host retains the checksum above. Public SDK decoding verified zero
metallic pixels in all 47 nonmetal exports. Evidence is in
`gallery-normalized/browser-report.json` and
`normalized-live-material-evidence.txt` under the same ignored evidence folder.
The README hero and `normalized-live-close.png` were captured from that exact
level with native Xvfb/Mesa; the preview stays on loopback port 39231.

`mise run check-fps-kit` also passed after regeneration. These results cover CLI
generation, actual WASM execution, compiled material data and software graphics.
They do not establish physical-device performance.
