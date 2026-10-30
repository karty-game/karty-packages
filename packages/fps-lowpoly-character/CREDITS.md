# Asset credits

## Vinrax

Original painted model/UV foundations:

- [PSX Cop Character](https://vinrax.itch.io/psx-cop-character): original policeman
  and the common body foundation.
- [PSX Casual Male Character](https://vinrax.itch.io/psx-casual-male-character):
  the scientist's complete head/hair foundation.
- [PSX Female Secretary Character](https://vinrax.itch.io/psx-secretary-character):
  the guard's complete female head foundation.

Karty adaptations include fitted shared-rig weights, role costumes, face/hair
texture variants, glasses, helmets, neckline interfaces and the scientist's
outward surface correction. Original artist attribution must remain visible.
The scientist/guard face texture variants used built-in image generation;
the preserved prompts and provenance are under `source/`.

The downloaded-source records describe CC BY 4.0 and link the creator's itch.io
pages, which permit project use with credit. Current corresponding Sketchfab
listings do not provide a consistent basis for that historical assertion: the
[cop listing](https://sketchfab.com/3d-models/cop-character-psx-style-5576b8cbd33343f998c8b86d8bcfda48)
still lists CC Attribution, while the
[casual male listing](https://sketchfab.com/3d-models/male-character-ps1-style-78640adbd76b48129e9316259695c742)
now lists Free Standard/NoAI. Historical archive hashes below record provenance,
not a new license grant. Resolve the asset-redistribution terms before publishing
this package. The MIT code license does not relicense these models or textures.

| Original source archive        | SHA-256                                                            |
| ------------------------------ | ------------------------------------------------------------------ |
| PSX Cop Character              | `daa5db5ccba836f7d67a0a57efe746bbf525c0bdd4d82733ea8c4bf3a868abd2` |
| PSX Casual Male Character      | `bb33671cfd4c74922390d7b305f119c28b3f9f3e58e361963b4efe4346cb7e18` |
| PSX Female Secretary Character | `9f556b1009283d62d83cc67e5240802b525363ff61dc2c014b2f277d147fcc21` |

## Quaternius

The soldier head, eyes and brows derive from
[Universal Base Characters](https://quaternius.com/packs/universalbasecharacters.html),
[CC0 1.0 Universal — Public Domain Dedication](https://creativecommons.org/publicdomain/zero/1.0/).
Models by Quaternius. Adaptations reduce head geometry and fit the common rig and costume.

## Mesh2Motion

The user-supplied policeman export is the canonical rig and 20-clip input.
Its SHA-256 is
`30b7d48fdee566fadff47a1cc4fdea1bc35eac30a1a05f784661fa28738f66e5`.
Its exact bytes are retained in `source/policeman-original.glb`.
[Mesh2Motion](https://github.com/Mesh2Motion/mesh2motion-app#licenses) identifies
its rigs and animations as
[CC0 1.0 Universal](https://github.com/Mesh2Motion/mesh2motion-app/blob/main/LICENSE-CC0.MD);
its software is MIT. The public-domain dedication is available at
https://creativecommons.org/publicdomain/zero/1.0/.
This does not change the supplied policeman model's separate Vinrax terms.

The shared bank uses the previous source-key-preserving optimization. It contains
all 20 supplied clips without retargeting, added animation or key-time resampling.
No Mixamo, CMU, Valve or Quaternius animation library is substituted in this
package. The adapted model-only GLBs use Meshopt compression; some textures use
WebP.
