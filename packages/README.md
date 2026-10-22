# Game packages

Reusable code and assets for Karty games. Each folder contains a package
manifest, source and its license. Code packages include standalone Go modules;
resource-only packages need no Go dependency.

| Package                              | Use it for                                                    | Example                                                                              |
| ------------------------------------ | ------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| [Controllers](controllers/README.md) | First-person, top-down 2D and isometric movement and cameras  | [Willowbrook](../samples/willowbrook/README.md), [Vigil](../samples/vigil/README.md) |
| [FPS Kit](fps-kit/README.md)         | 256px materials, static/dynamic crates and placeholder decals | [Ready-to-copy room](fps-kit/examples/room/world.yaml)                               |
| [Dialogue](dialogue/README.md)       | Yarn Spinner compilation, story execution and visual novel UI | [Last Train](../samples/last-train/README.md)                                        |
| [Savegame](savegame/README.md)       | Separate profile data and saved run slots                     | [Last Train](../samples/last-train/README.md)                                        |
| [Tiled](tiled/README.md)             | Orthogonal Tiled JSON maps, image importing and 2D sprites    | [Willowbrook](../samples/willowbrook/README.md)                                      |

Playable first-person cameras are built into the generated SDK through
`Game.AttachFirstPerson`; they do not require the Controllers package.
This component requires the SDK 0.0.18 candidate and matching host artifacts,
prepared locally until that version is published.

## Add a package

From your game directory, import a folder from this checkout:

```sh
karty package import /path/to/karty/packages/controllers
```

The CLI records the dependency in `karty.toml`. Folder imports link to the
source during development. To share a fixed copy, export an archive:

```sh
karty package export --output controllers.kpkg /path/to/karty/packages/controllers
```

Another game can import that `.kpkg` file with `karty package import`.
Read each package's README for its compiler, SDK and host requirements.
Builds restore declared packages under `.karty/packages/`; edit the original
source folder and keep it or the archive available to the game.
