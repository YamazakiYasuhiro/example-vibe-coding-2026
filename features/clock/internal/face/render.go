package face

import (
	"image"
	"image/color"
	"math"
	"time"
)

// DefaultSize is the default clock face edge length in pixels.
const DefaultSize = 200

// Render draws a circular analog clock. Pixels outside the circle have alpha 0.
func Render(size int, t time.Time) *image.RGBA {
	if size < 2 {
		size = DefaultSize
	}

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	cx := float64(size-1) / 2
	cy := float64(size-1) / 2
	r := float64(size) * 0.48

	faceColor := color.RGBA{R: 240, G: 240, B: 240, A: 255}
	rimColor := color.RGBA{R: 40, G: 40, B: 40, A: 255}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			dist := math.Hypot(dx, dy)
			if dist <= r {
				img.SetRGBA(x, y, faceColor)
			}
			// Thin rim near the circumference.
			if math.Abs(dist-r) <= 1.2 && dist <= r+0.5 {
				img.SetRGBA(x, y, rimColor)
			}
		}
	}

	a := HandAngles(t)
	drawHand(img, cx, cy, a.Hour, r*0.55, 3, color.RGBA{R: 20, G: 20, B: 20, A: 255})
	drawHand(img, cx, cy, a.Minute, r*0.75, 2, color.RGBA{R: 30, G: 30, B: 30, A: 255})
	drawHand(img, cx, cy, a.Second, r*0.90, 1, color.RGBA{R: 200, G: 40, B: 40, A: 255})

	// Center hub.
	for y := int(cy) - 3; y <= int(cy)+3; y++ {
		for x := int(cx) - 3; x <= int(cx)+3; x++ {
			if x < 0 || y < 0 || x >= size || y >= size {
				continue
			}
			if math.Hypot(float64(x)-cx, float64(y)-cy) <= 3 {
				img.SetRGBA(x, y, rimColor)
			}
		}
	}

	return img
}

func drawHand(img *image.RGBA, cx, cy, deg, length float64, thickness int, c color.RGBA) {
	rad := deg * math.Pi / 180
	x1 := cx + length*math.Sin(rad)
	y1 := cy - length*math.Cos(rad)
	drawLine(img, cx, cy, x1, y1, thickness, c)
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 float64, thickness int, c color.RGBA) {
	steps := int(math.Hypot(x1-x0, y1-y0)) + 1
	if steps < 1 {
		steps = 1
	}
	b := img.Bounds()
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := x0 + (x1-x0)*t
		y := y0 + (y1-y0)*t
		for dy := -thickness; dy <= thickness; dy++ {
			for dx := -thickness; dx <= thickness; dx++ {
				px := int(math.Round(x)) + dx
				py := int(math.Round(y)) + dy
				if px < b.Min.X || py < b.Min.Y || px >= b.Max.X || py >= b.Max.Y {
					continue
				}
				img.SetRGBA(px, py, c)
			}
		}
	}
}
