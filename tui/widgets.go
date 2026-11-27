package tui

import (
	"github.com/Owloops/termui/v3/widgets"

	uw "github.com/Owloops/updo/widgets"
)

type DetailsManager struct {
	TimingBreakdownWidget *uw.TimingBreakdown
	LogsWidget            *widgets.Tree
}

func NewDetailsManager() *DetailsManager {
	return &DetailsManager{}
}

func (m *DetailsManager) InitializeWidgets() {
	m.TimingBreakdownWidget = uw.NewTimingBreakdown()
	m.LogsWidget = widgets.NewTree()
}
