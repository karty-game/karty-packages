# First-person player

Attach a first-person controller to a live camera after its level is ready:

```go
options := fpscontroller.DefaultFirstPersonOptions()
options.Speed = 4 // metres per second
player, err := fpscontroller.AttachFirstPerson(game.Game, camera, options)
if err != nil {
    return err
}
```

The controller supplies keyboard and pointer input, mouse capture, visible
mobile sticks, a player body, gravity and collision. It updates automatically;
you do not need controller input hooks or an `OnUpdate` movement adapter.
Keep ordinary game hooks for your own interactions, weapons and objectives.

## Controls and movement

WASD/arrows walk, Shift runs, Q/E turn, click requests mouse look and Escape
releases it. Mouse capture still requires the browser's permitted user gesture.
On touch input, the lower-left stick moves and the lower-right stick looks.
Each stick retains its own contact until release or cancellation. The display
is created automatically on first touch and does not intercept game/UI input.

The controller uses the camera's logical viewport dimensions by default. Start
with `DefaultFirstPersonOptions`, then adjust speed, sensitivity, pitch limits,
body settings or the nested `Touch` appearance/gesture settings. Explicit zero
values are preserved; an invalid configuration returns an error before attaching.
If the camera renders at a different resolution from the game's logical canvas,
set `Width` and `Height` to that canvas size. The host already maps pointer
coordinates; do not scale incoming events again. Floating sticks follow their
touch origins, including when a portrait view crops the logical horizontal edges.

Physics mode starts from the host's accepted pose and submits horizontal
velocity at the fixed 60 Hz update rate. The host retains vertical momentum and
resolves gravity and collisions. The default upright capsule places the eyes
1.7 metres above the feet. Its shared character motor walks slopes up to
50 degrees and climbs supported steps up to 0.25 metres when headroom permits.
Games do not need authored stair-height maps or positional lifts. Tall ledges,
steeper slopes and blocked headroom remain obstacles. An existing world's
physics configuration is retained. Jumping is an explicit gameplay extension
rather than an implicit Space binding.

For an inspection camera, set `options.Mode = fpscontroller.FirstPersonFree`. This
omits the player body and applies movement directly to the camera. Camera
collision settings still apply.

## Lifetime and gameplay extensions

One local first-person controller may be attached at a time. A second attachment
returns `ErrCapacity`; close the old controller before switching players.
The game owns its camera. The controller owns the body and touch display it
creates, and releases them when closed, when its camera is destroyed, when its
level is released, or when the cartridge shuts down.

`player.SetEnabled(false)` releases controls and stops controlled movement.
Re-enabling starts with fresh input. Focus loss and modal input capture also
reset held controls. `player.Teleport(position, yaw, pitch)` clears input and
momentum while retaining the camera/body identities. `Close` is idempotent.

Accepted poses and input are dispatched before controller movement; the game
update runs afterwards. This order applies to both `engine.Hooks` and custom
`engine.Client` implementations.

For custom movement, `BeforeMove` can adjust a proposal and `Move` can replace
its application. Call `ApplyMotion` from a movement override for ordinary
walking. These optional hooks let a game add swimming or authored traversal
without implementing another input router or drawing its own mobile controls.
The proposal pointer is borrowed for the `BeforeMove` call; do not retain it.
`OnReset` resets any game-owned movement state. A custom scene hit region can
use `SceneContains`; standard host UI input filtering already applies.

Construction and first touch display may allocate. Steady movement/input and
unchanged overlay updates use fixed storage. Public portable controller math
remains available for games implementing different control systems.
