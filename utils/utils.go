package utils

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gizak/termui/v3/widgets"
)

func BoolToFloat64(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

func FormatBool(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func FormatDurationMillisecond(d time.Duration) string {
	return fmt.Sprintf("%d ms", d.Milliseconds())
}

func FormatDurationMinute(d time.Duration) string {
	return d.Round(time.Second).String()
}

func FormatInt(i int) string {
	if i == -1 {
		return "Unknown"
	}
	return strconv.Itoa(i)
}

func UpdatePlot(plot *widgets.Plot, dataPoint float64) {
	if len(plot.Data) == 0 || len(plot.Data[0]) == 0 {
		plot.Data = [][]float64{{dataPoint}}
	} else {
		plot.Data[0] = append(plot.Data[0], dataPoint)
		if len(plot.Data[0]) > 100 {
			plot.Data[0] = plot.Data[0][1:]
		}
	}
}
