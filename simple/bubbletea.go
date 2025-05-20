package simple

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
	"github.com/Owloops/updo/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	config        Config
	monitor       *stats.Monitor
	checkResult   net.WebsiteCheckResult
	outputLines   []string
	done          bool
	checks        int
	lastCheck     time.Time
	quitRequested bool
	alertSent     bool
}

func initialModel(config Config) Model {
	monitor, _ := stats.NewMonitor()
	return Model{
		config:        config,
		monitor:       monitor,
		outputLines:   make([]string, 0, 100),
		done:          false,
		lastCheck:     time.Time{},
		quitRequested: false,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		createCheckWebsiteCmd(m.config),
		tickCmd(m.config.RefreshInterval),
	)
}

type tickMsg time.Time

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "q" || keyMsg.String() == "ctrl+c" {
			m.done = true
			return m, tea.Quit
		}
	}

	switch msg := msg.(type) {
	case tickMsg:
		if m.quitRequested {
			m.done = true
			return m, tea.Quit
		}

		m.lastCheck = time.Time(msg)
		return m, tea.Batch(
			tickCmd(m.config.RefreshInterval),
			createCheckWebsiteCmd(m.config),
		)

	case net.WebsiteCheckResult:
		m.checkResult = msg
		m.monitor.AddResult(msg)
		m.checks++

		stats := m.monitor.GetStats()
		pingLine := formatPingLine(msg, stats, m.checks-1, m.config)
		m.outputLines = append(m.outputLines, pingLine)

		if len(m.outputLines) > 500 {
			m.outputLines = m.outputLines[len(m.outputLines)-400:]
		}

		if m.config.Count > 0 && m.checks >= m.config.Count {
			m.done = true
			return m, tea.Quit
		}

		if m.config.ReceiveAlert {
			utils.HandleAlerts(msg.IsUp, &m.alertSent)
		}

		if m.quitRequested {
			m.done = true
			return m, tea.Quit
		}

		return m, nil
	}

	return m, cmd
}

func formatPingLine(result net.WebsiteCheckResult, stats stats.Stats, seq int, config Config) string {
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F5E0DC"))
	highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FAB387"))

	var statusIndicator string
	if result.IsUp {
		statusIndicator = successStyle.Render("●")
	} else {
		statusIndicator = errorStyle.Render("●")
	}

	ipInfo := ""
	if result.ResolvedIP != "" {
		ipInfo = " from " + valueStyle.Render(result.ResolvedIP)
	}

	var statusText string
	if result.StatusCode > 0 {
		statusText = fmt.Sprintf("status=%s", valueStyle.Render(fmt.Sprintf("%d", result.StatusCode)))
	} else {
		statusText = errorStyle.Render("failed")
	}

	assertInfo := ""
	if result.AssertText != "" && !result.AssertionPassed {
		assertInfo = " " + errorStyle.Render("(assertion failed)")
	}

	return fmt.Sprintf("%s Response%s: seq=%s time=%s %s%s uptime=%s",
		statusIndicator,
		ipInfo,
		valueStyle.Render(fmt.Sprintf("%d", seq)),
		highlightStyle.Render(fmt.Sprintf("%dms", result.ResponseTime.Milliseconds())),
		statusText,
		assertInfo,
		valueStyle.Render(fmt.Sprintf("%.1f%%", stats.UptimePercent)),
	)
}

func (m Model) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#89B4FA"))
	urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1")).Bold(true)
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4"))
	sslStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F9E2AF"))

	headerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#CBA6F7")).
		Padding(0, 1).
		MarginBottom(1).
		Align(lipgloss.Left).
		Width(40)

	var sb strings.Builder

	header := titleStyle.Render("🐤 UPDO") + " " +
		urlStyle.Render(m.config.URL)

	headerContent := header
	if strings.HasPrefix(m.config.URL, "https://") && !m.config.SkipSSL {
		sslDaysRemaining := net.GetSSLCertExpiry(m.config.URL)
		if sslDaysRemaining > 0 {
			headerContent = header + "\n" + sslStyle.Render(fmt.Sprintf("SSL certificate expires in %d days", sslDaysRemaining))
		}
	}

	sb.WriteString(headerBox.Render(headerContent) + "\n")

	if m.monitor.ChecksCount == 0 {
		sb.WriteString(infoStyle.Render("Initializing connection..."))
	} else {
		for _, line := range m.outputLines {
			sb.WriteString(line + "\n")
		}

		if m.done {
			titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#89B4FA")).PaddingLeft(0)
			stats := m.monitor.GetStats()

			sb.WriteString("\n")
			sb.WriteString(titleStyle.Render(fmt.Sprintf("--- %s statistics ---", m.config.URL)) + "\n")
			sb.WriteString(fmt.Sprintf("%d checks, %d successful (%.1f%%)",
				stats.ChecksCount, stats.SuccessCount, stats.UptimePercent) + "\n")

			if stats.ChecksCount > 0 {
				rtStats := fmt.Sprintf("response time min/avg/max/stddev = %d/%d/%d/%.1f ms",
					stats.MinResponseTime.Milliseconds(),
					stats.AvgResponseTime.Milliseconds(),
					stats.MaxResponseTime.Milliseconds(),
					stats.StdDev)

				if stats.ChecksCount >= 2 && stats.P95 > 0 {
					rtStats += fmt.Sprintf(", 95th percentile: %d ms", stats.P95.Milliseconds())
				}

				sb.WriteString(rtStats + "\n")
			}
		}
	}

	if !m.done {
		sb.WriteString("\n")
		sb.WriteString(infoStyle.Render("Press q to quit"))
	}

	return sb.String()
}

func createCheckWebsiteCmd(config Config) tea.Cmd {
	return func() tea.Msg {
		netConfig := net.NetworkConfig{
			Timeout:         config.Timeout,
			ShouldFail:      config.ShouldFail,
			FollowRedirects: config.FollowRedirects,
			SkipSSL:         config.SkipSSL,
			AssertText:      config.AssertText,
		}

		result := net.CheckWebsite(config.URL, netConfig)
		return result
	}
}

func StartBubbleTeaMonitoring(config Config) {
	model := initialModel(config)

	p := tea.NewProgram(model)

	_, err := p.Run()
	if err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
