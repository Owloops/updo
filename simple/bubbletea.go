package simple

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
	"github.com/Owloops/updo/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	config      Config
	monitor     *stats.Monitor
	checkResult net.WebsiteCheckResult
	outputLines []string
	done        bool
	checks      int
	lastCheck   time.Time
	alertSent   bool
}

type quitMsg struct{}

func initialModel(config Config) Model {
	monitor, err := stats.NewMonitor()
	if err != nil {
		log.Fatalf("Failed to initialize stats monitor: %v", err)
	}
	return Model{
		config:      config,
		monitor:     monitor,
		outputLines: make([]string, 0, 100),
		done:        false,
		lastCheck:   time.Time{},
	}
}

func (m Model) Init() tea.Cmd {
	return tickCmd(time.Millisecond)
}

type tickMsg time.Time
type resultMsg net.WebsiteCheckResult

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.done = true
			return m, tea.Quit
		}

	case quitMsg:
		m.done = true
		return m, tea.Quit

	case tickMsg:
		return m, checkWebsiteCmd(m.config)

	case resultMsg:
		result := net.WebsiteCheckResult(msg)
		m.checkResult = result
		m.monitor.AddResult(result)
		m.checks++

		stats := m.monitor.GetStats()
		pingLine := formatPingLine(result, stats, m.checks-1, m.config)
		m.outputLines = append(m.outputLines, pingLine)

		if len(m.outputLines) > 500 {

			copy(m.outputLines, m.outputLines[len(m.outputLines)-400:])
			m.outputLines = m.outputLines[:400]
		}

		if m.config.Count > 0 && m.checks >= m.config.Count {
			m.done = true
			return m, tea.Quit
		}

		if m.config.ReceiveAlert {
			utils.HandleAlerts(result.IsUp, &m.alertSent)
		}

		return m, tickCmd(m.config.RefreshInterval)
	}

	return m, nil
}

var (
	successStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8"))
	valueStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#F5E0DC"))
	highlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAB387"))
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#89B4FA"))
	urlStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1")).Bold(true)
	infoStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4"))
	sslStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#F9E2AF"))
	headerBox      = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#CBA6F7")).
			Padding(0, 1).
			MarginBottom(1).
			Align(lipgloss.Left).
			Width(40)
)

func formatPingLine(result net.WebsiteCheckResult, stats stats.Stats, seq int, config Config) string {
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
	var sb strings.Builder

	sb.WriteString(m.renderHeader())

	if m.monitor.ChecksCount == 0 {
		sb.WriteString(infoStyle.Render("Initializing connection..."))
	} else {
		sb.WriteString(m.renderOutputLines())

		if m.done {
			sb.WriteString(m.renderStatistics())
		}
	}

	return sb.String()
}

func (m Model) renderHeader() string {
	header := titleStyle.Render("🐤 UPDO") + " " +
		urlStyle.Render(m.config.URL)

	headerContent := header
	if strings.HasPrefix(m.config.URL, "https://") && !m.config.SkipSSL {
		sslDaysRemaining := net.GetSSLCertExpiry(m.config.URL)
		if sslDaysRemaining > 0 {
			headerContent = header + "\n" + sslStyle.Render(fmt.Sprintf("SSL certificate expires in %d days", sslDaysRemaining))
		}
	}

	return headerBox.Render(headerContent) + "\n"
}

func (m Model) renderOutputLines() string {
	var sb strings.Builder
	for _, line := range m.outputLines {
		sb.WriteString(line + "\n")
	}
	return sb.String()
}

func (m Model) renderStatistics() string {
	var sb strings.Builder
	statsTitle := lipgloss.NewStyle().Foreground(lipgloss.Color("#89B4FA")).PaddingLeft(0)
	stats := m.monitor.GetStats()

	successPercent := 0.0
	if stats.ChecksCount > 0 {
		successPercent = float64(stats.SuccessCount) / float64(stats.ChecksCount) * 100
	}

	sb.WriteString("\n")
	sb.WriteString(statsTitle.Render(fmt.Sprintf("--- %s statistics ---", m.config.URL)) + "\n")
	sb.WriteString(fmt.Sprintf("%d checks, %d successful (%.1f%%)",
		stats.ChecksCount, stats.SuccessCount, successPercent) + "\n")
	sb.WriteString(fmt.Sprintf("uptime: %.1f%%", stats.UptimePercent) + "\n")

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

	return sb.String()
}

func checkWebsiteCmd(config Config) tea.Cmd {
	return func() tea.Msg {
		netConfig := net.NetworkConfig{
			Timeout:         config.Timeout,
			ShouldFail:      config.ShouldFail,
			FollowRedirects: config.FollowRedirects,
			SkipSSL:         config.SkipSSL,
			AssertText:      config.AssertText,
			Headers:         config.Headers,
		}

		result := net.CheckWebsite(config.URL, netConfig)
		return resultMsg(result)
	}
}

func StartBubbleTeaMonitoring(config Config) {

	p := tea.NewProgram(initialModel(config))

	go func() {

		c := make(chan os.Signal, 1)

		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		<-c

		signal.Stop(c)
		close(c)

		p.Send(quitMsg{})

		<-time.After(150 * time.Millisecond)
		os.Exit(0)
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
