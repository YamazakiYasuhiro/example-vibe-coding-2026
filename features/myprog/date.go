package main

import (
	"fmt"
	"time"
)

// FormatToday formats t according to named format.
// Supported format names: "iso" (default), "slash", "jp".
// Empty format is treated as "iso".
func FormatToday(t time.Time, format string) (string, error) {
	if format == "" {
		format = "iso"
	}

	switch format {
	case "iso":
		return t.Format("2006-01-02"), nil
	case "slash":
		return t.Format("2006/01/02"), nil
	case "jp":
		return t.Format("2006年01月02日"), nil
	default:
		return "", fmt.Errorf("unsupported format: %q (want iso, slash, or jp)", format)
	}
}
