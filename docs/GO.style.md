# Go Style Guide

Style conventions for Go code in this repository.

## Comments

Avoid comments. Code should be self-explanatory through clear naming and structure.

If logic isn't obvious, refactor into a well-named function instead of adding a comment.

Acceptable comments:
- Package-level doc comments (required for public packages)
- Exported function doc comments (one line, starts with function name)
- `TODO` and `FIXME` tags
- Non-obvious "why" explanations (never "what")

```go
// Package metrics provides Prometheus remote write functionality.
package metrics

// NewClient creates a metrics client with the given configuration.
func NewClient(cfg Config) *Client { ... }

// Drain body to enable connection reuse
_, _ = io.Copy(io.Discard, resp.Body)
```

## Structure

Organize files in this order:

1. Package declaration
2. Imports (stdlib, blank line, external, blank line, internal)
3. Constants
4. Package-level variables
5. Types (structs, interfaces)
6. Constructor functions
7. Methods
8. Helper functions

```go
package notifications

import (
    "fmt"
    "strings"

    "github.com/spf13/cobra"

    "github.com/myorg/myproject/config"
)

const (
    _maxRetries  = 3
    _httpTimeout = 10 * time.Second
)

var _globalClient *Client

type Client struct {
    config Config
    mu     sync.RWMutex
}

func NewClient(cfg Config) *Client {
    return &Client{config: cfg}
}

func (c *Client) Send(msg string) error {
    // ...
}

func formatMessage(msg string) string {
    // ...
}
```

## Project Layout

```
project/
├── main.go                 # Entry point
├── cmd/                    # CLI commands (Cobra)
│   ├── root/
│   │   └── root.go
│   └── subcommand/
│       └── subcommand.go
├── config/                 # Configuration loading
├── domain1/                # Feature packages
├── domain2/
├── utils/                  # Shared utilities
├── Makefile
├── go.mod
└── README.md
```

Package by domain, not by layer. Use `config/`, `metrics/`, `notifications/` — not `models/`, `services/`, `handlers/`.

No `internal/` unless you need to prevent imports. Flat is better than nested.

## Naming

### Packages

Lowercase, single word, no underscores:

```go
package metrics    // good
package httputil   // good
package http_util  // bad
package httpUtil   // bad
```

### Files

Lowercase with underscores. Pair with `_test.go`:

```
client.go
client_test.go
formatter_slack.go
formatter_discord.go
```

### Variables and Functions

| Scope | Convention | Example |
|-------|------------|---------|
| Exported | `PascalCase` | `LoadConfig`, `HTTPClient` |
| Unexported | `camelCase` | `parseHeaders`, `formatURL` |
| Package constants | `_camelCase` | `_maxRetries`, `_httpTimeout` |
| Loop variables | Single letter | `i`, `k`, `v` |
| Receivers | 1-2 letters | `c`, `cfg`, `ws` |

```go
const (
    _maxRetries    = 3
    _defaultRegion = "us-east-1"
)

func (c *Client) Send() error { ... }
func (ws *Workspace) Name() string { ... }
```

### Acronyms

Keep acronyms uppercase:

```go
userID    // not userId
httpURL   // not httpUrl
xmlParser // not XMLParser (start of name)
```

### Booleans

Use `is`, `has`, `should`, `can` prefixes:

```go
isValid
hasItems
shouldRetry
canConnect
```

## Types

### Structs

Group related fields. Exported fields first:

```go
type Client struct {
    Config     Config
    HTTPClient *http.Client

    mu       sync.RWMutex
    samples  []*Sample
    stopChan chan struct{}
}
```

### Interfaces

Keep interfaces small. Define where used, not where implemented:

```go
type Formatter interface {
    Format(payload Payload) ([]byte, error)
}

func Send(f Formatter, p Payload) error {
    data, err := f.Format(p)
    // ...
}
```

Accept interfaces, return structs:

```go
func NewService(store Storage) *Service { ... }  // good
func NewService(store Storage) Storage { ... }   // bad
```

### Receivers

Always use pointer receivers for consistency:

```go
func (c *Client) Send() error { ... }
func (c *Client) Config() Config { ... }  // even for getters
```

## Functions

### Constructors

Name constructors `New` or `NewX`:

```go
func NewClient(cfg Config) *Client {
    return &Client{
        config:   cfg,
        stopChan: make(chan struct{}),
    }
}
```

Initialize all fields explicitly. Don't rely on zero values for important state.

### Return Early

Guard clauses first, happy path last:

```go
func (c *Client) Process(data []byte) error {
    if data == nil {
        return errors.New("data is nil")
    }
    if len(data) == 0 {
        return nil
    }

    // main logic here
    return c.send(data)
}
```

### Named Returns

Avoid named returns except for documentation:

```go
// Good: documents what's returned
func ParseConfig(path string) (config *Config, err error) {
    // but don't use naked return
    return config, err
}

// Bad: naked return obscures control flow
func process() (result int, err error) {
    // ...
    return  // what's being returned?
}
```

## Error Handling

### Wrap with Context

Use `fmt.Errorf` with `%w`:

```go
if err != nil {
    return fmt.Errorf("failed to deploy to %s: %w", region, err)
}
```

### Check Errors Immediately

```go
result, err := doSomething()
if err != nil {
    return err
}
// use result
```

### Sentinel Errors

Use `errors.Is` and `errors.As`:

```go
if errors.Is(err, os.ErrNotExist) {
    // handle not found
}

var notFoundErr *NotFoundError
if errors.As(err, &notFoundErr) {
    // handle typed error
}
```

### Deferred Cleanup

Handle errors in deferred functions:

```go
defer func() {
    if err := resp.Body.Close(); err != nil {
        log.Printf("failed to close body: %v", err)
    }
}()
```

Or ignore explicitly:

```go
defer func() {
    _, _ = io.Copy(io.Discard, resp.Body)  // drain for connection reuse
    _ = resp.Body.Close()
}()
```

## Configuration

### Loading Pattern

Use Viper or koanf with struct tags:

```go
type Config struct {
    URL      string `mapstructure:"url"`
    Timeout  int    `mapstructure:"timeout"`
    Retries  int    `mapstructure:"retries"`
}

func LoadConfig(path string) (*Config, error) {
    viper.SetConfigFile(path)
    viper.SetDefault("timeout", 10)
    viper.SetDefault("retries", 3)

    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }

    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

### Configuration Structs

For function options, prefer simple structs over functional options:

```go
type Options struct {
    Timeout    time.Duration
    MaxRetries int
    Region     string
}

func Deploy(opts Options) error { ... }

// Usage
Deploy(Options{
    Timeout: 30 * time.Second,
    Region:  "us-east-1",
})
```

## Concurrency

### Goroutine Lifecycle

Always provide explicit start and stop:

```go
type Worker struct {
    wg       sync.WaitGroup
    stopChan chan struct{}
}

func (w *Worker) Start() {
    w.wg.Add(1)
    go w.run()
}

func (w *Worker) Stop() {
    close(w.stopChan)
    w.wg.Wait()
}

func (w *Worker) run() {
    defer w.wg.Done()
    for {
        select {
        case <-w.stopChan:
            return
        default:
            // work
        }
    }
}
```

### Mutex Patterns

Use `sync.RWMutex` when reads dominate:

```go
type Cache struct {
    mu    sync.RWMutex
    items map[string]Item
}

func (c *Cache) Get(key string) (Item, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    item, ok := c.items[key]
    return item, ok
}

func (c *Cache) Set(key string, item Item) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.items[key] = item
}
```

### Channel Patterns

Buffer channels when you know the count:

```go
results := make(chan Result, len(items))

for _, item := range items {
    go func(i Item) {
        results <- process(i)
    }(item)
}

for range items {
    result := <-results
    // handle
}
```

## Testing

### Table-Driven Tests

```go
func TestParseURL(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:  "valid https",
            input: "https://example.com",
            want:  "https://example.com",
        },
        {
            name:  "adds https",
            input: "example.com",
            want:  "https://example.com",
        },
        {
            name:    "empty input",
            input:   "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseURL(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got %q, want %q", got, tt.want)
            }
        })
    }
}
```

### Test File Organization

Tests live in the same package:

```
config/
├── config.go
└── config_test.go
```

### Temporary Files

Clean up with defer:

```go
func TestLoadConfig(t *testing.T) {
    f, err := os.CreateTemp("", "config-*.toml")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(f.Name())

    // write test data and test
}
```

## CLI (Cobra)

### Command Structure

```go
var deployCmd = &cobra.Command{
    Use:   "deploy",
    Short: "Deploy to production",
    RunE: func(cmd *cobra.Command, args []string) error {
        // return error, don't os.Exit
        return deploy(cfg)
    },
}

func init() {
    deployCmd.Flags().StringVarP(&region, "region", "r", "us-east-1", "AWS region")
    deployCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simulate deployment")
}
```

### Flag Defaults as Constants

```go
const (
    _defaultRegion  = "us-east-1"
    _defaultTimeout = 30
)

func init() {
    cmd.Flags().StringVarP(&region, "region", "r", _defaultRegion, "AWS region")
    cmd.Flags().IntVar(&timeout, "timeout", _defaultTimeout, "Timeout in seconds")
}
```

## Build & Tooling

### Makefile

```makefile
.DEFAULT_GOAL := help

BINARY := myapp

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "%-15s %s\n", $$1, $$2}'

build: ## Build binary
	go build -ldflags="-s -w" -o $(BINARY) .

test: ## Run tests
	go test -v ./...

lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	gofmt -s -w .
	goimports -w .

check: lint test ## Run all checks
```

### Version Injection

```go
var (
    version = "dev"
    commit  = "unknown"
    date    = "unknown"
)

// Build with: go build -ldflags="-X main.version=1.0.0 -X main.commit=$(git rev-parse HEAD)"
```

### Embedded Files

```go
//go:embed templates/*
var templates embed.FS

//go:embed version.txt
var version string
```

## Formatting

Run before commit:

```bash
gofmt -s -w .
goimports -w .
```

Line length: aim for 100 characters, hard limit at 120.

## Patterns to Avoid

### Don't

```go
// Don't use init() for complex logic
func init() {
    db, err := connectDB()  // bad: can't handle errors
}

// Don't use package-level error variables
var ErrNotFound = errors.New("not found")  // use typed errors or inline

// Don't return interfaces
func NewStorage() Storage { ... }  // bad: return concrete type

// Don't use naked returns
func process() (x int, err error) {
    x = 42
    return  // unclear
}

// Don't ignore errors silently
result, _ := riskyOperation()  // bad unless truly intentional
```

### Do

```go
// Do use explicit initialization
func main() {
    db, err := connectDB()
    if err != nil {
        log.Fatal(err)
    }
}

// Do return concrete types
func NewStorage() *Storage { ... }

// Do be explicit
result, err := riskyOperation()
if err != nil {
    return err
}

// Do ignore intentionally with comment
_ = conn.Close()  // best-effort cleanup
```

## References

- <https://go.dev/doc/effective_go>
- <https://github.com/golang/go/wiki/CodeReviewComments>
- <https://google.github.io/styleguide/go/>
- <https://github.com/uber-go/guide/blob/master/style.md>
- <https://no-color.org>
