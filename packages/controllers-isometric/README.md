# Isometric Controllers for Karty

![Vigil sample preview](previews/controller.jpg)

_Vigil, an existing Karty sample showing this view style. The sample's
artwork and gameplay are separate from this package._

Camera-relative movement and ground picking for fixed isometric views,
including sprite-based 2D games presented with an orthographic 3D camera.
Includes following, zoom and optional camera zones/lookahead. Movement uses
world X/Y metres; picking assumes Z=0. The default camera yaw is 45 degrees,
pitch is -30 degrees and orthographic height is 13.2.

## Install

From your game directory:

```sh
karty package import /path/to/karty-packages/packages/controllers-isometric
```

The package name is `controllers-isometric`, version `0.0.1`. Its standalone Go module is
`github.com/karty-game/controllers-isometric`; its Go package identifier remains `controllers`.
It requires Go 1.27 and has no external dependencies or generated SDK bindings.
To create a portable archive, run from the packages repository root:

```sh
mise run package -- controllers-isometric
```

This produces `dist/releases/controllers-isometric-0.0.1.kpkg` and its checksum.
The three controller packages release independently.

## Connect a player

```go
import "github.com/karty-game/controllers-isometric"

// During player setup:
player, err := controllers.NewIsometric(controllers.DefaultIsometricOptions())
// Handle err before using player.
```

`Step(acceptedPosition)` returns a movement displacement. Apply your collision
rules, then feed the accepted position to `Follow(currentFocus,
acceptedPosition, snap)`. `FollowOffset` shifts the camera focus; `FollowRate`
controls smoothing per update. Isometric movement uses the configured camera yaw.

`SetTarget(worldPosition)` moves directly toward a pointer target without
overshooting it. Keyboard or stick input takes over and cancels that target.
`CancelTarget` also stops a blocked target. A navigation system can use
`Speed()/60` as its travel budget. Pathfinding, collision response and animation
remain game-owned.

`ZoomBy(factor)` clamps to `MinZoom`/`MaxZoom`. Apply the result as orthographic
height to the camera; `CameraPosition(focus)` adds its configured fixed 3D
offset.

`ScreenToGround(screen, focus, width, height)` uses the current camera focus
and logical projection rectangle, including its current zoom. Do not pass CSS
display dimensions. It returns false for invalid dimensions. The projection
must match the configured yaw, pitch and Z=0 ground.

## Camera framing

Create a `CameraComposer` with `NewCameraComposer(DefaultCameraFramingOptions())`
and replace `Follow` with `player.ComposeCamera(composer, currentFocus,
acceptedPosition, width, height, snap)`. It returns the focus used for picking.
Pass `snap=true` on spawn, teleport and discontinuous portal transitions.

The composer provides per-axis half-life damping, dead/soft zones, normalized
screen position and bounded lookahead from accepted movement. `MaxLookahead`
uses projected metres. Reset both the composer and input when replacing a shot.
[Camera framing contract](docs/camera-framing-v1.md) describes bounds, invalid
updates and viewport handling.

## Input and lifetime

Map your game's keys to `Forward`, `Backward`, `Left`, `Right`, and the `Alt`
controls for independent arrow-key aliases. Use `RunLeft` and `RunRight` for
the two Shift keys. Releasing one alias preserves another held key.
`BeginTouch`, `MoveTouch` and `EndTouch` support game-owned touch controls;
coordinates use logical viewport pixels. Route move/end/cancel to an owned
finger even when it leaves the scene. Hit-test UI before beginning a gesture.

Call `Reset` on focus loss, pause or player/view replacement. Start from the
default options, change the fields you need, then construct. `Configure`
validates transactionally and resets input; explicit zero speeds remain zero.
Keep one controller per player and do not copy it with active input. Calls run
once per fixed 60 Hz update, independently of rendering. Construction can
allocate; steady input and movement updates allocate no memory.

## Migration

This replaces the corresponding preset from the combined `controllers` package.
Import this package folder and change the Go import to the module above. Existing
constructor and method names for this preset are retained. Use import aliases if
a game needs multiple controller packages; their vector/option types are distinct.

Older combined archives and already-pinned games remain unchanged. See the
[split contract](../controllers-v2.md) for the package mapping.

## Validate

From the repository root, run `mise run test`, `mise run check-packages` and
`mise run build`. Each code package is tested through its own Go module.
