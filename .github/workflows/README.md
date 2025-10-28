# GitHub Actions Workflows

## CI Workflow (`ci.yml`)

**Triggers:**
- Push to any branch
- Pull request to any branch
- Manual dispatch

**Jobs:**
1. **test** - Runs on Ubuntu
   - Installs Go dependencies
   - Runs tests
   - Runs linter
   - Builds for current platform

2. **cross-platform** - Matrix build
   - Linux (amd64, arm64)
   - Windows (amd64)
   - macOS (amd64, arm64)
   - Uploads artifacts

## Build and Release Workflow (`build.yml`)

**Triggers:**
- Push tags starting with 'v*'
- Manual dispatch

**Jobs:**
1. **test** - Same as CI test job
2. **build** - Full matrix build for all platforms
3. **verify-build** - Verifies all artifacts were created
4. **release** - Creates GitHub release with packages

## Workflow Summary

- **ci.yml**: Fast feedback for all branches and PRs
- **build.yml**: Full release process for tagged versions