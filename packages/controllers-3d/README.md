# 3D Controllers for Karty

![Crate Room sample preview](previews/controller.jpg)

A ready-to-play first-person player: keyboard and mouse input, two touch sticks,
mouse capture, a host physics body, camera movement and automatic cleanup.
The package owns controller behavior and presentation. The SDK supplies generic
entity, input, component lifetime and physics primitives.

## Install

Version 0.0.2 is a release candidate requiring SDK 0.0.18 or newer. Until its
archive is published, import this working folder explicitly:

```sh
karty package import /path/to/karty-packages/packages/controllers-3d
```

Select the package's Go adapter processor in `karty.toml`:

```toml
[[build.processors]]
use = "controllers-3d.sdk"
inputs = ["go.mod"]
```

The processor reads the authored module name and generates its typed adapter
under `.karty/processed/code/controllers-3d.sdk/controller/`. Builds own this
output; keep package sources and project configuration as the authored inputs.

## Make a camera playable

After creating a camera in a ready level:

```go
import (
    "example.com/my-game/.karty/engine"
    fpscontroller "example.com/my-game/.karty/processed/code/controllers-3d.sdk/controller"
)

options := fpscontroller.DefaultFirstPersonOptions()
options.Speed = 4
player, err := fpscontroller.AttachFirstPerson(engine.Game{}, camera, options)
// Handle err, then retain player for pause, teleport or cleanup.
```

The package registers with the SDK's generic owner-scoped component lifecycle.
Input and accepted poses reach the player before movement, and game updates run
next. Closing the controller, destroying its camera, releasing its level or
shutting down the cartridge releases its body and touch display. Game hooks
remain available for interactions, weapons and objectives.

WASD/arrows walk, Shift runs, Q/E turn, clicking requests mouse capture and
Escape releases it. Touch sticks retain independent fingers. Options preserve
fixed movement sticks, drag look, deadzones, appearance, scene hit testing,
input reset callbacks and custom movement hooks.

`SetEnabled(false)` clears held input while retaining falling momentum.
`Teleport` clears input and momentum without replacing the camera or body.
`FirstPersonFree` omits the body for inspection cameras. `BeforeMove`, `Move`
and `ApplyMotion` support gameplay extensions such as swimming.
See [the complete controller contract](first-person.md).

## Portable input math

The original Go API remains available at `github.com/karty-game/controllers-3d`
with Go package identifier `controllers`. `NewFirstPerson`, `Key`, `Pointer`,
`Step`, `Reset` and the direct touch methods support custom adapters. These
portable options are distinct from the generated SDK adapter's typed options.
Version 0.0.1 archives remain immutable for existing consumers.

The `.kpkg` package includes the implementation, bridge templates, processor and
license. It needs no private engine source. Go 1.27 is required; `golang.org/x/mod`
is used only by the build-time module parser. Generated game code imports the
project's SDK directly.

## Validate and release

From the packages repository root:

```sh
mise run test
mise run check-packages
mise run package -- controllers-3d
```

Tests cover deterministic processor output, malformed requests, portable input
math and the package-owned player policy using a small public SDK test double.
Consumer checks separately compile the real SDK adapter with Go and TinyGo and
exercise the actual browser renderer. Native doubles do not establish WASM,
physics rendering or device performance.
