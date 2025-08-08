package metrics

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	prompb "buf.build/gen/go/prometheus/prometheus/protocolbuffers/go"
	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
	"github.com/golang/snappy"
	"google.golang.org/protobuf/proto"
)

const (
	_maxRetries  = 3
	_retryDelay  = 1 * time.Second
	_httpTimeout = 10 * time.Second
)

type WriteClient struct {
	config     Config
	httpClient *http.Client
	mu         sync.RWMutex
	samples    []*prompb.TimeSeries
	ctx        context.Context
	cancel     context.CancelFunc
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

func NewWriteClient(cfg Config) *WriteClient {
	ctx, cancel := context.WithCancel(context.Background())

	return &WriteClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: _httpTimeout,
		},
		samples:  make([]*prompb.TimeSeries, 0),
		ctx:      ctx,
		cancel:   cancel,
		stopChan: make(chan struct{}),
	}
}

func (c *WriteClient) Start() {
	c.wg.Add(1)
	go c.pushLoop()
}

func (c *WriteClient) Stop() {
	c.flushSamples()
	close(c.stopChan)
	c.cancel()
	c.wg.Wait()
}

func (c *WriteClient) AddCheck(target config.Target, result net.WebsiteCheckResult, region string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	timeSeries := ConvertCheckToTimeSeries(target, result, region, time.Time{})
	c.samples = append(c.samples, timeSeries...)
}

func (c *WriteClient) AddSSLExpiry(target config.Target, daysUntilExpiry int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ts := ConvertSSLExpiryToTimeSeries(target, daysUntilExpiry, time.Time{}); ts != nil {
		c.samples = append(c.samples, ts)
	}
}

func (c *WriteClient) pushLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.PushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.flushSamples()
		}
	}
}

func (c *WriteClient) flushSamples() {
	c.mu.Lock()
	samples := make([]*prompb.TimeSeries, len(c.samples))
	copy(samples, c.samples)
	c.samples = c.samples[:0]
	c.mu.Unlock()

	if len(samples) == 0 {
		return
	}

	if err := c.sendSamples(samples); err != nil {
		fmt.Printf("Error sending metrics to Prometheus: %v\n", err)
	}
}

func (c *WriteClient) sendSamples(samples []*prompb.TimeSeries) error {
	samples = c.normalizeTimestamps(samples)

	writeReq := &prompb.WriteRequest{
		Timeseries: samples,
	}

	data, err := proto.Marshal(writeReq)
	if err != nil {
		return fmt.Errorf("failed to marshal protobuf: %w", err)
	}

	compressed := snappy.Encode(nil, data)

	for attempt := range _maxRetries {
		if err := c.doRequest(compressed); err != nil {
			if attempt < _maxRetries-1 {
				time.Sleep(_retryDelay * time.Duration(attempt+1))
				continue
			}
			return fmt.Errorf("failed to send after %d attempts: %w", _maxRetries, err)
		}
		return nil
	}

	return nil
}

func (c *WriteClient) doRequest(data []byte) error {
	req, err := http.NewRequestWithContext(c.ctx, "POST", c.config.ServerURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	for key, value := range c.config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, readErr := io.ReadAll(resp.Body)
		if readErr == nil && len(body) > 0 {
			return fmt.Errorf("server responded with status %d: %s - %s", resp.StatusCode, resp.Status, string(body))
		}
		return fmt.Errorf("server responded with status %d: %s", resp.StatusCode, resp.Status)
	}

	return nil
}

func (c *WriteClient) normalizeTimestamps(samples []*prompb.TimeSeries) []*prompb.TimeSeries {
	batchTimestamp := time.Now().Truncate(time.Millisecond).UnixMilli()

	validSeries := samples[:0]
	for _, ts := range samples {
		if ts == nil || len(ts.Samples) == 0 {
			continue
		}

		validSamples := ts.Samples[:0]
		for _, sample := range ts.Samples {
			if sample != nil {
				sample.Timestamp = batchTimestamp
				validSamples = append(validSamples, sample)
			}
		}

		if len(validSamples) > 0 {
			ts.Samples = validSamples
			validSeries = append(validSeries, ts)
		}
	}

	return validSeries
}

var _globalClient *WriteClient

func InitRemoteWrite(cfg Config) {
	_globalClient = NewWriteClient(cfg)
	_globalClient.Start()
}

func StopRemoteWrite() {
	if _globalClient != nil {
		_globalClient.Stop()
		_globalClient = nil
	}
}

func RecordCheck(target config.Target, result net.WebsiteCheckResult, region string) {
	if _globalClient != nil {
		_globalClient.AddCheck(target, result, region)
	}
}

func RecordSSLExpiry(target config.Target, daysUntilExpiry int) {
	if _globalClient != nil {
		_globalClient.AddSSLExpiry(target, daysUntilExpiry)
	}
}
