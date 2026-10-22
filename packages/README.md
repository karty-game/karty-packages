# Game packages

Reusable code and assets for Karty games. Each folder contains a package
manifest, source and its license. Code packages include standalone Go modules;
resource-only packages need no Go dependency.

| Package                                                  | Use it for                                                    |
| -------------------------------------------------------- | ------------------------------------------------------------- |
| [3D Controllers](controllers-3d/README.md)               | First-person movement, mouse look and touch controls          |
| [Isometric Controllers](controllers-isometric/README.md) | Camera-relative movement, picking and camera framing          |
| [Top-down Controllers](controllers-topdown/README.md)    | 2D movement, pointer targets and camera framing               |
| [FPS Kit](fps-kit/README.md)                             | 256px materials, crates and placeholder decals                |
| [Dialogue](dialogue/README.md)                           | Yarn Spinner compilation, story execution and visual novel UI |
| [Savegame](savegame/README.md)                           | Separate profile data and saved run slots                     |
| [Tiled](tiled/README.md)                                 | Orthogonal Tiled JSON maps, image importing and 2D sprites    |

The three controller packages can be installed and released independently.
[Migration from the combined package](controllers-v2.md) explains their module
paths and type boundaries.

Playable first-person cameras are also built into the generated SDK through
`Game.AttachFirstPerson`. This component requires the SDK 0.0.18 candidate and
matching host artifacts, prepared locally until that version is published.
Use the portable 3D package when you need to own the integration instead.

## Add a package

From your game directory, import a folder from this checkout:

```sh
karty package import /path/to/karty-packages/packages/controllers-topdown
```

The CLI records the dependency in `karty.toml`. Folder imports link to the
source during development. To share a fixed copy, export an archive from this
repository root:

```sh
mise run package -- controllers-topdown
```

Another game can import the resulting `.kpkg` file with `karty package import`.
Read each package's README for its compiler, SDK and host requirements.
Builds restore declared packages under `.karty/packages/`; edit the original
source folder and keep it or the archive available to the game.
