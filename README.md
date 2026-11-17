<div align="center">

<img src="assets/updo.svg" width="72" alt="Updo"/>

# Updo

Updo is a command-line tool for monitoring website uptime and performance, with response times, SSL certificate expiry and alerts.

[![Latest Release](https://img.shields.io/github/v/release/Owloops/updo?style=flat-square&color=FF6C19)](https://github.com/Owloops/updo/releases/latest)
[![Go Reference](https://img.shields.io/badge/reference-FF6C19?style=flat-square&logo=go&logoColor=white)](https://pkg.go.dev/github.com/Owloops/updo)
[![License: MIT](https://img.shields.io/badge/license-MIT-FF6C19?style=flat-square)](LICENSE)

<img src="images/demo.gif" alt="Updo demo" width="700"/>

</div>

## Features

<table width="100%">
  <tr>
    <td width="33%" valign="top" align="center">
      <br>
      <h3 align="center"><img src="https://api.iconify.design/lucide/activity.svg?color=%23FF6C19" width="20" height="20">&nbsp; Real-time</h3>
      <p align="center">Uptime percentage, response times and SSL certificate tracking, updated as it runs.</p>
      <br>
    </td>
    <td width="33%" valign="top" align="center">
      <br>
      <h3 align="center"><img src="https://api.iconify.design/lucide/list.svg?color=%23FF6C19" width="20" height="20">&nbsp; Multi-target</h3>
      <p align="center">Watch many URLs at once, from the command line or a TOML config, each with its own interval and rules.</p>
      <br>
    </td>
    <td width="33%" valign="top" align="center">
      <br>
      <h3 align="center"><img src="https://api.iconify.design/lucide/globe.svg?color=%23FF6C19" width="20" height="20">&nbsp; Multi-region</h3>
      <p align="center">Deploy probes as AWS Lambda functions across 13 regions to see your site from where your users are.</p>
      <br>
    </td>
  </tr>
  <tr>
    <td width="33%" valign="top" align="center">
      <br>
      <h3 align="center"><img src="https://api.iconify.design/lucide/code.svg?color=%23FF6C19" width="20" height="20">&nbsp; Flexible HTTP</h3>
      <p align="center">Custom headers, POST and PUT bodies, redirect and SSL options, response assertions, IPv4 or IPv6 only checks.</p>
      <br>
    </td>
    <td width="33%" valign="top" align="center">
      <br>
      <h3 align="center"><img src="https://api.iconify.design/lucide/bell.svg?color=%23FF6C19" width="20" height="20">&nbsp; Alerts</h3>
      <p align="center">Desktop notifications plus webhooks. Slack and Discord are detected and formatted automatically.</p>
      <br>
    </td>
    <td width="33%" valign="top" align="center">
      <br>
      <h3 align="center"><img src="https://api.iconify.design/lucide/chart-line.svg?color=%23FF6C19" width="20" height="20">&nbsp; Output</h3>
      <p align="center">Interactive TUI, plain text or JSON logs, and Prometheus export for Grafana dashboards and long-term storage.</p>
      <br>
    </td>
  </tr>
</table>

<img src="assets/dividers/table-of-contents.svg" alt="" width="100%"/>

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
  - [Options](#options)
  - [Examples](#examples)
- [Configuration File](#configuration-file)
  - [Example Configuration](#example-configuration)
  - [Configuration Options](#configuration-options)
- [Multi-Region Monitoring](#multi-region-monitoring)
  - [Prerequisites](#prerequisites)
- [Webhook Notifications](#webhook-notifications)
  - [Supported Platforms](#supported-platforms)
  - [Integration Examples](#integration-examples)
- [Prometheus & Grafana Integration](#prometheus--grafana-integration)
- [Structured Logging](#structured-logging)
- [Keyboard Shortcuts](#keyboard-shortcuts)
- [Exit Codes](#exit-codes)
- [Community](#community)

<img src="assets/dividers/installation.svg" alt="" width="100%"/>

## Installation

```bash
# macOS and Linux, Homebrew
brew install owloops/tap/updo

# Linux and macOS, install script picks the right release asset
curl -sSL https://raw.githubusercontent.com/Owloops/updo/main/install.sh | bash

# Arch Linux
yay -S updo

# Nix, run without installing
nix run github:Owloops/updo -- monitor https://example.com

# Docker, build the image and run it in one line
docker run --rm -it "$(docker build -q https://github.com/Owloops/updo.git)" monitor https://example.com

# Any platform with Go 1.27 or newer
go install github.com/Owloops/updo@latest
```

### Package files

Every release ships native packages for Linux plus tarballs for macOS and an executable for Windows. Grab them from the [latest release](https://github.com/Owloops/updo/releases/latest):

| Platform | File |
|---|---|
| Debian, Ubuntu | `updo_<version>_linux_amd64.deb`, `dpkg -i` |
| Fedora, RHEL, openSUSE | `updo_<version>_linux_amd64.rpm`, `rpm -i` or `zypper install` |
| Alpine | `updo_<version>_linux_amd64.apk`, `apk add --allow-untrusted` |
| Arch | `updo-bin` from the AUR, or the `.pkg.tar.zst` above |
| macOS | `updo_Darwin_arm64`, `updo_Darwin_x86_64` |
| Windows | `updo_Windows_x86_64.exe`, move it somewhere on your `PATH` |

The install script handles the correct asset, architecture and `PATH` for you, which is why it is the shortest path on Linux and macOS.

## Usage

```bash
# Monitor URLs
updo monitor <website-url> [options]
updo monitor <url1> <url2> <url3>

# Using configuration file
updo monitor --config <config-file>

# Generate shell completions
updo completion bash > updo_completion.bash
```

### Options

**Basic:**

- `--url, --config`: Target URL or TOML config file
- `--refresh`: Check interval in seconds (default: 5)
- `--timeout`: Request timeout in seconds (default: 10)  
- `--count`: Number of checks (0 = infinite)
- `--simple`: Text output instead of TUI

**HTTP:**

- `--header`: Custom HTTP headers (repeatable)
- `--request`: HTTP method (default: GET)
- `--data`: Request body data
- `--skip-ssl, --follow-redirects, --accept-redirects`: SSL and redirect options
- `--assert-text`: Expected response text
- `-4, --ipv4` / `-6, --ipv6`: Restrict checks to one address family

**Multi-region:**

- `--regions`: AWS regions (comma-separated or 'all')
- `--profile`: AWS profile for remote executors

**Output & Alerts:**

- `--log`: JSON structured logging
- `--webhook-url, --webhook-header`: Webhook notifications
- `--only, --skip`: Target filtering

> [!NOTE]
> When using CLI flags, all settings (headers, webhook URL, timeouts, etc.) apply globally to all monitored targets. For per-target configuration, use a TOML configuration file.

### Examples

```bash
# Basic monitoring
updo monitor https://example.com

# Set custom refresh and timeout
updo monitor --refresh 10 --timeout 5 https://example.com

# Simple mode and logging
updo monitor --simple --count 10 https://example.com
updo monitor --log --count 10 https://example.com > output.json

# Custom requests
updo monitor --header "Authorization: Bearer token" --assert-text "Welcome" https://example.com
updo monitor --request POST --header "Content-Type: application/json" --data '{"test":"data"}' https://api.example.com

# Multi-target monitoring
updo monitor https://google.com https://github.com https://cloudflare.com
updo monitor --config example-config.toml --only Google,GitHub

# Multi-region monitoring
updo monitor --regions us-east-1,eu-west-1 https://example.com
updo monitor --regions all --profile production https://example.com

# Webhook notifications
updo monitor --webhook-url "https://hooks.slack.com/services/YOUR/WEBHOOK" https://example.com
```

<img src="assets/dividers/configuration.svg" alt="" width="100%"/>

## Configuration File

Use TOML configuration for complex monitoring setups with multiple targets.

### Example Configuration

```toml
[global]
refresh_interval = 5
timeout = 10
webhook_url = "https://hooks.slack.com/services/YOUR/WEBHOOK"
only = ["Google", "API"]  # Monitor only these targets

[[targets]]
url = "https://www.google.com"
name = "Google"
refresh_interval = 3
assert_text = "Google"

[[targets]]
url = "https://api.example.com/health"
name = "API"
method = "POST"
headers = ["Authorization: Bearer token"]
```

### Configuration Options

**Global settings** (apply to all targets unless overridden):

- `refresh_interval`, `timeout`, `follow_redirects`, `accept_redirects`, `receive_alert`, `count`
- `body_size_limit`: Response body cap in bytes (default `1048576` = 1 MiB; `0` means no limit)
- `webhook_url`, `webhook_headers`: Default webhook settings
- `only`, `skip`: Target filtering arrays
- `regions`: AWS regions for remote executors

**Target settings** (can override global):

- `url` (required), `name`: Target identification  
- `method`, `headers`, `body`: HTTP request options
- `assert_text`, `should_fail`: Response validation
- `skip_ssl`, `follow_redirects`, `accept_redirects`: Connection options
- `body_size_limit`: Per-target response body cap (set to `0` to disable capping for this target)
- `webhook_url`, `webhook_headers`: Per-target notifications
- `regions`: Target-specific AWS regions

> [!IMPORTANT]
> Response bodies are capped at `body_size_limit` bytes when evaluating `assert_text`. If your asserted text appears beyond the cap, the assertion fails and the probe logs a warning (visible in the Recent Logs widget in TUI mode, or on stderr in simple mode). Raise `body_size_limit` or set it to `0` for targets returning large payloads.

<img src="assets/dividers/multi-region.svg" alt="" width="100%"/>

## Multi-Region Monitoring

Deploy remote executors as AWS Lambda functions across 13 global regions for distributed monitoring from multiple geographic locations.

```bash
# Deploy remote executors to AWS regions
updo aws deploy --regions us-east-1,eu-west-1

# Monitor using remote executors
updo monitor --regions us-east-1,eu-west-1 https://example.com

# Cleanup when done
updo aws destroy --regions all
```

### Prerequisites

**AWS CLI configured** with appropriate credentials and the following permissions:

| Service | Required Permissions |
|---------|---------------------|
| Lambda | CreateFunction, UpdateFunctionCode, DeleteFunction, GetFunction, InvokeFunction |
| IAM | CreateRole, AttachRolePolicy, DetachRolePolicy, DeleteRole, GetRole |
| STS | GetCallerIdentity |

**Supported regions:** us-east-1, us-west-1, us-west-2, eu-west-1, eu-central-1, eu-west-2, ap-southeast-1, ap-southeast-2, ap-northeast-1, ap-northeast-2, ap-south-1, sa-east-1, ca-central-1

**Troubleshooting:** If you get credential errors, run `aws sso login --profile your-profile` to refresh expired sessions.

<img src="assets/dividers/webhooks.svg" alt="" width="100%"/>

## Webhook Notifications

Updo can send webhook notifications when targets go up or down. Updo **automatically detects** Slack and Discord webhooks by URL pattern and formats messages accordingly with rich formatting. Custom webhooks receive a generic JSON payload.

### Supported Platforms

- **Slack** - Auto-detected via `hooks.slack.com` URL, sends rich messages with attachments and color coding
- **Discord** - Auto-detected via `discord.com/api/webhooks` URL, sends embeds with color and structured fields
- **Custom** - Any other webhook URL receives generic JSON format

### Integration Examples

**Slack Webhook (Auto-Detected):**

```toml
[[targets]]
url = "https://api.example.com"
name = "Production API"
webhook_url = "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

Updo automatically formats Slack messages with:
- Color-coded attachments (red for down, green for up)
- Unicode symbols (✘ for down, ✔ for up)
- Structured fields for URL, error, status code, response time, and timestamp

**Discord Webhook (Auto-Detected):**

```toml
[[targets]]
url = "https://api.example.com"
name = "Production API"
webhook_url = "https://discord.com/api/webhooks/123456789/YOUR_WEBHOOK_TOKEN"
```

Updo automatically formats Discord messages with:
- Color-coded embeds (red for down, green for up)
- Unicode symbols (✘ for down, ✔ for up)
- Structured fields with inline formatting
- Clickable URL links

**Custom Webhook:**

For custom webhooks, Updo sends a generic JSON payload:

```json
{
  "event": "target_down",
  "target": "Production API",
  "url": "https://api.example.com",
  "timestamp": "2024-01-01T12:00:00Z",
  "response_time_ms": 1500,
  "status_code": 500,
  "error": "Internal Server Error"
}
```

```toml
[[targets]]
url = "https://critical-service.example.com"
name = "Critical Service"
webhook_url = "https://alerts.internal.com/webhook"
webhook_headers = [
  "Authorization: Bearer YOUR_TOKEN",
  "X-Service: updo-monitor"
]
```

<img src="assets/dividers/prometheus.svg" alt="" width="100%"/>

## Prometheus & Grafana Integration

Export updo metrics to Prometheus for long-term storage, visualization, and alerting:

```bash
# Basic Prometheus integration
updo monitor --prometheus-url http://localhost:9090/api/v1/write https://example.com

# Via environment variables (CLI flag optional if URL provided via env)
export UPDO_PROMETHEUS_RW_SERVER_URL="https://prometheus.example.com/api/v1/write"
export UPDO_PROMETHEUS_USERNAME="admin"
export UPDO_PROMETHEUS_PASSWORD="secret"
updo monitor https://example.com
```

**Available metrics:**

- Target uptime and response times
- HTTP status codes and timing breakdown (DNS, TCP, TTFB, download)
- SSL certificate expiry and assertion results

**Quick start with Docker:**

```bash
# Clone and start the monitoring stack
git clone https://github.com/Owloops/updo.git
cd updo/examples/prometheus-grafana
docker compose up -d
```

Access Grafana at [http://localhost:3000](http://localhost:3000) for pre-built dashboards.

<table>
   <tr>
      <td align="center">
         <video src="https://github.com/user-attachments/assets/67c8e51d-fe6f-436a-a34d-cdc2bbf23f46" controls style="width:100%; max-width:720px; height:400px;"></video>
      </td>
   </tr>
</table>

> [!TIP]
> Full documentation lives in [examples/prometheus-grafana/README.md](examples/prometheus-grafana/README.md): complete setup, authentication options, metrics reference and PromQL examples.

<img src="assets/dividers/logging.svg" alt="" width="100%"/>

## Structured Logging

The `--log` flag outputs JSON-formatted logs for programmatic consumption:

- **Check logs** (stdout): HTTP requests, responses, and timing information
- **Metrics logs** (stdout): Uptime, response time stats, success rate
- **Error logs** (stderr): Failures, warnings, and assertion results

Usage examples:

```bash
# All logs to one file
updo monitor --log https://example.com > all.json 2>&1

# Metrics to one file, errors to another
updo monitor --log https://example.com > metrics.json 2> errors.json

# Processing with jq
updo monitor --log https://example.com | jq 'select(.type=="check") | .response_time_ms'
```

<img src="assets/dividers/shortcuts.svg" alt="" width="100%"/>

## Keyboard Shortcuts

In the TUI:

| Keys | Action |
|---|---|
| <kbd>↑</kbd> <kbd>↓</kbd> | Move through targets |
| <kbd>Tab</kbd> | Collapse or expand every target group |
| <kbd>Enter</kbd> | Collapse or expand the selected group |
| <kbd>/</kbd> | Search, <kbd>Esc</kbd> to leave search |
| <kbd>l</kbd> | Show or hide logs for the selected target |
| <kbd>Esc</kbd> | Close the logs panel |
| <kbd>q</kbd> <kbd>Ctrl</kbd>+<kbd>c</kbd> | Quit |

## Exit Codes

Useful for cron jobs, scripts and CI:

- `0`: every checked target was up
- `1`: at least one target was down or failed its assertion
- `2`: usage or configuration error

Runs stopped with `Ctrl+C` or `q` exit with `0`, since the run did not finish on its own.

<img src="assets/dividers/community.svg" alt="" width="100%"/>

## Community

Contributions are welcome. [CONTRIBUTING.md](CONTRIBUTING.md) covers building, testing, commit style and what a mergeable pull request looks like.

The TUI runs on [termui](https://github.com/gizak/termui) and [tcell](https://github.com/gdamore/tcell), which the [Owloops fork](https://github.com/Owloops/termui) keeps on a maintained terminal layer.

The project has been featured in [awesome-cli-apps](https://github.com/agarrharr/awesome-cli-apps), [awesome-readme](https://github.com/matiassingers/awesome-readme), [Terminal Trove](https://terminaltrove.com/updo) and [Jupiter Broadcasting](https://www.jupiterbroadcasting.com/show/linux-unplugged/664/).

<br>

<div align="center">

<sub>Built with&nbsp;<sub><img src="https://api.iconify.design/ph/heart-fill.svg?color=%23FF6C19" width="14" height="14" alt="love"></sub>&nbsp;by <a href="https://owloops.com">Owloops</a></sub>

</div>
