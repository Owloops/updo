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
- Displays various metrics like uptime percentage, average response time, and SSL certificate expiry
- Desktop alert notifications for website status changes
- Customizable refresh intervals and request timeouts
- Supports HTTP and HTTPS, with options to skip SSL verification
- Assertion on response body content

## Installation

Make sure you have Go [installed](https://go.dev/doc/install) on your system.

### Download executable binaries

You can download executable binaries from the latest release page.

> [![Latest Release](https://img.shields.io/github/v/release/Owloops/updo?style=flat-square)](https://github.com/Owloops/updo/releases/latest)

### Build from source

You can install Updo by cloning the repository and building the binary:

```bash
git clone https://github.com/Owloops/updo.git
cd updo
go build
```

Another way to install it if you have go in your machine just:

```sh
GOBIN="absolute_path_where_you_want_binaries_to_be_installed" go install github.com/sheepla/pingu@latest
```

## Usage

Run Updo using the following command:

```bash
./updo --url <website-url> [options]
```

### Options

- `--url`: URL of the website to monitor (required)
- `--refresh`: Refresh interval in seconds (default: 5)
- `--should-fail`: Invert status code success (default: false)
- `--timeout`: HTTP request timeout in seconds (default: 10)
- `--follow-redirects`: Follow redirects (default: true)
- `--skip-ssl`: Skip SSL certificate verification (default: false)
- `--assert-text`: Text to assert in the response body
- `--receive-alert`: Enable alert notifications (default: true)
- `--help`: Display help message

### Example

```bash
./updo --url https://example.com --refresh=10 --should-fail=false
```

## Keyboard Shortcuts

- `q` or `Ctrl+C`: Quit the application

## Contributing

Contributions to Updo are welcome! Feel free to create issues or submit pull requests.

## License

This project is licensed under the [MIT License](LICENSE).
