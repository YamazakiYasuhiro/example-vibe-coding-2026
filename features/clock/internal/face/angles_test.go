package face

import (
	"math"
	"testing"
	"time"
)

func TestHandAngles(t *testing.T) {
	loc := time.Local
	tests := []struct {
		name      string
		t         time.Time
		hourDeg   float64
		minuteDeg float64
		secondDeg float64
	}{
		{
			name:      "midnight",
			t:         time.Date(2026, 9, 18, 0, 0, 0, 0, loc),
			hourDeg:   0,
			minuteDeg: 0,
			secondDeg: 0,
		},
		{
			name:      "six_hours",
			t:         time.Date(2026, 9, 18, 6, 0, 0, 0, loc),
			hourDeg:   180,
			minuteDeg: 0,
			secondDeg: 0,
		},
		{
			name:      "thirty_sec",
			t:         time.Date(2026, 9, 18, 0, 0, 30, 0, loc),
			hourDeg:   0,
			minuteDeg: 3, // min*6 + sec*0.1
			secondDeg: 180,
		},
		{
			name:      "thirty_min",
			t:         time.Date(2026, 9, 18, 0, 30, 0, 0, loc),
			hourDeg:   15,
			minuteDeg: 180,
			secondDeg: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HandAngles(tt.t)
			assertNear(t, "Hour", got.Hour, tt.hourDeg)
			assertNear(t, "Minute", got.Minute, tt.minuteDeg)
			assertNear(t, "Second", got.Second, tt.secondDeg)
		})
	}
}

func TestHandAngles_SecondTicks(t *testing.T) {
	t0 := time.Date(2026, 9, 18, 12, 0, 0, 0, time.Local)
	t1 := t0.Add(time.Second)
	a0 := HandAngles(t0)
	a1 := HandAngles(t1)
	if a0.Second == a1.Second {
		t.Fatalf("second angle did not change after 1s: %v", a0.Second)
	}
}

func assertNear(t *testing.T, label string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.01 {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
}
