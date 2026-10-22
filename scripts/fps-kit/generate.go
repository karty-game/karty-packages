//go:build ignore

// Generate the original FPS blockout palette. Run from the packages repository root.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"golang.org/x/mod/semver"
)

const root = "packages/fps-kit"

// Design coordinates use a 128-unit grid with two-pixel strokes at export.
const size = 128
const textureSize = 256

type swatch struct {
	name  string
	base  color.RGBA
	style string
}

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 255} }

func mix(a, b color.RGBA, amount int) color.RGBA {
	channel := func(x, y uint8) uint8 { return uint8((int(x)*(100-amount) + int(y)*amount) / 100) }
	return rgb(channel(a.R, b.R), channel(a.G, b.G), channel(a.B, b.B))
}

func rect(dst *image.RGBA, x, y, w, h int, c color.RGBA) {
	draw.Draw(dst, image.Rect(x, y, x+w, y+h), image.NewUniform(c), image.Point{}, draw.Src)
}

func frame(dst *image.RGBA, inset, thickness int, c color.RGBA) {
	n := size - inset*2
	rect(dst, inset, inset, n, thickness, c)
	rect(dst, inset, size-inset-thickness, n, thickness, c)
	rect(dst, inset, inset, thickness, n, c)
	rect(dst, size-inset-thickness, inset, thickness, n, c)
}

// A compact, original bitmap alphabet keeps artwork independent of system fonts.
var glyphs = map[rune]string{
	'0': "111101101101111", '1': "010110010010111", '2': "111001111100111",
	'3': "111001111001111", '4': "101101111001001", '5': "111100111001111",
	'6': "111100111101111", '7': "111001010010010", '8': "111101111101111",
	'9': "111101111001111", 'M': "1000111011101011000110001",
}

func text(dst *image.RGBA, value string, x, y, scale int, c color.RGBA) {
	for _, letter := range value {
		bits := glyphs[letter]
		width := len(bits) / 5
		for i, bit := range bits {
			if bit == '1' {
				rect(dst, x+i%width*scale, y+i/width*scale, scale, scale, c)
			}
		}
		x += (width + 1) * scale
	}
}

func texture(s swatch, index int) *image.RGBA {
	if s.style == "metal" {
		return metal(s)
	}
	if s.style == "brick" || s.style == "concrete" || s.style == "wood" || s.style == "endgrain" || s.style == "batten" {
		return masonry(s)
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	ink := rgb(231, 239, 236)
	dark := mix(s.base, rgb(16, 23, 31), 38)
	light := mix(s.base, ink, 24)
	rect(dst, 0, 0, size, size, s.base)
	switch s.style {
	case "grid":
		for i := 16; i < size; i += 16 {
			rect(dst, i, 0, 1, size, mix(s.base, ink, 10))
			rect(dst, 0, i, size, 1, mix(s.base, ink, 10))
		}
		rect(dst, 64, 0, 1, size, light)
		rect(dst, 0, 64, size, 1, light)
		frame(dst, 0, 1, dark)
		rect(dst, 60, 63, 9, 3, ink)
		rect(dst, 63, 60, 3, 9, ink)
		// Quarter-metre edge ticks and one-metre repeat label.
		for i := 32; i < size; i += 32 {
			rect(dst, i, 2, 1, 4, ink)
			rect(dst, 2, i, 4, 1, ink)
		}
		text(dst, "1M", 10, 10, 1, ink)
	case "checker":
		rect(dst, 0, 0, 64, 64, mix(s.base, ink, 18))
		rect(dst, 64, 64, 64, 64, mix(s.base, ink, 18))
		frame(dst, 0, 1, dark)
		for _, p := range []image.Point{{32, 32}, {96, 32}, {32, 96}, {96, 96}} {
			rect(dst, p.X-3, p.Y, 7, 1, light)
			rect(dst, p.X, p.Y-3, 1, 7, light)
		}
	case "panel", "ceiling":
		frame(dst, 0, 2, dark)
		frame(dst, 7, 1, light)
		frame(dst, 9, 1, dark)
		for _, x := range []int{14, 110} {
			for _, y := range []int{14, 110} {
				rect(dst, x, y, 4, 4, dark)
				rect(dst, x, y, 3, 1, light)
			}
		}
		if s.style == "ceiling" {
			for y := 48; y <= 76; y += 7 {
				rect(dst, 40, y, 48, 2, dark)
				rect(dst, 40, y+2, 48, 1, light)
			}
		} else {
			rect(dst, 10, 91, 108, 17, dark)
			text(dst, fmt.Sprintf("%02d", index), 16, 96, 1, ink)
			for x := 77; x < 109; x += 7 {
				rect(dst, x, 97, 4, 3, light)
			}
		}
	case "hazard":
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				if (x+y)%32 < 16 {
					dst.SetRGBA(x, y, rgb(33, 41, 50))
				}
			}
		}
	}
	output := image.NewRGBA(image.Rect(0, 0, textureSize, textureSize))
	for y := range textureSize {
		for x := range textureSize {
			output.SetRGBA(x, y, dst.RGBAAt(x*size/textureSize, y*size/textureSize))
		}
	}
	return output
}

// Seamless brushed metal has no painted prop frame to misalign on the octagon.
func metal(s swatch) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, textureSize, textureSize))
	for y := range textureSize {
		for x := range textureSize {
			relief := (noise(x, y, 16)-.5)*5 + (noise(x/16*16, y, 2)-.5)*4
			if s.name == "bumper-red-metal" && x%128 < 2 {
				relief -= 20 // shallow repeating rolled-panel grooves
			}
			channel := func(v uint8) uint8 { return uint8(max(0, min(255, int(v)+int(relief)))) }
			dst.SetRGBA(x, y, rgb(channel(s.base.R), channel(s.base.G), channel(s.base.B)))
		}
	}
	return dst
}

// Periodic value noise joins exactly across the 1 m repeat. Albedo has no
// directional light/shadow; luminance relief feeds Karty's Materialize pipeline.
func noise(x, y, period int) float64 {
	n := textureSize / period
	hash := func(x, y int) float64 {
		v := uint32((x%n+n)%n)*374761393 + uint32((y%n+n)%n)*668265263 + 1447
		v = (v ^ (v >> 13)) * 1274126177
		return float64(v^(v>>16)) / math.MaxUint32
	}
	u, v := float64(x%period)/float64(period), float64(y%period)/float64(period)
	u, v = u*u*(3-2*u), v*v*(3-2*v)
	a, b := hash(x/period, y/period), hash(x/period+1, y/period)
	c, d := hash(x/period, y/period+1), hash(x/period+1, y/period+1)
	return (a+(b-a)*u)*(1-v) + (c+(d-c)*u)*v
}

func masonry(s swatch) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, textureSize, textureSize))
	for y := range textureSize {
		for x := range textureSize {
			// Rounded quantisation limits compressed size and avoids high-frequency shimmer.
			grain := noise(x, y, 2)*.18 + noise(x, y, 8)*.22 + noise(x, y, 32)*.3 + noise(x, y, 64)*.3
			base := s.base
			relief := (grain - .5) * 32
			if s.style == "wood" || s.style == "endgrain" || s.style == "batten" {
				// Seamless grain and plank joints have no painted prop outline or brace.
				// The crate's actual geometry supplies its frame at every size and origin.
				grain = noise(x, y/8*8, 8)*.55 + noise(x, y/16*16, 32)*.45
				relief = (grain - .5) * 22
				if s.style != "batten" && y%64 < 2 {
					relief -= 18
				}
				if s.style == "endgrain" {
					relief += (noise(x, y, 4) - .5) * 8
				}
			} else if s.style == "brick" {
				row := y / 32
				bx, by := (x+(row%2)*32)%64, y%32
				joint := min(bx, 63-bx, by, 31-by)
				if joint < 2 {
					base = rgb(100, 104, 101)
					relief = (grain-.5)*18 - 15
				} else {
					// Subtle brick-to-brick variation plus chipped, bevelled edges.
					relief += (noise((x+(row%2)*32)/64*64, row*32, 32) - .5) * 18
					if joint < 4 {
						relief -= float64(4-joint) * 5
					}
				}
			} else {
				// Formwork joints, aggregate, occasional small pores and shallow tie holes.
				if x < 2 || x > 253 || y < 2 || y > 253 {
					relief -= 24
				}
				if noise(x, y, 2) < .13 {
					relief -= 17
				}
				for _, p := range []image.Point{{24, 24}, {232, 24}, {24, 232}, {232, 232}} {
					d := (x-p.X)*(x-p.X) + (y-p.Y)*(y-p.Y)
					if d < 6 {
						relief -= 25
					} else if d < 12 {
						relief -= 9
					}
				}
			}
			adjust := func(c uint8) uint8 { return uint8(max(0, min(255, int(c)+int(math.Round(relief/2))*2))) }
			dst.SetRGBA(x, y, rgb(adjust(base.R), adjust(base.G), adjust(base.B)))
		}
	}
	return dst
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func main() {
	check := flag.Bool("check", false, "compare tracked assets without writing")
	flag.Parse()
	generate(*check)
}

func packageVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read package version from %s: %w", path, err)
	}
	var metadata struct {
		Name    string `toml:"name"`
		Version string `toml:"version"`
	}
	if err := toml.Unmarshal(data, &metadata); err != nil {
		return "", fmt.Errorf("read package version from %s: %w", path, err)
	}
	if metadata.Name != "fps-kit" || len(metadata.Version) > 64 || semver.Canonical("v"+metadata.Version) != "v"+metadata.Version {
		return "", fmt.Errorf("%s must declare fps-kit with a canonical x.y.z version (optional prerelease)", path)
	}
	return metadata.Version, nil
}

func generate(check bool) {
	version, err := packageVersion(filepath.Join(root, "karty-package.toml"))
	must(err)
	var all []swatch
	for _, c := range []swatch{
		{"orange", rgb(222, 119, 48), ""}, {"red", rgb(193, 65, 74), ""},
		{"blue", rgb(55, 117, 180), ""}, {"yellow", rgb(219, 181, 66), ""},
		{"green", rgb(70, 147, 111), ""}, {"white", rgb(186, 196, 202), ""},
		{"purple", rgb(141, 91, 174), ""},
	} {
		for _, style := range []string{"grid", "panel", "checker"} {
			all = append(all, swatch{"wall-" + c.name + "-" + style, c.base, style})
		}
	}
	for _, c := range []swatch{{"gray", rgb(99, 112, 122), ""}, {"dark", rgb(47, 59, 70), ""}} {
		for _, style := range []string{"grid", "checker"} {
			all = append(all, swatch{"floor-" + c.name + "-" + style, c.base, style})
		}
		all = append(all, swatch{"ceiling-" + c.name + "-panel", c.base, "ceiling"})
	}
	all = append(all, swatch{"crate-panel", rgb(161, 135, 88), "wood"},
		swatch{"crate-lid", rgb(161, 135, 88), "endgrain"}, swatch{"trim-hazard", rgb(219, 181, 66), "hazard"})
	// Painted construction materials keep the same zone colors as the grid kit.
	for _, c := range []swatch{
		{"orange", rgb(184, 109, 64), ""}, {"red", rgb(156, 75, 71), ""},
		{"blue", rgb(80, 116, 144), ""}, {"yellow", rgb(182, 157, 87), ""},
		{"green", rgb(90, 128, 102), ""}, {"white", rgb(181, 181, 170), ""},
		{"purple", rgb(125, 102, 145), ""},
	} {
		for _, style := range []string{"brick", "concrete"} {
			all = append(all, swatch{"wall-" + c.name + "-" + style, c.base, style})
		}
	}
	all = append(all, swatch{"wall-brick", rgb(143, 88, 68), "brick"}, swatch{"wall-concrete", rgb(143, 145, 136), "concrete"})
	all = append(all, swatch{"crate-frame", rgb(108, 84, 54), "batten"})
	all = append(all, swatch{"bumper-red-metal", rgb(172, 48, 57), "metal"},
		swatch{"bumper-metal-cap", rgb(86, 94, 103), "metal"})
	manifest := fmt.Sprintf("# Generated by scripts/fps-kit/generate.go; the version is maintained here.\nformat_version = 1\nname = \"fps-kit\"\nversion = %q\nprefabs = \"prefabs.yaml\"\n", version)
	preview := image.NewRGBA(image.Rect(0, 0, 6*textureSize, ((len(all)+5)/6)*textureSize))
	for i, s := range all {
		img := texture(s, i/3+1)
		var buf bytes.Buffer
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		must(encoder.Encode(&buf, img))
		emit(filepath.Join(root, "textures", s.name+".png"), buf.Bytes(), check)
		manifest += fmt.Sprintf("\n[[textures]]\nname = %q\nsource = %q\nprofile = \"environment\"\n", s.name, "textures/"+s.name+".png")
		manifest += fmt.Sprintf("[textures.materialize]\npreset = %q\n", materialPreset(s))
		if s.style != "metal" {
			// Pigment brightness is not metalness: white paint and masonry stay dielectric.
			manifest += "metallic_scale = 0\n"
		}
		x, y := (i%6)*textureSize, (i/6)*textureSize
		draw.Draw(preview, image.Rect(x, y, x+textureSize, y+textureSize), img, image.Point{}, draw.Src)
	}
	decals := decalImages()
	decalNames := make([]string, 0, len(decals))
	for name := range decals {
		decalNames = append(decalNames, name)
	}
	sort.Strings(decalNames)
	decalPreview := image.NewRGBA(image.Rect(0, 0, 6*textureSize, ((len(decals)+5)/6)*textureSize))
	draw.Draw(decalPreview, decalPreview.Bounds(), image.NewUniform(rgb(27, 36, 47)), image.Point{}, draw.Src)
	for index, name := range decalNames {
		var encoded bytes.Buffer
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		must(encoder.Encode(&encoded, decals[name]))
		emit(filepath.Join(root, "decals", name+".png"), encoded.Bytes(), check)
		manifest += fmt.Sprintf("\n[[textures]]\nname = %q\nsource = %q\nprofile = \"sprite\"\n", name, "decals/"+name+".png")
		x, y := (index%6)*textureSize, (index/6)*textureSize
		draw.Draw(decalPreview, image.Rect(x, y, x+textureSize, y+textureSize), decals[name], image.Point{}, draw.Over)
	}
	emit(filepath.Join(root, "karty-package.toml"), []byte(manifest), check)
	var buf bytes.Buffer
	must(png.Encode(&buf, preview))
	emit(filepath.Join(root, "palette.png"), buf.Bytes(), check)
	buf.Reset()
	must(png.Encode(&buf, decalPreview))
	emit(filepath.Join(root, "decals.png"), buf.Bytes(), check)
	emit(filepath.Join(root, "prefabs.yaml"), prefabSource(), check)
	fmt.Printf("FPS kit %s: %d opaque materials, %d transparent decals, %dx%d; check=%v\n", version, len(all), len(decals), textureSize, textureSize, check)
}

func materialPreset(s swatch) string {
	switch s.style {
	case "metal", "brick", "concrete":
		return s.style
	case "wood", "endgrain", "batten":
		return "wood"
	}
	if strings.HasPrefix(s.name, "floor-") {
		return "floor"
	}
	return "plaster"
}

func prefabSource() []byte {
	var output strings.Builder
	output.WriteString("# Metres. Static crate origins sit on the floor; dynamic origins are centered.\n# Frames are geometry, so seamless wood stays aligned across sizes and rotation.\n# Each moving crate uses one box collider attached by the game.\nprefabs:\n")
	for _, dynamic := range []bool{false, true} {
		for _, box := range []struct {
			name    string
			w, d, h float64
		}{
			{"small", .5, .5, .5}, {"medium", 1, 1, 1}, {"large", 2, 2, 2}, {"long", 2, 1, 1}, {"tall", 1, 1, 2},
		} {
			name := box.name
			bottom := float64(0)
			if dynamic {
				name += "-dynamic"
				bottom = -box.h / 2
			}
			top, thickness, inset := bottom+box.h, min(box.w, box.d, box.h)*.07, min(box.w, box.d, box.h)*.012
			fmt.Fprintf(&output, "  - id: crate-%s\n    solids:\n", name)
			add := func(name string, x0, x1, y0, y1, z0, z1 float64, side, cap string) {
				writeBox(&output, name, x0, x1, y0, y1, z0, z1, side, cap, !dynamic)
			}
			add("panel", -box.w/2+inset, box.w/2-inset, -box.d/2+inset, box.d/2-inset, bottom+thickness, top-thickness, "crate-panel", "crate-lid")
			add("lower-frame", -box.w/2, box.w/2, -box.d/2, box.d/2, bottom, bottom+thickness, "crate-frame", "crate-lid")
			add("upper-frame", -box.w/2, box.w/2, -box.d/2, box.d/2, top-thickness, top, "crate-frame", "crate-lid")
			for _, post := range []struct {
				name           string
				x0, x1, y0, y1 float64
			}{
				{"sw", -box.w / 2, -box.w/2 + thickness, -box.d / 2, -box.d/2 + thickness},
				{"se", box.w/2 - thickness, box.w / 2, -box.d / 2, -box.d/2 + thickness},
				{"ne", box.w/2 - thickness, box.w / 2, box.d/2 - thickness, box.d / 2},
				{"nw", -box.w / 2, -box.w/2 + thickness, box.d/2 - thickness, box.d / 2},
			} {
				add("post-"+post.name, post.x0, post.x1, post.y0, post.y1, bottom+thickness, top-thickness, "crate-frame", "crate-frame")
			}
		}
		if !dynamic {
			extras, err := os.ReadFile("scripts/fps-kit/props.yaml")
			must(err)
			output.WriteString(strings.SplitN(string(extras), "prefabs:\n", 2)[1])
			for _, bumper := range []struct {
				name   string
				height float64
			}{{"low", .75}, {"cover", 1.25}, {"tall", 2}} {
				fmt.Fprintf(&output, "  - id: bumper-%s\n    solids:\n      - id: body\n", bumper.name)
				output.WriteString("        footprint: [{ x: 0.85, y: 0 }, { x: 0.601, y: 0.601 }, { x: 0, y: 0.85 }, { x: -0.601, y: 0.601 }, { x: -0.85, y: 0 }, { x: -0.601, y: -0.601 }, { x: 0, y: -0.85 }, { x: 0.601, y: -0.601 }]\n")
				fmt.Fprintf(&output, "        bottom: { c: 0 }\n        top: { c: %g }\n", bumper.height)
				output.WriteString("        side_material: bumper-red-metal\n        top_material: bumper-metal-cap\n        bottom_material: bumper-metal-cap\n        collision: true\n")
			}
		}
	}
	return []byte(output.String())
}

func writeBox(out *strings.Builder, name string, x0, x1, y0, y1, z0, z1 float64, side, cap string, collision bool) {
	number := func(n float64) string {
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", n), "0"), ".")
	}
	fmt.Fprintf(out, "      - id: %s\n        footprint: [{ x: %s, y: %s }, { x: %s, y: %s }, { x: %s, y: %s }, { x: %s, y: %s }]\n",
		name, number(x0), number(y0), number(x1), number(y0), number(x1), number(y1), number(x0), number(y1))
	fmt.Fprintf(out, "        bottom: { c: %s }\n        top: { c: %s }\n        side_material: %s\n        top_material: %s\n        bottom_material: %s\n", number(z0), number(z1), side, cap, cap)
	if collision {
		out.WriteString("        collision: true\n")
	}
}

func emit(path string, data []byte, check bool) {
	if check {
		actual, err := os.ReadFile(path)
		must(err)
		if !bytes.Equal(actual, data) {
			panic("stale generated asset: " + path)
		}
		return
	}
	must(os.MkdirAll(filepath.Dir(path), 0755))
	must(os.WriteFile(path, data, 0644))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
