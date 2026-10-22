//go:build ignore

package main

import (
	"image"
	"image/color"
	"math"
)

// decalImages supplies original, self-contained blockout markers. Each marker is
// 256 px with transparent margins; opaque ink and a two-tone contour stay readable
// over both the light and dark kit materials. No fonts or external art are used.
func decalImages() map[string]*image.RGBA {
	out := make(map[string]*image.RGBA, 36)
	paper := color.RGBA{244, 248, 239, 255}
	cyan := color.RGBA{70, 218, 226, 255}
	gold := color.RGBA{255, 202, 73, 255}
	red := color.RGBA{255, 108, 108, 255}
	green := color.RGBA{113, 232, 157, 255}
	ink := color.RGBA{27, 39, 56, 255}
	add := func(name string, paint func(*image.RGBA)) {
		dst := image.NewRGBA(image.Rect(0, 0, 256, 256))
		paint(dst)
		out["decal-"+name] = decalContour(dst)
	}
	add("shape-cross", func(d *image.RGBA) {
		decalPoly(d, cyan, 96, 37, 160, 37, 160, 96, 219, 96, 219, 160, 160, 160, 160, 219, 96, 219, 96, 160, 37, 160, 37, 96, 96, 96)
	})
	add("shape-triangle", func(d *image.RGBA) { decalPoly(d, gold, 128, 33, 225, 212, 31, 212) })
	add("shape-circle", func(d *image.RGBA) { decalEllipse(d, 128, 128, 89, 89, cyan) })
	add("shape-square", func(d *image.RGBA) { decalBox(d, 43, 43, 213, 213, cyan) })
	for n := 0; n <= 9; n++ {
		label := string(rune('0' + n))
		add("number-"+label, func(d *image.RGBA) { decalWord(d, label, 22, 128, 128, paper) })
	}
	for _, word := range []string{"nice", "great", "ok", "yes", "no", "wrong"} {
		add("feedback-"+word, func(d *image.RGBA) {
			accent := green
			if word == "great" {
				accent = cyan
			}
			if word == "ok" {
				accent = gold
			}
			if word == "no" || word == "wrong" {
				accent = red
			}
			decalPoly(d, ink, 41, 64, 215, 64, 231, 80, 231, 176, 215, 192, 41, 192, 25, 176, 25, 80)
			decalBox(d, 49, 79, 207, 85, accent)
			decalBox(d, 49, 171, 207, 177, accent)
			scale := 192 / (len(word)*6 - 1)
			if scale > 12 {
				scale = 12
			}
			decalWord(d, word, scale, 128, 128, paper)
		})
	}
	add("icon-question", func(d *image.RGBA) {
		decalPoly(d, gold, 65, 83, 65, 58, 89, 34, 163, 34, 190, 61, 190, 101, 145, 137, 145, 163, 111, 163, 111, 122, 155, 87, 155, 72, 143, 60, 107, 60, 96, 71, 96, 83)
		decalBox(d, 111, 190, 145, 223, gold)
	})
	add("icon-exclamation", func(d *image.RGBA) {
		decalPoly(d, gold, 104, 33, 152, 33, 143, 164, 113, 164)
		decalBox(d, 109, 190, 147, 225, gold)
	})
	add("icon-eye", func(d *image.RGBA) {
		decalPoly(d, paper, 26, 128, 57, 91, 96, 69, 160, 69, 199, 91, 230, 128, 199, 165, 160, 187, 96, 187, 57, 165)
		decalEllipse(d, 128, 128, 46, 46, cyan)
		decalEllipse(d, 128, 128, 25, 25, ink)
		decalEllipse(d, 117, 116, 8, 8, paper)
	})
	add("icon-bomb", func(d *image.RGBA) {
		decalLine(d, 149, 81, 164, 51, 13, gold)
		decalLine(d, 164, 51, 190, 49, 13, gold)
		decalLine(d, 191, 32, 191, 20, 7, gold)
		decalLine(d, 208, 45, 222, 40, 7, gold)
		decalLine(d, 208, 63, 220, 73, 7, gold)
		decalPoly(d, paper, 137, 59, 171, 78, 158, 101, 124, 82)
		decalEllipse(d, 118, 151, 76, 76, ink)
		decalLine(d, 73, 130, 82, 112, 9, paper)
		decalLine(d, 82, 112, 99, 102, 9, paper)
		decalEllipse(d, 146, 180, 8, 8, gold)
	})
	add("icon-explosion", func(d *image.RGBA) {
		decalPoly(d, gold, 128, 27, 145, 77, 192, 45, 181, 96, 230, 104, 190, 138, 222, 183, 169, 179, 157, 229, 123, 191, 80, 220, 82, 169, 29, 161, 65, 123, 35, 80, 89, 82, 100, 32, 118, 77)
		decalPoly(d, red, 127, 84, 142, 113, 177, 104, 160, 137, 177, 161, 142, 157, 124, 181, 112, 151, 79, 151, 100, 126, 88, 102, 118, 108)
		decalPoly(d, paper, 127, 112, 142, 134, 124, 150, 111, 132)
	})
	add("icon-rocket", func(d *image.RGBA) {
		decalPoly(d, gold, 82, 166, 109, 175, 89, 212, 45, 224, 57, 182)
		decalPoly(d, red, 89, 123, 49, 136, 39, 185, 91, 168)
		decalPoly(d, red, 132, 167, 119, 207, 70, 217, 87, 165)
		decalPoly(d, paper, 77, 143, 129, 76, 168, 43, 220, 33, 210, 85, 177, 124, 110, 176)
		decalPoly(d, red, 168, 43, 220, 33, 210, 85)
		decalEllipse(d, 151, 102, 24, 24, ink)
		decalEllipse(d, 151, 102, 14, 14, cyan)
		decalLine(d, 89, 164, 117, 136, 8, ink)
	})
	add("icon-star", func(d *image.RGBA) {
		decalStar(d, 128, 130, 100, 44, 5, gold)
	})
	add("icon-heart", func(d *image.RGBA) {
		decalEllipse(d, 85, 94, 51, 52, red)
		decalEllipse(d, 171, 94, 51, 52, red)
		decalPoly(d, red, 38, 116, 218, 116, 202, 151, 128, 221, 54, 151)
		decalLine(d, 62, 87, 70, 72, 8, paper)
	})
	add("icon-flag", func(d *image.RGBA) {
		decalBox(d, 54, 31, 66, 225, paper)
		decalPoly(d, cyan, 73, 38, 215, 38, 181, 82, 215, 126, 73, 126)
		decalBox(d, 38, 218, 91, 230, paper)
	})
	add("icon-target", func(d *image.RGBA) {
		decalEllipse(d, 128, 128, 84, 84, red)
		decalEllipse(d, 128, 128, 61, 61, paper)
		decalEllipse(d, 128, 128, 39, 39, red)
		decalEllipse(d, 128, 128, 16, 16, paper)
		decalLine(d, 128, 21, 128, 66, 8, ink)
		decalLine(d, 128, 190, 128, 235, 8, ink)
		decalLine(d, 21, 128, 66, 128, 8, ink)
		decalLine(d, 190, 128, 235, 128, 8, ink)
	})
	add("icon-arrow", func(d *image.RGBA) {
		decalPoly(d, cyan, 29, 101, 139, 101, 139, 46, 230, 128, 139, 210, 139, 155, 29, 155)
	})
	add("icon-key", func(d *image.RGBA) {
		decalEllipse(d, 82, 90, 52, 52, gold)
		decalPoly(d, gold, 94, 116, 116, 94, 223, 201, 202, 223, 180, 201, 165, 216, 145, 196, 160, 181)
		decalEllipse(d, 82, 90, 22, 22, ink)
	})
	add("icon-lock", func(d *image.RGBA) {
		decalEllipse(d, 128, 86, 55, 57, paper)
		decalEllipse(d, 128, 86, 30, 33, ink)
		decalPoly(d, gold, 58, 105, 198, 105, 212, 119, 212, 211, 198, 225, 58, 225, 44, 211, 44, 119)
		decalEllipse(d, 128, 153, 16, 16, ink)
		decalPoly(d, ink, 120, 158, 136, 158, 141, 190, 115, 190)
	})
	add("icon-skull", func(d *image.RGBA) {
		decalEllipse(d, 128, 104, 77, 72, paper)
		decalPoly(d, paper, 72, 129, 184, 129, 177, 184, 159, 184, 159, 216, 97, 216, 97, 184, 79, 184)
		decalPoly(d, ink, 75, 102, 109, 109, 105, 137, 78, 137)
		decalPoly(d, ink, 181, 102, 147, 109, 151, 137, 178, 137)
		decalPoly(d, ink, 128, 143, 142, 167, 114, 167)
		decalBox(d, 114, 190, 120, 216, ink)
		decalBox(d, 136, 190, 142, 216, ink)
	})
	add("icon-waypoint", func(d *image.RGBA) {
		decalEllipse(d, 128, 94, 65, 65, cyan)
		decalPoly(d, cyan, 68, 118, 188, 118, 128, 229)
		decalEllipse(d, 128, 94, 29, 29, ink)
		decalEllipse(d, 128, 94, 15, 15, paper)
	})
	add("icon-check", func(d *image.RGBA) {
		decalPoly(d, green, 35, 125, 64, 96, 105, 137, 193, 49, 222, 78, 105, 195)
	})
	return out
}

// decalContour expands only occupied pixels, so holes and unused margins remain
// transparent. Two opaque bands avoid premultiplication or alpha-fringe issues.
func decalContour(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			if c := src.RGBAAt(x, y); c.A != 0 {
				dst.SetRGBA(x, y, c)
				continue
			}
			nearest := 50
			for dy := -6; dy <= 6; dy++ {
				for dx := -6; dx <= 6; dx++ {
					distance := dx*dx + dy*dy
					if distance <= 36 && distance < nearest && src.RGBAAt(x+dx, y+dy).A != 0 {
						nearest = distance
					}
				}
			}
			if nearest <= 9 {
				dst.SetRGBA(x, y, color.RGBA{244, 248, 239, 255})
			} else if nearest <= 36 {
				dst.SetRGBA(x, y, color.RGBA{19, 28, 42, 255})
			}
		}
	}
	return dst
}

func decalBox(d *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			d.SetRGBA(x, y, c)
		}
	}
}

func decalEllipse(d *image.RGBA, cx, cy, rx, ry int, c color.RGBA) {
	for y := cy - ry; y <= cy+ry; y++ {
		for x := cx - rx; x <= cx+rx; x++ {
			dx, dy := float64(x-cx)/float64(rx), float64(y-cy)/float64(ry)
			if dx*dx+dy*dy <= 1 {
				d.SetRGBA(x, y, c)
			}
		}
	}
}

func decalPoly(d *image.RGBA, c color.RGBA, points ...int) {
	minX, minY, maxX, maxY := 256, 256, 0, 0
	for i := 0; i < len(points); i += 2 {
		minX = min(minX, points[i])
		maxX = max(maxX, points[i])
		minY = min(minY, points[i+1])
		maxY = max(maxY, points[i+1])
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			inside := false
			for i, j := 0, len(points)-2; i < len(points); j, i = i, i+2 {
				xi, yi, xj, yj := float64(points[i]), float64(points[i+1]), float64(points[j]), float64(points[j+1])
				if (yi > float64(y)) != (yj > float64(y)) && float64(x) < (xj-xi)*(float64(y)-yi)/(yj-yi)+xi {
					inside = !inside
				}
			}
			if inside {
				d.SetRGBA(x, y, c)
			}
		}
	}
}

func decalLine(d *image.RGBA, x0, y0, x1, y1, width int, c color.RGBA) {
	length := math.Hypot(float64(x1-x0), float64(y1-y0))
	steps := max(1, int(length*2))
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		decalEllipse(d, int(math.Round(float64(x0)+float64(x1-x0)*t)), int(math.Round(float64(y0)+float64(y1-y0)*t)), width/2, width/2, c)
	}
}

func decalStar(d *image.RGBA, cx, cy, outer, inner, count int, c color.RGBA) {
	points := make([]int, 0, count*4)
	for i := 0; i < count*2; i++ {
		radius := outer
		if i%2 != 0 {
			radius = inner
		}
		a := float64(i)*math.Pi/float64(count) - math.Pi/2
		points = append(points, cx+int(math.Round(math.Cos(a)*float64(radius))), cy+int(math.Round(math.Sin(a)*float64(radius))))
	}
	decalPoly(d, c, points...)
}

// Five-by-seven glyphs keep small exported labels legible and reproducible.
var decalGlyphs = map[rune][7]string{
	'0': {"01110", "11011", "11011", "11011", "11011", "11011", "01110"},
	'1': {"00110", "01110", "00110", "00110", "00110", "00110", "01111"},
	'2': {"01110", "11011", "00011", "00110", "01100", "11000", "11111"},
	'3': {"11110", "00011", "00011", "01110", "00011", "00011", "11110"},
	'4': {"11011", "11011", "11011", "11111", "00011", "00011", "00011"},
	'5': {"11111", "11000", "11000", "11110", "00011", "00011", "11110"},
	'6': {"01111", "11000", "11000", "11110", "11011", "11011", "01110"},
	'7': {"11111", "00011", "00011", "00110", "00110", "01100", "01100"},
	'8': {"01110", "11011", "11011", "01110", "11011", "11011", "01110"},
	'9': {"01110", "11011", "11011", "01111", "00011", "00011", "11110"},
	'n': {"11001", "11001", "11101", "11111", "11011", "11001", "11001"},
	'i': {"11111", "00100", "00100", "00100", "00100", "00100", "11111"},
	'c': {"01111", "11000", "11000", "11000", "11000", "11000", "01111"},
	'e': {"11111", "11000", "11000", "11110", "11000", "11000", "11111"},
	'g': {"01111", "11000", "11000", "11011", "11011", "11011", "01111"},
	'r': {"11110", "11011", "11011", "11110", "11100", "11010", "11011"},
	'a': {"01110", "11011", "11011", "11111", "11011", "11011", "11011"},
	't': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'o': {"01110", "11011", "11011", "11011", "11011", "11011", "01110"},
	'k': {"11011", "11011", "11110", "11100", "11110", "11011", "11011"},
	'y': {"11011", "11011", "11011", "01110", "00100", "00100", "00100"},
	's': {"01111", "11000", "11000", "01110", "00011", "00011", "11110"},
	'w': {"10001", "10001", "10001", "10101", "10101", "11111", "01110"},
}

func decalWord(d *image.RGBA, word string, scale, cx, cy int, c color.RGBA) {
	x0 := cx - (len(word)*6-1)*scale/2
	y0 := cy - 7*scale/2
	for i, r := range word {
		for y, row := range decalGlyphs[r] {
			for x, bit := range row {
				if bit == '1' {
					decalBox(d, x0+(i*6+x)*scale, y0+y*scale, x0+(i*6+x+1)*scale, y0+(y+1)*scale, c)
				}
			}
		}
	}
}
