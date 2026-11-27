package tui

import (
	"os"
	"strings"

	ui "github.com/Owloops/termui/v3"
)

const (
	_accentEnv = "UPDO_ACCENT"

	_colorMuted       ui.Color = 245
	_colorFaint       ui.Color = 240
	_colorSelectionBg ui.Color = 237
	_colorSelectionFg ui.Color = 255
	_colorBrandOrange ui.Color = 208
)

var _accents = map[string]ui.Color{
	"orange":  _colorBrandOrange,
	"cyan":    ui.ColorCyan,
	"blue":    ui.ColorBlue,
	"magenta": ui.ColorMagenta,
	"green":   ui.ColorGreen,
}

type palette struct {
	text      ui.Style
	strong    ui.Style
	muted     ui.Style
	faint     ui.Style
	accent    ui.Style
	accentBar ui.Style
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

func accentFromEnv() ui.Color {
	if color, ok := _accents[strings.ToLower(strings.TrimSpace(os.Getenv(_accentEnv)))]; ok {
		return color
	}
	return _colorBrandOrange
}
