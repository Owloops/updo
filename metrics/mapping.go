package metrics

import (
	"sort"
	"strconv"
	"time"

	"maps"

	prompb "buf.build/gen/go/prometheus/prometheus/protocolbuffers/go"
	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
)

const _nameLbl = "__name__"

type TimeSeries struct {
	Name   string
	Labels map[string]string
}

func MapTargetLabels(target config.Target, result net.WebsiteCheckResult, region string) map[string]string {
	labels := make(map[string]string)
	labels["name"] = target.Name
	labels["url"] = target.URL
	labels["region"] = region
	return labels
}

func MapSeries(name string, labels map[string]string) []*prompb.Label {
	pbLabels := make([]*prompb.Label, 0, len(labels)+1)

	pbLabels = append(pbLabels, &prompb.Label{
		Name:  _nameLbl,
		Value: _defaultMetricPrefix + name,
	})

	for key, value := range labels {
		if key == "" || value == "" {
			continue
		}
		pbLabels = append(pbLabels, &prompb.Label{
			Name:  key,
			Value: value,
		})
	}

	sort.Slice(pbLabels, func(i, j int) bool {
		return pbLabels[i].Name < pbLabels[j].Name
	})

	return pbLabels
}

func ConvertCheckToTimeSeries(target config.Target, result net.WebsiteCheckResult, region string, timestamp time.Time) []*prompb.TimeSeries {
	var timeSeries []*prompb.TimeSeries
	labels := MapTargetLabels(target, result, region)
	ts := timestamp.UnixMilli()

	upValue := 0.0
	if result.IsUp {
		upValue = 1.0
	}

	timeSeries = append(timeSeries, &prompb.TimeSeries{
		Labels: MapSeries("target_up", labels),
		Samples: []*prompb.Sample{
			{
				Timestamp: ts,
				Value:     upValue,
			},
		},
	})

	if result.ResponseTime > 0 {
		timeSeries = append(timeSeries, &prompb.TimeSeries{
			Labels: MapSeries("response_time_seconds", labels),
			Samples: []*prompb.Sample{
				{
					Timestamp: ts,
					Value:     result.ResponseTime.Seconds(),
				},
			},
		})
	}

	if result.StatusCode > 0 {
		statusLabels := make(map[string]string)
		maps.Copy(statusLabels, labels)
		statusLabels["status_code"] = strconv.Itoa(result.StatusCode)

		timeSeries = append(timeSeries, &prompb.TimeSeries{
			Labels: MapSeries("http_status_code_total", statusLabels),
			Samples: []*prompb.Sample{
				{
					Timestamp: ts,
					Value:     1.0,
				},
			},
		})
	}

	if result.TraceInfo != nil {
		timingMetrics := map[string]time.Duration{
			"wait_seconds":               result.TraceInfo.Wait,
			"dns_lookup_seconds":         result.TraceInfo.DNSLookup,
			"tcp_connection_seconds":     result.TraceInfo.TCPConnection,
			"time_to_first_byte_seconds": result.TraceInfo.TimeToFirstByte,
			"download_duration_seconds":  result.TraceInfo.DownloadDuration,
		}

		for metricName, duration := range timingMetrics {
			if duration > 0 {
				timeSeries = append(timeSeries, &prompb.TimeSeries{
					Labels: MapSeries(metricName, labels),
					Samples: []*prompb.Sample{
						{
							Timestamp: ts,
							Value:     duration.Seconds(),
						},
					},
				})
			}
		}
	}

	if target.AssertText != "" {
		assertValue := 0.0
		if result.AssertionPassed {
			assertValue = 1.0
		}

		timeSeries = append(timeSeries, &prompb.TimeSeries{
			Labels: MapSeries("assertion_passed", labels),
			Samples: []*prompb.Sample{
				{
					Timestamp: ts,
					Value:     assertValue,
				},
			},
		})
	}

	return timeSeries
}

func ConvertSSLExpiryToTimeSeries(target config.Target, daysUntilExpiry int, timestamp time.Time) *prompb.TimeSeries {
	if daysUntilExpiry < 0 {
		return nil
	}

	labels := make(map[string]string)
	labels["name"] = target.Name
	labels["url"] = target.URL

	return &prompb.TimeSeries{
		Labels: MapSeries("ssl_cert_expiry_days", labels),
		Samples: []*prompb.Sample{
			{
				Timestamp: timestamp.UnixMilli(),
				Value:     float64(daysUntilExpiry),
			},
		},
	}
}
