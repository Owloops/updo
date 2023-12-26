# Updo - Website Monitoring Tool

Updo is a command-line tool for monitoring website uptime and performance. It provides real-time metrics on website status, response time, SSL certificate expiry, and more, with alert notifications.

## Features

- Real-time monitoring of website uptime and performance
- Displays various metrics like uptime percentage, average response time, and SSL certificate expiry
- Desktop alert notifications for website status changes
- Customizable refresh intervals and request timeouts
- Supports HTTP and HTTPS, with options to skip SSL verification
- Assertion on response body content

## Installation

Make sure you have Go [installed](https://go.dev/doc/install) on your system.

You can install Updo by cloning the repository and building the binary:

```bash
git clone https://github.com/Owloops/updo.git
cd updo
go build
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
