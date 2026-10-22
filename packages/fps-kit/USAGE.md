# FPS Kit usage and reference

[← Package overview and screenshots](README.md)

## Local candidate setup

The FPS starter declares a checksum-pinned archive dependency. Its planned
release URL is
`https://github.com/karty-game/karty-fps-kit/releases/download/v0.1.0/fps-kit-0.1.0.kpkg`.
Creation of the [package repository](https://github.com/karty-game/karty-fps-kit),
publication of this archive and SDK 0.0.18 are pending. A fresh first restore needs
the archive to be available; a matching verified cache supports subsequent
restores. `karty package restore`, `karty build` and `karty dev` restore declared
packages. Restored docs and examples live under `.karty/packages/fps-kit/`.

To test the current checkout before publication, install the local SDK candidate
and matching host, create the FPS project, then replace its release dependency
with a local exported archive:

```sh
# CLI repository root:
karty package export --output /tmp/fps-kit-0.1.0.kpkg packages/fps-kit

# Generated FPS game directory:
karty package import --replace /tmp/fps-kit-0.1.0.kpkg
karty dev
```

Import copies the local archive under `packages/` and records its checksum.
Commit that archive and `karty.toml` for a portable local dependency. Direct
`karty package import` accepts local folders and archives; URL dependencies are
configured in `karty.toml`.

## Example rooms

For a furnished starter room, copy `examples/room/level.toml` and `world.yaml`
from the restored `.karty/packages/fps-kit/` into your game's `levels/room/`
folder, replacing the starter room files. The
room keeps `levels.room` and the starter spawn `(2, 2, 1.7)`. Remove the starter's
`game.addCrates(event.Handle)` call when replacing its level: its crate positions
belong to the starter's two rooms and corridor. Run `karty dev`.
It includes every material, five static crate sizes, three bumper heights, a stack, low cover,
a 25 cm step and small wall swatches. Its runtime renderer declarations also support the optional
dynamic crate helper below.

The examples target the SDK 0.0.18 candidate and matching host currently used by
the FPS starter. That candidate must be installed locally until published.
The asset package adds no new engine API, wire format or SDK compatibility pin.

Import the package with `karty package import /path/to/fps-kit` or import its
exported `.kpkg` archive. Package texture references select their PNG declarations
automatically; do not copy them into your level manifest.

## Materials

All 49 PNGs are **256×256**, opaque, power-of-two and tileable. Scale marks
assume one texture repeat per metre. Grid ticks divide a metre into quarters;
the finer grid uses eighths. The deliberately simple palette stays readable at
low render resolutions and under level lighting.

| Material name                             | Variants                                                                 |
| ----------------------------------------- | ------------------------------------------------------------------------ |
| `wall-<color>-grid`                       | orange, red, blue, yellow, green, white, purple                          |
| `wall-<color>-panel`                      | The same seven colors; inset panel and zone number                       |
| `wall-<color>-checker`                    | The same seven colors; half-metre checks                                 |
| `wall-<color>-brick`                      | The same seven colors; painted running-bond masonry                      |
| `wall-<color>-concrete`                   | The same seven colors; rough painted formwork                            |
| `wall-brick`, `wall-concrete`             | Natural clay brick and neutral concrete                                  |
| `floor-<tone>-grid`                       | gray, dark                                                               |
| `floor-<tone>-checker`                    | gray, dark                                                               |
| `ceiling-<tone>-panel`                    | gray, dark                                                               |
| `crate-panel`, `crate-lid`, `crate-frame` | Seamless boards and grain; structural frames are geometry                |
| `bumper-red-metal`, `bumper-metal-cap`    | Red painted metal sides and neutral metal caps; automatic `metal` preset |
| `trim-hazard`                             | Seamless yellow/charcoal diagonal stripes                                |

Always prefix these names with `fps-kit.` in your game. White is an off-white
to retain detail in bright lighting. Palette order is orange, red, blue, yellow,
green, white, purple (grid/panel/checker), gray floor/ceiling, dark floor/ceiling,
crate panel/lid and hazard, then brick/concrete pairs in the same color order,
natural brick and neutral concrete, crate frame wood, and bumper side/cap metal.

Brick and concrete add restrained aggregate, pores, mortar and shallow joints
without baked directional lighting. A repeat covers 1 m; bricks are 25 × 12.5 cm
including mortar. Karty's normal world-material build uses its pinned native
Materialize tool to derive normal, height and ambient-occlusion maps from these
PNGs, then packages the maps and mip levels in the level's material atlas.
They participate in the ordinary level lighting, including runtime props.
The supplied room includes two lights to reveal that relief.

Package format 1 exports albedo/profile declarations with optional Materialize
settings for environment textures. Both bumper textures select `preset = "metal"`
automatically. The package supplies source PNGs; Karty generates their normal,
height, ambient-occlusion, metallic and smoothness data during the ordinary build.
It carries no separate custom normal-map files.

Brick, concrete, wood and painted surfaces use explicit matching presets with
`metallic_scale = 0`, so bright colors stay nonmetallic. Materialize infers each
surface's relief from its own albedo texture, with per-texture luminance
normalization before generating height, normal and AO. No color variants share
generated maps. Original RGB remains the input for metallic and smoothness.
Only the bumper side/cap materials use
the metal preset.

Use `[textures.materialize]` in `karty-package.toml` to configure an environment
export, or in `level.toml` for a local texture:

```toml
[[textures]]
name = "bumper-red-metal"
source = "textures/bumper-red-metal.png"
profile = "environment"
[textures.materialize]
preset = "metal"
```

The optional numeric controls are `normal_strength`, `height_contrast`,
`metallic_scale` and `ao_depth_scale`. Each accepts a finite value from 0 to 16.
Omission preserves the preset default; an explicit zero remains an override.
Materialize settings apply only to environment textures. Sprite decals use their
ordinary sprite profile.

```yaml
uv: { mode: wrap, scale: { x: 1, y: 1 } }
# In your room:
floor_material: fps-kit.floor-gray-grid
ceiling_material: fps-kit.ceiling-dark-panel
# On a boundary edge:
material: fps-kit.wall-blue-panel
```

Keep the room properties in their normal YAML locations; this snippet shows
the individual settings. Increasing UV scale to `{x: 2, y: 2}` makes one repeat
cover two metres; its calibration marks then represent that larger scale.

## Placeholder decals

![Transparent placeholder decal inventory](decals.png)

The package also exports 36 original **256×256 transparent PNG decals**:

| Prefix            | Names                                                                                                                    |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `decal-shape-`    | cross, triangle, circle, square                                                                                          |
| `decal-number-`   | 0–9                                                                                                                      |
| `decal-feedback-` | nice, great, ok, yes, no, wrong                                                                                          |
| `decal-icon-`     | question, exclamation, eye, bomb, explosion, rocket, star, heart, flag, target, arrow, key, lock, skull, waypoint, check |

Use `fps-kit.` before each name, for example `fps-kit.decal-icon-eye`. The
outlined marks stay readable on light and dark surfaces. Their background is
transparent; the dark preview sheet is not part of the individual decals.
They are visual placeholders with no collision or built-in goal/AI behavior.
Give each placement an ID and tags so game scripts can find or replace it.

Place one as an ordinary fixed world sprite in `world.yaml`:

```yaml
contents:
  - id: temporary-goal
    kind: actor
    position: { x: 4, y: 7.98, z: 1.6 }
    actor:
      tags: [placeholder, goal]
      sprite:
        {
          texture: fps-kit.decal-icon-star,
          facing: fixed,
          alpha: cutout,
          width: 0.8,
          height: 0.8,
          origin_x: 0.5,
          origin_y: 0.5,
        }
```

This example faces toward -Y on a north wall at Y=8. Fixed decals are one-sided:
use `yaw_degrees: -90` on an east wall, `90` on a west wall, and `180` on a south
wall. For a floor decal, use `pitch_degrees: 90` and a small positive Z offset
such as `0.015`. Keep wall placements slightly inside the room to avoid
z-fighting. Width and height use metres; no separate PNG declaration is needed.

The FPS starter includes room numbers, a floor arrow and a goal star. The
`examples/decals` room shows the whole set on a wall plus floor examples; copy
its two files into `levels/room` to preview it with the starter. Use
`game.World(handle).FindEntity("temporary-goal")` or query the `goal` tag for
scripting, then replace the entity's sprite or destroy it when real content is
available.

Decals use the ordinary sprite profile and do not generate Materialize maps.
Only selected decals enter the level. The build's default sprite cap reduces
their runtime images to at most 128px, independently of the 256px source;
world units per pixel and explicit level caps can reduce them further. The 36
source PNGs total about 53 KiB. There is no emoji-font or external-art dependency.

## Static and dynamic crates

| Size   | Width × depth × height (metres) | Static prefab  | Dynamic geometry       |
| ------ | ------------------------------- | -------------- | ---------------------- |
| Small  | 0.5 × 0.5 × 0.5                 | `crate-small`  | `crate-small-dynamic`  |
| Medium | 1 × 1 × 1                       | `crate-medium` | `crate-medium-dynamic` |
| Large  | 2 × 2 × 2                       | `crate-large`  | `crate-large-dynamic`  |
| Long   | 2 × 1 × 1                       | `crate-long`   | `crate-long-dynamic`   |
| Tall   | 1 × 1 × 2                       | `crate-tall`   | `crate-tall-dynamic`   |

Static prefabs have centered X/Y footprints and a floor-level origin at Z=0.
They include collision and need no game code:

```yaml
instances:
  - id: storage
    prefab: fps-kit.crate-medium
    transform: { translation: { x: 4, y: 6, z: 0 } }
```

Dynamic geometry has a centered XYZ origin and no baked collision. A runtime
`Solid3D` plus a matching dynamic box body makes it movable and pushable. Keep
these out of static `instances`, which bake immovable world geometry.

The example room already declares local renderer wrappers for all five sizes.
For your own level, copy only the wrappers you need from its `prefabs` list:

```yaml
prefabs:
  - id: crate-medium-dynamic
    renderable: true
    instances:
      - id: shape
        prefab: fps-kit.crate-medium-dynamic
```

This explicitly includes the package geometry in the level's runtime renderer
catalog. Use the wrapper's local ID (`crate-medium-dynamic`) in Go. Unused
package exports are otherwise omitted by the package resolver.

Copy `examples/crates.go.tmpl` to your game's `src/crates.go`, change its import
to your game's module, and replace the FPS starter's `game.addCrates` call with
this in its `ready` callback, after attaching the player:

```go
if err := spawnCrates(&game.Game, event.Handle); err != nil {
    game.label.SetText("Could not create crates: " + err.Error())
    return
}
```

The helper spawns all five sizes above the floor with gravity, rotation,
friction and matching collider dimensions. Its positions leave room around the
gallery's static crates, bumpers, cover, step and player spawn. Keep the runtime wrappers for all
five when using it unchanged. Dynamic origin height is half the crate height
when resting on the floor. Visual scaling does not resize physics colliders;
update both together. Level release owns entity/body cleanup. For moving stacks,
spawn separate crates/bodies at successive heights.

Additional static exports:

| Prefab         | Dimensions and behavior                         |
| -------------- | ----------------------------------------------- |
| `crate-stack`  | Three 1 m crates, 2 × 1 × 2 m overall           |
| `cover-low`    | 2 × 0.25 × 1 m barrier with hazard top          |
| `bumper-low`   | 1.7 m octagon, 0.75 m high; static metal bumper |
| `bumper-cover` | 1.7 m octagon, 1.25 m high; static metal bumper |
| `bumper-tall`  | 1.7 m octagon, 2 m high; static metal bumper    |

Author slopes directly in your level’s sectors as needed.

Bumpers have centered X/Y footprints and floor-level origins. Each uses one
collision-enabled octagonal solid, red metal sides and metal caps. The physics
lab uses all three heights.

Package instances accept ordinary rotation, uniform scale and material remaps.
Crate frames are geometry, so they remain aligned at every size, origin and
rotation. Seamless wood textures contain no painted frame or diagonal brace.
Each crate uses one inset panel box, two caps and four corner posts: 84 triangles
before compiler processing (6,720 triangles for the physics lab's 80 crates).
Moving crates still use one box collider each. All sizes share three 256px wood
textures. There are no external meshes or continuously generated assets.

## Size, sources and sharing

The 49 environment PNGs decode to 12.25 MiB of RGBA pixels if all are selected,
before the compiled material atlas and generated material data.
Karty normalizes world materials to 256×256 atlas tiles and adds its own padding,
maps and mip levels. PNG size is not runtime GPU memory or a frame-rate claim.
Only selected exports enter a consuming level. Dynamic renderer definitions and
instances also carry their ordinary geometry/body costs.

The artwork is original, generated from tracked geometric recipes, and covered
by the included MIT license. The supplied reference images are not redistributed
or used as texture pixels. The generator owns the PNGs, manifest and crate frame
geometry; handwritten static extras live in `scripts/fps-kit/props.yaml`.
From the CLI repository root:

```sh
mise run generate-fps-kit
mise run check-fps-kit
karty package export --output /tmp/fps-kit-0.1.0.kpkg packages/fps-kit
```

Import that `.kpkg` into another game for a fixed, checksummed copy. Neither
import nor export runs the generator. Version 0.1.0 is local source; no published
package or SDK release is implied.

Maintainers publish a versioned `.kpkg` and its SHA-256 checksum, then update the
SDK starter's archive URL and checksum together. The canonical artwork and
package documentation stay in this repository; the SDK template declares the
release dependency. A local fixture export establishes package behavior only;
it does not establish that the public release is available.
