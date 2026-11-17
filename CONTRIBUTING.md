# Contributing to updo

Thanks for taking the time to contribute. This page covers the practical side: how to build, how to test, and what makes a pull request easy to merge.

## Before you start

Small fixes can go straight to a pull request. For anything larger, open an issue first so we can agree on the approach before you write code. Questions and ideas are welcome in Discussions.

## Setting up

You need Go 1.27 or newer. Build the binary from the repository root:

```bash
go build -o updo .
```

If you change anything under `lambda/`, rebuild the embedded Lambda binary too:

```bash
make build-lambda
```

That regenerates `aws/bootstrap.zip`, which is committed and reproducible, so a rebuild with no source change leaves the tree clean.

## Testing

Run the test suite before pushing:

```bash
go test ./...
```

Formatting and lint are enforced in CI:

```bash
gofmt -l .
golangci-lint run
```

The `golangci-lint` installed on your machine can be out of step with the Go toolchain and fail to typecheck. If that happens, run it through the toolchain instead, which is what CI effectively does: `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run`.

If you touch `tui/` or `widgets/`, run it in a real terminal, and in tmux as well if you can, since terminal detection is where most surprises live.

## Code style and commits

The style guide lives in [docs/GO.style.md](docs/GO.style.md) and covers naming, file structure, error handling, comments and testing patterns.

Commit messages follow `type(scope): description`, lower case, one line, for example `fix(net): close connections after each check`. Unrelated changes belong in separate commits, so a reviewer can read them in isolation.

## Pull requests

Keep them small and focused. A pull request that does one thing gets reviewed in minutes, one that does five tends to sit. Make sure CI is green, fill in the pull request template, and explain why the change is needed rather than restating the diff.

## AI assistance

Using AI tools is fine. Say so in the pull request, and make sure you understand and can maintain what you submit. Please don't send bulk generated changes or issues you haven't read yourself.
