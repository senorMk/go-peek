package capture

import (
	"errors"
	"fmt"
	"math"
)

// Rect is in global desktop points with the Core Graphics top-left origin.
// Negative origins are valid for displays above or to the left of the primary.
type Rect struct{ X, Y, Width, Height float64 }

func (r Rect) argument() (string, error) {
	for _, n := range []float64{r.X, r.Y, r.Width, r.Height} {
		if math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > 1_000_000 {
			return "", errors.New("Invalid capture region")
		}
	}
	if r.Width < 2 || r.Height < 2 {
		return "", errors.New("Capture region must be at least 2 × 2 points")
	}
	x, y := math.Floor(r.X), math.Floor(r.Y)
	w, h := math.Ceil(r.X+r.Width)-x, math.Ceil(r.Y+r.Height)-y
	return fmt.Sprintf("%.0f,%.0f,%.0f,%.0f", x, y, w, h), nil
}
