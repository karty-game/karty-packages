# Controller packages v2

Status: source packages prepared at `0.0.1`; no publication is implied.

The combined `controllers` catalog entry and source module are replaced by three
independent packages. Each includes its input helpers and its own controller
implementation, without shared package dependencies or private engine source.

| Package                 | Go module                                     | Constructor      |
| ----------------------- | --------------------------------------------- | ---------------- |
| `controllers-3d`        | `github.com/karty-game/controllers-3d`        | `NewFirstPerson` |
| `controllers-isometric` | `github.com/karty-game/controllers-isometric` | `NewIsometric`   |
| `controllers-topdown`   | `github.com/karty-game/controllers-topdown`   | `NewTopDown2D`   |

Import the matching folder or archive and update the Go import path. Remove the
old `controllers` dependency from `karty.toml` and `go.mod` after migrating all
imports. All three retain the Go package identifier `controllers`; use aliases
when importing multiple styles. Cross-package vectors and options are distinct
types. Top-down options omit the unused `CameraOffset`, `Yaw` and `Pitch` fields.

Each style retains its existing constructor/method names, default speeds, fixed
60 Hz updates, transactional configuration, input ownership/reset semantics and
steady zero-allocation behavior. Both planar modules own their camera composer
and projection implementation. Physics, navigation, animation and UI remain
game-owned. The 3D package supplies portable first-person calculations; the SDK
also offers its built-in `Game.AttachFirstPerson` component.

The catalog and release choices identify the packages independently. Previously
published combined archives and already-pinned projects are unchanged. Embedded
SDK starter copies retain their existing source contracts. This split changes no
engine API, wire protocol, SDK manifest or package archive format.

Previews are unchanged captures from Crate Room, Vigil and Willowbrook, showing
their respective view styles. Sample artwork/gameplay is not included. Crate Room
uses the SDK built-in first-person component. Each package records its image
source in `previews/README.md` and includes the MIT license.
