# Updo with Prometheus and Grafana

This example demonstrates how to monitor your websites with updo and visualize the metrics in Grafana using Prometheus as the data source.

## Quick Start

```bash
# Start the monitoring stack
docker compose up -d

# Run updo with Prometheus integration
updo monitor --prometheus-url http://localhost:9090/api/v1/write https://example.com

# Or use the example configuration
updo monitor --config updo-example.toml --prometheus-url http://localhost:9090/api/v1/write
```

**Access the interfaces:**

- Grafana: <http://localhost:3000> (anonymous access enabled)
- Prometheus: <http://localhost:9090>

## What's Included

- `docker-compose.yml`: Complete Prometheus + Grafana stack
- `updo-example.toml`: Sample updo configuration with various target types
- `grafana/`: Grafana dashboard and provisioning files (flat structure)
- `prometheus.yml`: Prometheus configuration

## Dashboard Overview

The Grafana dashboard provides actionable monitoring insights with the following panels:

### Core Monitoring

- **Target Uptime Status**: Real-time uptime tracking for all monitored targets
- **Response Time**: HTTP response time trends over time  
- **Current Status**: At-a-glance status indicators for each target

### Actionable Tables

- **HTTP Status Codes Summary**: Table view showing target, status code, and counts with color coding
  - 🟢 Green for 2xx success codes
  - 🔴 Red for 4xx/5xx error codes
- **SSL Certificate Expiry**: Table showing days until expiry with warning levels
  - 🔴 Red: <30 days (urgent)
  - 🟡 Yellow: 30-90 days (warning)
  - 🟢 Green: >90 days (safe)

### Performance Breakdown

- **DNS Lookup Time**: DNS resolution performance
- **TCP Connection Time**: Connection establishment metrics
- **Time to First Byte**: Server processing performance
- **Download Duration**: Content transfer time
- **Wait Time**: Pre-request delays

## How It Works

Updo exports metrics to Prometheus via the Remote Write protocol. This allows you to:

- Collect uptime and performance metrics in Prometheus
- Create custom dashboards in Grafana
- Set up alerting based on your monitoring data
- Analyze historical trends

## Metrics

Updo exports these metrics to Prometheus:

### Core Health & Performance

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|---------|
| `updo_target_up` | Gauge | Target availability (1 = up, 0 = down) | `name`, `url`, `region` |
| `updo_response_time_seconds` | Gauge | Total response time in seconds | `name`, `url`, `region` |
| `updo_http_status_code_total` | Counter | HTTP status codes received | `name`, `url`, `region`, `status_code` |

### Timing Breakdown

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|---------|
| `updo_wait_seconds` | Gauge | Time waiting before DNS lookup | `name`, `url`, `region` |
| `updo_dns_lookup_seconds` | Gauge | DNS resolution time | `name`, `url`, `region` |
| `updo_tcp_connection_seconds` | Gauge | TCP connection establishment time | `name`, `url`, `region` |
| `updo_time_to_first_byte_seconds` | Gauge | Server processing time (TTFB) | `name`, `url`, `region` |
| `updo_download_duration_seconds` | Gauge | Content transfer time | `name`, `url`, `region` |

### Quality & Operations

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|---------|
| `updo_assertion_passed` | Gauge | Text assertion result (1 = passed, 0 = failed) | `name`, `url`, `region` |
| `updo_ssl_cert_expiry_days` | Gauge | Days until SSL certificate expires | `name`, `url` |

## Example Queries

**Basic uptime:**

```promql
# Current uptime status
updo_target_up

# Uptime percentage over last 24 hours
avg_over_time(updo_target_up[24h]) * 100
```

**Response time analysis:**

```promql
# Current response times
updo_response_time_seconds

# Average response time by region
avg by (region) (updo_response_time_seconds)
```

**Error tracking:**

```promql
# Downtime percentage over time
(1 - avg_over_time(updo_target_up[5m])) * 100

# HTTP error rate (4xx/5xx responses)
rate(updo_http_status_code_total{status_code=~"[45].."}[5m])
```

**Timing breakdown analysis:**

```promql
# DNS issues - high DNS lookup times
updo_dns_lookup_seconds > 0.1

# Network performance - TCP connection time by region
avg by (region) (updo_tcp_connection_seconds)

# Server performance - Time to First Byte
updo_time_to_first_byte_seconds

# Bandwidth bottlenecks - download duration
updo_download_duration_seconds
```

## Configuration

### Basic Usage

```bash
# Basic Prometheus integration
updo monitor --prometheus-url http://localhost:9090/api/v1/write https://example.com

# Multi-region monitoring
updo monitor --regions us-east-1,eu-west-1 --prometheus-url http://localhost:9090/api/v1/write https://example.com
```

### Authentication & Configuration

Configure Prometheus integration via environment variables:

```bash
# Server URL (optional if using --prometheus-url flag)
export UPDO_PROMETHEUS_RW_SERVER_URL="https://prometheus.example.com/api/v1/write"

# HTTP Basic Auth
export UPDO_PROMETHEUS_USERNAME="your-username"
export UPDO_PROMETHEUS_PASSWORD="your-password"
updo monitor https://example.com

# Bearer Token Authentication
export UPDO_PROMETHEUS_BEARER_TOKEN="your-bearer-token"
updo monitor https://example.com

# Custom headers (format: "Header-Name: value")
export UPDO_PROMETHEUS_AUTH_HEADER="X-API-Key: your-api-key"
updo monitor https://example.com

# Custom push interval (default: 5s)
export UPDO_PROMETHEUS_PUSH_INTERVAL="10s"
updo monitor https://example.com
```

**Supported Environment Variables:**

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `UPDO_PROMETHEUS_RW_SERVER_URL` | Prometheus Remote Write endpoint URL | - | `http://localhost:9090/api/v1/write` |
| `UPDO_PROMETHEUS_USERNAME` | HTTP Basic Auth username | - | `myuser` |
| `UPDO_PROMETHEUS_PASSWORD` | HTTP Basic Auth password | - | `mypass` |
| `UPDO_PROMETHEUS_BEARER_TOKEN` | Bearer token for Authorization header | - | `abc123` |
| `UPDO_PROMETHEUS_AUTH_HEADER` | Custom auth header (format: "Name: value") | - | `X-API-Key: secret` |
| `UPDO_PROMETHEUS_PUSH_INTERVAL` | Metrics push frequency | `5s` | `10s`, `30s`, `1m` |

> **Note**: Authentication via command line flags is not supported to avoid exposing credentials in shell history. Environment variables provide secure credential management for CI/CD and production environments. The Prometheus URL can be provided via either the `--prometheus-url` CLI flag or the `UPDO_PROMETHEUS_RW_SERVER_URL` environment variable.

### Remote Write Client Settings

The Remote Write client uses these defaults:

- Push Interval: 5 seconds
- Retry Attempts: 3
- Timeout: 10 seconds per request
- Compression: Snappy compression enabled

## Troubleshooting

**Connection Issues:**

1. Verify containers are running: `docker ps`
2. Check Prometheus logs: `docker logs prometheus-grafana-prometheus-1`
3. Ensure Remote Write endpoint responds: `curl -X POST http://localhost:9090/api/v1/write`

**Missing Metrics:**

1. Check updo output for error messages
2. Verify the Remote Write endpoint URL is correct
3. Query Prometheus directly: `curl "http://localhost:9090/api/v1/query?query=updo_target_up"`

**Performance:**

- For high-frequency monitoring, consider increasing push intervals
- Monitor Prometheus ingestion rate and memory usage
- Set appropriate retention policies for your needs
