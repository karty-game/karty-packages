package kpkg

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestInvalidSourceNamesAndSymlinksFailBeforeExport(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"../escape", "/absolute", `back\slash`, "NUL.png", "trailing.", "bad\x00name", "a/../b"} {
		if validPath(name) {
			t.Fatalf("accepted nonportable path %q", name)
		}
	}

	for _, mode := range []string{"symlink", "case-component", "inventory", "oversized-file", "oversized-manifest"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			root, directory := fixture(t)
			switch mode {
			case "symlink":
				outside := filepath.Join(t.TempDir(), "outside")
				writeFixture(t, outside, []byte("private"))
				if err := os.Symlink(outside, filepath.Join(directory, "leak")); err != nil {
					t.Fatal(err)
				}
			case "case-component":
				writeFixture(t, filepath.Join(directory, "Assets/a"), []byte("a"))
				writeFixture(t, filepath.Join(directory, "assets/b"), []byte("b"))
			case "inventory":
				writeFixture(t, filepath.Join(directory, InventoryName), []byte("{}"))
			case "oversized-file", "oversized-manifest":
				name, size := "large.bin", int64(MaxFileBytes+1)
				if mode == "oversized-manifest" {
					name, size = ManifestName, MaxManifestBytes+1
				}

				path := filepath.Join(directory, name)
				writeFixture(t, path, []byte("x"))
				if err := os.Truncate(path, size); err != nil {
					t.Fatal(err)
				}
			}

			if _, err := Pack(t.Context(), root, "sample-kit", filepath.Join(root, "out"), ""); err == nil {
				t.Fatalf("invalid source mode %s was exported", mode)
			}
		})
	}
}

func TestManifestIdentityVersionAndDeclaredAssets(t *testing.T) {
	t.Parallel()
	for _, contents := range []string{
		"format_version=2\nname='sample-kit'\nversion='0.1.0'\n",
		"format_version=1\nname='different'\nversion='0.1.0'\n",
		"format_version=1\nname='sample-kit'\nversion='v0.1.0'\n",
		"format_version=1\nname='sample-kit'\nversion='0.1'\n",
		"format_version=1\nname='sample-kit'\nversion='0.1.0+mutable'\n",
		"format_version=1\nname='sample-kit'\nversion='0.1.0'\nunknown=true\n",
		"format_version=1\nname='sample-kit'\nversion='0.1.0'\nprefabs='../escape.yaml'\n",
	} {
		t.Run(contents, func(t *testing.T) {
			t.Parallel()
			root, directory := fixture(t)
			writeFixture(t, filepath.Join(directory, ManifestName), []byte(contents))
			if _, err := Resolve(t.Context(), root, "sample-kit"); err == nil {
				t.Fatal("invalid manifest was accepted")
			}
		})
	}

	for _, source := range []string{"../texture.png", "/absolute.png", `back\slash.png`, "missing.png", "bad.png"} {
		t.Run(source, func(t *testing.T) {
			t.Parallel()
			root, directory := fixture(t)
			data := readFixture(t, filepath.Join(directory, ManifestName))
			data = append(data, []byte("[[textures]]\nname='wall'\nprofile='environment'\nsource="+strconv.Quote(source)+"\n")...)
			writeFixture(t, filepath.Join(directory, ManifestName), data)
			writeFixture(t, filepath.Join(directory, "bad.png"), []byte("not a PNG"))
			if _, err := Resolve(t.Context(), root, "sample-kit"); err == nil {
				t.Fatal("invalid declared texture was accepted")
			}
		})
	}
}

func TestMaterializeSettingsAndTransparentSprites(t *testing.T) {
	t.Parallel()
	for _, test := range []struct{ profile, options string }{
		{"environment", "preset='metal'\nnormal_strength=0\nmetallic_scale=1\n"},
		{"sprite", ""},
	} {
		root, directory := fixture(t)
		pixels := image.NewNRGBA(image.Rect(0, 0, 3, 7))
		pixels.SetNRGBA(1, 2, color.NRGBA{R: 255, A: 128})
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, pixels); err != nil {
			t.Fatal(err)
		}

		writeFixture(t, filepath.Join(directory, "texture.png"), encoded.Bytes())
		contents := readFixture(t, filepath.Join(directory, ManifestName))
		contents = append(contents, []byte("[[textures]]\nname='wall'\nsource='texture.png'\nprofile="+strconv.Quote(test.profile)+"\n")...)
		if test.options != "" {
			contents = append(contents, []byte("[textures.materialize]\n"+test.options)...)
		}

		writeFixture(t, filepath.Join(directory, ManifestName), contents)
		if _, err := Resolve(t.Context(), root, "sample-kit"); err != nil {
			t.Fatalf("valid non-256 PNG/profile rejected: %v", err)
		}
	}

	for _, options := range []materialize{{Preset: "unknown"}, {NormalStrength: floatPointer(-1)}, {MetallicScale: floatPointer(17)}} {
		if err := validateMaterialize(options); err == nil {
			t.Fatal("invalid Materialize override accepted")
		}
	}
}

func floatPointer(value float64) *float64 { return &value }

func TestCancellationAndOutputOwnership(t *testing.T) {
	t.Parallel()
	root, directory := fixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := Pack(ctx, root, "sample-kit", filepath.Join(root, "out"), ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled package export succeeded: %v", err)
	}

	if _, err := Pack(t.Context(), root, "sample-kit", filepath.Join(directory, "release"), ""); err == nil {
		t.Fatal("exported artifacts into their own source tree")
	}
}
