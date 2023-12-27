package utils

import (
	"fmt"
	"time"
)

func BoolToFloat64(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

func FormatDurationMillisecond(d time.Duration) string {
	return fmt.Sprintf("%d ms", d.Milliseconds())
}

func FormatDurationMinute(d time.Duration) string {
	return d.Round(time.Second).String()
}
