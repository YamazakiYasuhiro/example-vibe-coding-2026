package face

import (
	"image"
	"testing"
	"time"
)

func TestRender_SizeAndTransparency(t *testing.T) {
	const size = 200
	ts := time.Date(2026, 9, 18, 0, 0, 0, 0, time.Local)
	img := Render(size, ts)

	if img.Bounds() != image.Rect(0, 0, size, size) {
		t.Fatalf("bounds = %v, want %v", img.Bounds(), image.Rect(0, 0, size, size))
	}

	corners := []image.Point{
		{0, 0},
		{size - 1, 0},
		{0, size - 1},
		{size - 1, size - 1},
	}
	for _, p := range corners {
		if _, _, _, a := img.At(p.X, p.Y).RGBA(); a != 0 {
			t.Fatalf("corner %v alpha = %d, want 0", p, a)
		}
	}

	cx, cy := size/2, size/2
	if _, _, _, a := img.At(cx, cy).RGBA(); a == 0 {
		t.Fatalf("center (%d,%d) alpha = 0, want non-zero", cx, cy)
	}

	// A point clearly outside the circle but inside the square.
	outside := image.Point{X: 0, Y: size / 2}
	if _, _, _, a := img.At(outside.X, outside.Y).RGBA(); a != 0 {
		t.Fatalf("outside point %v alpha = %d, want 0", outside, a)
	}
}
