<div align="center">

# 🐤 Updo - Website Monitoring Tool

<p align="center">
  <img src="images/demo.png" alt="Updo demo" width="600"/>
</p>

Updo is a command-line tool for monitoring website uptime and performance. It provides real-time metrics on website status, response time, SSL certificate expiry, and more, with alert notifications.

![Language:Go](https://img.shields.io/static/v1?label=Language&message=Go&color=blue&style=flat-square)
![License:MIT](https://img.shields.io/static/v1?label=License&message=MIT&color=blue&style=flat-square)
[![Latest Release](https://img.shields.io/github/v/release/Owloops/updo?style=flat-square)](https://github.com/Owloops/updo/releases/latest)
</div>

## Features

- Real-time monitoring of website uptime and performance
- **Multi-target monitoring** - Monitor multiple URLs concurrently
- Displays various metrics like uptime percentage, average response time, and SSL certificate expiry
- Desktop alert notifications for website status changes
- **Webhook notifications** - Send alerts to Slack, Discord, or any webhook endpoint
- Customizable refresh intervals and request timeouts per target
- Supports HTTP and HTTPS, with options to skip SSL verification
- Assertion on response body content
- TOML configuration file support for complex setups
- Command-line interface with simple usage
- Simple mode with text output
- Automatic terminal capability detection

## Demo

<https://github.com/user-attachments/assets/f8a15cc7-7b30-448f-ab49-35396e6ed46f>

## Installation

<details>
<summary>Quick install script (Linux, macOS, Windows/MSYS)</summary>

#### One-line install command

```bash
curl -sSL https://raw.githubusercontent.com/Owloops/updo/main/install.sh | bash
```

This script automatically:

- Detects your OS and architecture
- Downloads the latest release
- Makes the binary executable
- Installs to /usr/local/bin (or ~/.local/bin if permission denied)
- Removes quarantine attribute on macOS

</details>

<details>
<summary>Download executable binaries</summary>

#### You can download executable binaries from the latest release page

> [![Latest Release](https://img.shields.io/github/v/release/Owloops/updo?style=flat-square)](https://github.com/Owloops/updo/releases/latest)
</details>

<details>
<summary>Build from source</summary>

#### You can install Updo by cloning the repository and building the binary

Make sure your system has Go [installed](https://go.dev/doc/install).

> ```bash
> git clone https://github.com/Owloops/updo.git
> cd updo
> go build
> ```
>
#### Build with version information

To include version information in the binary, use ldflags:

```bash
go build -ldflags="-X 'main.version=v1.0.0' -X 'main.commit=$(git rev-parse HEAD)' -X 'main.date=$(date)'"
```

Check the version with:

```bash
./updo --version
```

#### Another way to install it if you have go in your machine just

```sh
GOBIN="absolute_path_where_you_want_binaries_to_be_installed" go install github.com/Owloops/updo@latest
```

</details>

> [!NOTE]  
> You may get a warning message on Windows and MacOS, which is discussed in this issue <https://github.com/Owloops/updo/issues/4>
>
> ### macOS Security
>
> macOS may prevent running downloaded binaries due to security measures. If you get a warning message like "cannot be opened because the developer cannot be verified", you can remove the quarantine attribute with this command:
>
> ```bash
> xattr -d com.apple.quarantine /path/to/updo
> ```
>
> Replace `/path/to/updo` with the actual path to the downloaded binary (e.g. `~/Downloads/updo_Darwin_arm64/updo`)

## Usage

Run Updo using the following command:

```bash
# Monitor single URL
./updo monitor [options] <website-url>

# Monitor multiple URLs
./updo monitor <url1> <url2> <url3>

# Using configuration file
./updo monitor --config <config-file>

# Alternative syntax using --url flag
./updo monitor --url <website-url> [options]

# Generate shell completions
./updo completion bash > updo_completion.bash
```

### Docker

You can run Updo using Docker:

```console
# Build Docker image from locally cloned repo
docker build -t updo .
# ... or build straight from repo URL (no cloning needed):
docker build -t updo https://github.com/Owloops/updo.git

# And now you can run Updo from the built image:
docker run -it updo monitor [options] <website-url>
# Or with the --url flag:
docker run -it updo monitor --url <website-url> [options]
```

### Options

- `-u, --url`: URL of the website to monitor
- `--urls`: Multiple URLs to monitor (comma-separated)
- `-C, --config`: Config file path (TOML format) for multi-target monitoring
- `-r, --refresh`: Refresh interval in seconds (default: 5)
- `-f, --should-fail`: Invert status code success (default: false)
- `-t, --timeout`: HTTP request timeout in seconds (default: 10)
- `-l, --follow-redirects`: Follow redirects (default: true)
- `-s, --skip-ssl`: Skip SSL certificate verification (default: false)
- `-a, --assert-text`: Text to assert in the response body
- `-n, --receive-alert`: Enable alert notifications (default: true)
- `--simple`: Use simple output instead of TUI
- `-H, --header`: HTTP header to send (can be used multiple times, format: 'Header-Name: value')
- `-X, --request`: HTTP request method to use (default: GET)
- `-d, --data`: HTTP request body data
- `--log`: Output structured logs in JSON format (includes requests, responses, and metrics)
- `-c, --count`: Number of checks to perform (0 = infinite, applies per target)
- `--only`: Only monitor specific targets (by name or URL, comma-separated)
- `--skip`: Skip specific targets (by name or URL, comma-separated)
- `-h, --help`: Display help message

### Examples

```bash
# Basic monitoring with defaults (URL as argument)
./updo monitor https://example.com

# Alternative syntax using --url flag
./updo monitor --url https://example.com

# Root command with --url flag (implicit monitor command)
./updo --url https://example.com

# Set custom refresh and timeout
./updo monitor -r 10 -t 5 https://example.com

# Use simple mode with a set number of checks
./updo monitor --simple -c 10 https://example.com

# Simple mode 
./updo monitor --simple https://example.com

# Assert text in the response
./updo monitor -a "Welcome" https://example.com

# Output structured logs in JSON format
./updo monitor --log https://example.com

# Run 10 checks with structured logging and save to a file
./updo monitor --log --count=10 https://example.com > output.json

# Monitoring with custom HTTP headers (long form)
./updo monitor --header "Authorization: Bearer token123" --header "User-Agent: updo-test" https://example.com

# Monitoring with custom HTTP headers (short form)
./updo monitor -H "Authorization: Bearer token123" -H "Content-Type: application/json" https://example.com

# Using a different HTTP method (POST, PUT, DELETE, etc.)
./updo monitor -X POST -H "Content-Type: application/json" https://api.example.com/endpoint

# Sending a request with body data
./updo monitor -X POST -H "Content-Type: application/json" -d '{"name":"test"}' https://api.example.com/data

# Sending requests with body data and viewing structured logs
./updo monitor --log -X POST -H "Content-Type: application/json" -d '{"name":"test"}' https://api.example.com/data

# Multi-target monitoring examples
# Monitor multiple URLs from command line
./updo monitor https://google.com https://github.com https://cloudflare.com

# Using --urls flag
./updo monitor --urls="https://google.com,https://github.com"

# Using TOML configuration file
./updo monitor -C example-config.toml

# Multi-target with custom count
./updo monitor --count=5 https://google.com https://github.com

# Target filtering examples
# Only monitor specific targets from config file
./updo monitor --config example-config.toml --only Google,GitHub

# Skip specific targets 
./updo monitor --config example-config.toml --skip "slow-api,maintenance-site"

# Combine filtering with other options
./updo monitor --config example-config.toml --only Google --simple --count=5
```

## Configuration File

Updo supports TOML configuration files for complex monitoring setups. This is especially useful for monitoring multiple targets with different settings.

### Example Configuration (example-config.toml)

```toml
[global]
refresh_interval = 5
timeout = 10
follow_redirects = true
receive_alert = true
count = 0  # 0 means infinite
# Target filtering (optional)
only = ["Google", "GitHub"]  # Only monitor these targets
skip = ["slow-api"]          # Skip these targets

[[targets]]
url = "https://www.google.com"
name = "Google"
refresh_interval = 3  # Override global setting
assert_text = "Google"

[[targets]]
url = "https://api.github.com/repos/octocat/Hello-World"
name = "GitHub-API"
timeout = 15  # Override global timeout
method = "GET"
headers = ["User-Agent: updo-monitor/1.0", "Accept: application/vnd.github.v3+json"]

[[targets]]
url = "https://www.cloudflare.com"
name = "Cloudflare"
refresh_interval = 5
follow_redirects = false  # Override global setting
receive_alert = false  # Disable alerts for this target

[[targets]]
url = "https://critical-api.example.com/health"
name = "Critical API"
webhook_url = "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
webhook_headers = { "X-Custom-Header" = "updo-monitor" }
```

### Configuration Options

**Global settings:**

- `only`: Array of target names/URLs to monitor exclusively
- `skip`: Array of target names/URLs to skip
- Other global settings apply to all targets unless overridden

> **Note:** Command line `--only` and `--skip` flags override config file settings.

**Target settings:**
Each target can override global settings and supports:

- `url`: Target URL (required)
- `name`: Display name for the target
- `refresh_interval`: Check interval in seconds
- `timeout`: Request timeout in seconds
- `method`: HTTP method (GET, POST, etc.)
- `headers`: Array of HTTP headers
- `body`: Request body for POST/PUT requests
- `assert_text`: Text to find in response body
- `should_fail`: Invert success status codes
- `skip_ssl`: Skip SSL certificate verification
- `follow_redirects`: Follow HTTP redirects
- `receive_alert`: Enable desktop alerts
- `webhook_url`: Webhook endpoint for notifications
- `webhook_headers`: Custom headers for webhook requests (as map)

## Webhook Notifications

Updo can send webhook notifications when targets go up or down. This enables integration with various services like Slack, Discord, PagerDuty, or custom alerting systems.

### Webhook Payload

When a target status changes, Updo sends a JSON payload:

```json
{
  "event": "target_down",  // or "target_up"
  "target": "Critical API",
  "url": "https://api.example.com",
  "timestamp": "2024-01-01T12:00:00Z",
  "response_time_ms": 1500,
  "status_code": 500,
  "error": "Internal Server Error"  // only for down events
}
```

### Integration Examples

**Slack Webhook:**

```toml
[[targets]]
url = "https://api.example.com"
name = "Production API"
webhook_url = "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

**Custom Webhook with Headers:**

```toml
[[targets]]
url = "https://critical-service.example.com"
name = "Critical Service"
webhook_url = "https://alerts.internal.com/webhook"
webhook_headers = { 
  "Authorization" = "Bearer YOUR_TOKEN",
  "X-Service" = "updo-monitor"
}
```

## Structured Logging

The `--log` flag outputs JSON-formatted logs for programmatic consumption:

- **Check logs** (stdout): HTTP requests, responses, and timing information
- **Metrics logs** (stdout): Uptime, response time stats, success rate
- **Error logs** (stderr): Failures, warnings, and assertion results

Usage examples:

```bash
# All logs to one file
./updo monitor --log https://example.com > all.json 2>&1

# Metrics to one file, errors to another
./updo monitor --log https://example.com > metrics.json 2> errors.json

# Processing with jq
./updo monitor --log https://example.com | jq 'select(.type=="check") | .response_time_ms'
```

## Keyboard Shortcuts

When monitoring multiple targets:

- `↑`: Move to previous target
- `↓`: Move to next target
- `/`: Activate search mode to filter targets
- `ESC`: Exit search mode
- `Backspace`: Delete characters while searching
- `q` or `Ctrl+C`: Quit the application

## Mentions

- [awesome-readme](https://github.com/matiassingers/awesome-readme)
- [termui](https://github.com/gizak/termui)
- [Terminal Trove](https://terminaltrove.com/updo)
- [cobra](https://github.com/spf13/cobra)
- [bubbletea](https://github.com/charmbracelet/bubbletea)

## Contributing

Contributions to Updo are welcome! Feel free to create issues or submit pull requests.

## License

This project is licensed under the [MIT License](LICENSE).
