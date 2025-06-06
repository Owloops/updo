package stats

import (
	"math"
	"time"

	"github.com/Owloops/updo/net"
	"github.com/caio/go-tdigest/v4"
)

type Monitor struct {
	ChecksCount       int
	SuccessCount      int
	TotalResponseTime time.Duration
	MinResponseTime   time.Duration
	MaxResponseTime   time.Duration
	StartTime         time.Time
	LastCheckTime     time.Time
	LastIP            string
	LastStatusCode    int
	TotalUptime       time.Duration
	IsUp              bool

	TDigest *tdigest.TDigest

	mean float64
	m2   float64
}

func NewMonitor() (*Monitor, error) {
	td, err := tdigest.New(tdigest.Compression(100))
	if err != nil {
		return nil, err
	}

	return &Monitor{
		StartTime:       time.Now(),
		MinResponseTime: time.Duration(math.MaxInt64),
		TDigest:         td,
	}, nil
}

func (m *Monitor) AddResult(result net.WebsiteCheckResult) {
	m.ChecksCount++
	m.LastIP = result.ResolvedIP
	m.LastStatusCode = result.StatusCode

	now := time.Now()
	if m.ChecksCount == 1 {
		m.LastCheckTime = now
		if result.IsUp {
			m.TotalUptime = now.Sub(m.StartTime)
		}
	} else {
		timeElapsedSinceLastCheck := now.Sub(m.LastCheckTime)
		m.LastCheckTime = now

		if m.IsUp {
			m.TotalUptime += timeElapsedSinceLastCheck
		}
	}

	m.IsUp = result.IsUp

	if result.IsUp {
		m.SuccessCount++
	}

	m.TotalResponseTime += result.ResponseTime

	if result.ResponseTime < m.MinResponseTime {
		m.MinResponseTime = result.ResponseTime
	}
	if result.ResponseTime > m.MaxResponseTime {
		m.MaxResponseTime = result.ResponseTime
	}

	if m.TDigest != nil {
		_ = m.TDigest.Add(result.ResponseTime.Seconds())
	}

	responseMs := float64(result.ResponseTime.Milliseconds())
	delta := responseMs - m.mean
	m.mean += delta / float64(m.ChecksCount)
	delta2 := responseMs - m.mean
	m.m2 += delta * delta2
}

type Stats struct {
	ChecksCount     int
	SuccessCount    int
	UptimePercent   float64
	AvgResponseTime time.Duration
	MinResponseTime time.Duration
	MaxResponseTime time.Duration
	StdDev          float64
	P95             time.Duration
	TotalDuration   time.Duration
	LastIP          string
	LastStatusCode  int
}

func (m *Monitor) GetStats() Stats {
	now := time.Now()

	currentUptime := m.TotalUptime
	if m.ChecksCount > 0 && m.IsUp {
		currentUptime += now.Sub(m.LastCheckTime)
	}

	stats := Stats{
		ChecksCount:     m.ChecksCount,
		SuccessCount:    m.SuccessCount,
		MinResponseTime: m.MinResponseTime,
		MaxResponseTime: m.MaxResponseTime,
		TotalDuration:   time.Since(m.StartTime),
		LastIP:          m.LastIP,
		LastStatusCode:  m.LastStatusCode,
	}

	totalMonitoredTime := time.Since(m.StartTime)
	if totalMonitoredTime > 0 {
		stats.UptimePercent = (float64(currentUptime) / float64(totalMonitoredTime)) * 100
	} else {
		stats.UptimePercent = 0
	}

	if m.ChecksCount > 0 {
		stats.AvgResponseTime = m.TotalResponseTime / time.Duration(m.ChecksCount)
	}

	if m.ChecksCount > 1 {
		variance := m.m2 / float64(m.ChecksCount-1)
		stats.StdDev = math.Sqrt(variance)
	}

	if m.TDigest != nil && m.ChecksCount >= 2 {
		p95Seconds := m.TDigest.Quantile(0.95)
		stats.P95 = time.Duration(p95Seconds * float64(time.Second))
	}

	return stats
}
