//go:build !windows

package winui

import (
	"fmt"
	"time"
)

// Options configures the desktop clock window.
type Options struct {
	Size      int
	QuitAfter time.Duration
}

// Run reports that the GUI requires Windows.
func Run(opts Options) error {
	return fmt.Errorf("clock GUI requires Windows")
}
