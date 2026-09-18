package face

import "time"

// Angles holds hand angles in degrees from 12 o'clock, clockwise.
type Angles struct {
	Hour   float64
	Minute float64
	Second float64
}

// HandAngles returns analog hand angles for t using wall-clock fields.
func HandAngles(t time.Time) Angles {
	sec := float64(t.Second())
	min := float64(t.Minute())
	hour := float64(t.Hour() % 12)

	return Angles{
		Second: sec * 6,
		Minute: min*6 + sec*0.1,
		Hour:   hour*30 + min*0.5,
	}
}
