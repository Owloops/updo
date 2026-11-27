package tui

import (
	"math"
	"os"
	"strconv"
	"strings"

	ui "github.com/Owloops/termui/v3"
	"github.com/gdamore/tcell/v2"
)

const (
	_accentEnv     = "UPDO_ACCENT"
	_gradientSteps = 48
	_rgbColorBits  = 64

	_colorMuted       ui.Color = 245
	_colorFaint       ui.Color = 240
	_colorSelectionBg ui.Color = 237
	_colorSelectionFg ui.Color = 255
	_colorBrandOrange ui.Color = 208
)

var (
	_orangeGradient = []gradientStop{
		{at: 0, red: 0xff, green: 0x87, blue: 0x00},
		{at: 1, red: 0xff, green: 0xd0, blue: 0x55},
	}

	_accents = map[string]ui.Color{
		"orange":  _colorBrandOrange,
		"cyan":    ui.ColorCyan,
		"blue":    ui.ColorBlue,
		"magenta": ui.ColorMagenta,
		"green":   ui.ColorGreen,
	}
)

type gradientStop struct {
	at               float64
	red, green, blue int32
}

type palette struct {
	text      ui.Style
	strong    ui.Style
	muted     ui.Style
	faint     ui.Style
	accent    ui.Style
	accentBar ui.Style
	chart     []ui.Style
	selected  ui.Style
	good      ui.Style
	warn      ui.Style
	bad       ui.Style
}

func newPalette(accentColor ui.Color) palette {
	return palette{
		text:      ui.NewStyle(ui.ColorClear),
		strong:    ui.NewStyle(ui.ColorClear, ui.ColorClear, ui.ModifierBold),
		muted:     ui.NewStyle(_colorMuted),
		faint:     ui.NewStyle(_colorFaint),
		accent:    ui.NewStyle(accentColor, ui.ColorClear, ui.ModifierBold),
		accentBar: ui.NewStyle(ui.ColorBlack, accentColor),
		chart:     chartGradient(accentColor),
		selected:  ui.NewStyle(_colorSelectionFg, _colorSelectionBg, ui.ModifierBold),
		good:      ui.NewStyle(ui.ColorGreen, ui.ColorClear, ui.ModifierBold),
		warn:      ui.NewStyle(ui.ColorYellow, ui.ColorClear, ui.ModifierBold),
		bad:       ui.NewStyle(ui.ColorRed, ui.ColorClear, ui.ModifierBold),
	}
}

// Bold black renders as bright black in many terminals, which washes out on a coloured pill.
func pill(style ui.Style) ui.Style {
	return ui.NewStyle(ui.ColorBlack, style.Fg)
}

func chartGradient(accentColor ui.Color) []ui.Style {
	if accentColor != _colorBrandOrange || strconv.IntSize < _rgbColorBits {
		return []ui.Style{ui.NewStyle(accentColor)}
	}

	gradient := make([]ui.Style, 0, _gradientSteps)
	for step := range _gradientSteps {
		position := float64(step) / float64(_gradientSteps-1)
		gradient = append(gradient, ui.NewStyle(gradientColor(_orangeGradient, position)))
	}
	return gradient
}

func gradientColor(stops []gradientStop, position float64) ui.Color {
	low, high := stops[0], stops[len(stops)-1]
	for i := 1; i < len(stops); i++ {
		if position <= stops[i].at {
			low, high = stops[i-1], stops[i]
			break
		}
	}

	ratio := (position - low.at) / (high.at - low.at)
	mix := func(from, to int32) int32 {
		return from + int32(math.Round(float64(to-from)*ratio))
	}
	return rgbColor(mix(low.red, high.red), mix(low.green, high.green), mix(low.blue, high.blue))
}

func rgbColor(red, green, blue int32) ui.Color {
	rgb := tcell.NewRGBColor(red, green, blue)
	return ui.Color(rgb) //nolint:gosec // RGB colours carry flag bits above 32, which chartGradient checks fit in int
}

func accentFromEnv() ui.Color {
	if color, ok := _accents[strings.ToLower(strings.TrimSpace(os.Getenv(_accentEnv)))]; ok {
		return color
	}
	return _colorBrandOrange
}
