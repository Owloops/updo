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
- Windows (amd64, 386)  
- macOS (amd64, arm64)

## Pre-release Checklist

- [ ] All tests pass
- [ ] Code merged to main
- [ ] Dependencies updated (`go mod tidy`)
