# Integration

The package owns appearance assets and animation selection. A game owns the
actor's position, navigation decisions, damage and requests. Physics owns accepted
displacement/contact. The renderer owns sampling, skin deformation and resource
lifetime. Keep these separate so a player and an NPC can share the same visuals.

`Clips()` maps aliases to exact source names, durations and loop/one-shot rules.
The `aware` alias selects the existing `Confused` clip. Talking and aware are
available for direct playback; locomotion does not automatically select them.
`Models()` returns model paths, triangle counts and joint counts. The delivered
scientist/guard/assault/policeman GLBs have 1,320/1,456/2,181/782 triangles and
66 rig joints each. The assault authoring source has 2,182 triangles before its
existing optimization.

## Export a local archive

From the packages repository root:

```sh
mise run package -- fps-lowpoly-character
```

Import `dist/releases/fps-lowpoly-character-0.0.1.kpkg` with
`karty package import`. No public release URL exists yet. The module import is
`github.com/karty-game/fps-lowpoly-character`, with Go package name `character`.

## Fixed-tick update

1. Player input or NPC steering requests movement through the game's character
   motor. Preserve host collision/gravity; do not integrate a second visual pawn.
2. Convert the accepted world displacement divided by the tick duration to local
   right/forward velocity using actor yaw. Supply actual vertical velocity and
   ground contact. A blocked actor consequently returns to idle.
3. Call the retained controller with `0 < dt <= 0.1` seconds. Fire/reload/use
   requests are pulses; clear each after submitting it. A rejected request is
   discarded, not queued. Death overrides actions until `Reset`.
4. Resolve output aliases through `Clips()`, then sample the resident bank.
   Each actor retains independent clocks and pose/blend state while geometry,
   textures and animation data can be shared.

Reload/fire use independent real-time clocks while locomotion follows distance.
Changing equipment, taking off or becoming prone cancels weapon actions.
Interaction requires standing/crouching, grounded and stationary; movement
interrupts it. Standing forward jog begins at 3.6 m/s. The inherited reference
speeds were measured for the preview's 1.8 m guard; model-scale changes require
calibration. See the known strafe defect below before relying on it.

## Pose sampling

`BaseTime` advances positively by `dt * abs(Rate)`. For a looping base clip,
reduce modulo its duration; when `Rate < 0`, sample the duration minus that phase.
Clamp one-shots at the last source key. `UpperTime` is independent; apply the
upper clip's own loop/clamp rule. `Rate == 0` holds the base pose. Do not multiply
`BaseTime` by `Rate` again. Source clips can begin after time zero; clamp samples
before the first key to that first key. Death holds at the last key until reset.

`Transition` changes when aliases/state change and on reset. A renderer can use
it to snapshot the current composed pose and begin a transition blend. This
package does not implement pose interpolation or mandate a blend duration.

For upper-body composition, apply rotations from `spine_01` and descendants over
the base pose; base translations, pelvis and legs keep their own clock. Resolve
parents and mesh-node transforms before building inverse-bind joint palettes.
Load/warm the complete required cast and bank before play. Changing a state or
appearance must only select resident resources.

## Karty integration

The next playable sample belongs in the CLI repository. It should consume a
checksummed package archive, use the ordinary first-person motor for the player
and separate animation controllers for NPCs. AI can later supply desired motion,
facing and action requests without choosing animation frames or changing assets.

The released SDK needs a skinned-character resource and playback API before a
sample can render these models. Importing this package provides the GLBs and Go
animation controller. It does not provide a host character renderer. No private
engine source is required to test or archive the package.

## Assets and runtime requirements

- `models/*.glb`: model, rig and embedded textures, without duplicate animation.
- `animations/shared.glb`: the common animation bank. Bind by canonical node
  name, including non-joint ancestors; GLB node indices differ between files.
- `source/*.blend`: editable current role sources with packed textures/actions.
  `source/policeman-original.glb` retains the supplied canonical rig/motion input.
- `assets.json`: file sizes/SHA-256, model inventory and the clip contract.

GLBs require `KHR_mesh_quantization` and `EXT_meshopt_compression`; role textures also require
`EXT_texture_webp`. A compatible authoring/importer tool must decode these
extensions. The controller does not embed GLBs or Blender sources into guests.

## Inherited limits

Strafe direction/contact calibration is known to be wrong, and lateral running
can reach the inherited 8x playback cap. Packaging preserves that behavior for
comparison; correct and visually review it before using lateral NPC locomotion.
Falling reuses `Jump_air`, prone idle holds a crawl pose, and crouch/crawl do not
resize the collision capsule. There are no dedicated swim, hurt, rifle,
crouched-reload or notebook clips or weapon meshes. Source deaths can intersect
the nominal floor. This is not the complete earlier 18-action art wish list.

Pose blending, lighting/shadows and graphics resource lifetime belong to the
renderer. Device-performance and visual acceptance still need validation after
SDK rendering support is added. See [validation](VALIDATION.md).
