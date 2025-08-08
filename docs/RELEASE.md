# Release Process

Updo uses GitHub Actions with GoReleaser to automatically create releases when a new tag is pushed.

## Creating a Release

1. **Ensure main branch is ready**

   ```bash
   git pull origin main
   go test ./...
   go mod tidy
   ```

2. **Create and push a tag**

   ```bash
   # Create tag (semantic versioning: vMAJOR.MINOR.PATCH)
   git tag v0.1.3
   
   # Push tag to trigger release
   git push origin v0.1.3
   ```

3. **Automatic process**
   - GitHub Actions triggers on tag push
   - GoReleaser builds binaries for all platforms
   - Creates GitHub release with artifacts

## Files Involved

- `.github/workflows/release.yml` - GitHub Actions workflow
- `.goreleaser.yaml` - Build configuration

## Platforms Supported

- Linux (amd64, arm64)
- Windows (amd64, arm64)  
- macOS (amd64, arm64)

## Version Information

Updo now includes version information accessible via the `--version` flag:

```bash
$ updo --version
updo version v0.1.5 (commit: abcdef123456, built: 2023-05-21T12:00:00Z)
```

The version information is automatically injected during the build process using ldflags:

- `version`: The git tag (e.g., v0.1.5)
- `commit`: The git commit SHA
- `date`: The build timestamp

For local builds, use:

```bash
go build -ldflags="-X 'main.version=v1.0.0' -X 'main.commit=$(git rev-parse HEAD)' -X 'main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
```

## Pre-release Checklist

- [ ] All tests pass
- [ ] Code merged to main
- [ ] Dependencies updated (`go mod tidy`)
