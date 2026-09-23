#!/bin/sh
#
# Install the pinned developer tools used by the Makefile.
#
# The tools land in $TOOLS_DIR, which defaults to ./.artifacts/bin inside this
# repository and is git-ignored. Nothing outside $TOOLS_DIR is modified, and no
# credentials are read or written.
#
# golangci-lint is installed from the official release archive (with checksum
# verification) rather than `go install` because the pinned release requires
# Go >= 1.22 while go.mod still declares go 1.21. Downloading the release keeps
# the linter version independent of the local Go toolchain, which is also what
# golangci-lint-action does in CI.
#
# Usage:
#   scripts/install-tools.sh                 # install every tool
#   scripts/install-tools.sh golangci-lint   # install one tool
#
# Environment:
#   TOOLS_DIR               destination directory (default: <repo>/.artifacts/bin)
#   GOLANGCI_LINT_VERSION   pinned version (default: v1.62.2)
#
# Exit codes: 0 = installed or already up to date, 1 = failure (message on stderr).
set -eu

GOLANGCI_LINT_VERSION="${GOLANGCI_LINT_VERSION:-v1.62.2}"
# Release assets and `golangci-lint --version` use the bare version, while the
# git tag carries a "v" prefix (tag v1.62.2 -> golangci-lint-1.62.2-<platform>.tar.gz).
GOLANGCI_LINT_BARE_VERSION=${GOLANGCI_LINT_VERSION#v}

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
TOOLS_DIR="${TOOLS_DIR:-$repo_root/.artifacts/bin}"

die() {
	printf 'install-tools: %s\n' "$1" >&2
	exit 1
}

info() {
	printf 'install-tools: %s\n' "$1"
}

have() {
	command -v "$1" >/dev/null 2>&1
}

# download <url> <destination>
download() {
	if have curl; then
		curl -fsSL --retry 3 --retry-delay 2 -o "$2" "$1"
	elif have wget; then
		wget -q -O "$2" "$1"
	else
		die "neither curl nor wget is available; cannot download $1"
	fi
}

# sha256 <file> -> prints the hex digest
sha256_of() {
	if have sha256sum; then
		sha256sum "$1" | awk '{print $1}'
	elif have shasum; then
		shasum -a 256 "$1" | awk '{print $1}'
	else
		die "neither sha256sum nor shasum is available; cannot verify $1"
	fi
}

# release_platform -> prints the golangci-lint release suffix, e.g. darwin-arm64
release_platform() {
	os=$(uname -s)
	arch=$(uname -m)

	case "$os" in
	Darwin) os=darwin ;;
	Linux) os=linux ;;
	FreeBSD) os=freebsd ;;
	*) die "unsupported OS '$os'; install golangci-lint $GOLANGCI_LINT_VERSION manually" ;;
	esac

	case "$arch" in
	x86_64 | amd64) arch=amd64 ;;
	arm64 | aarch64) arch=arm64 ;;
	armv7l | armv7) arch=armv7 ;;
	armv6l | armv6) arch=armv6 ;;
	i386 | i486 | i586 | i686) arch=386 ;;
	*) die "unsupported architecture '$arch'; install golangci-lint $GOLANGCI_LINT_VERSION manually" ;;
	esac

	printf '%s-%s' "$os" "$arch"
}

install_golangci_lint() {
	target="$TOOLS_DIR/golangci-lint"

	if [ -x "$target" ] && "$target" --version 2>/dev/null | grep -q "$GOLANGCI_LINT_BARE_VERSION"; then
		info "golangci-lint $GOLANGCI_LINT_VERSION already present in $TOOLS_DIR"
		return 0
	fi

	platform=$(release_platform)
	base="https://github.com/golangci/golangci-lint/releases/download/$GOLANGCI_LINT_VERSION"
	archive="golangci-lint-$GOLANGCI_LINT_BARE_VERSION-$platform.tar.gz"

	tmp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t nexus-util-tools)
	# shellcheck disable=SC2064  # expand tmp_dir now, on purpose
	trap "rm -rf '$tmp_dir'" EXIT INT TERM

	info "downloading $archive"
	download "$base/$archive" "$tmp_dir/$archive"

	info "verifying checksum"
	download "$base/golangci-lint-$GOLANGCI_LINT_BARE_VERSION-checksums.txt" "$tmp_dir/checksums.txt"
	expected=$(awk -v name="$archive" '$2 == name {print $1}' "$tmp_dir/checksums.txt")
	[ -n "$expected" ] || die "no checksum published for $archive"
	actual=$(sha256_of "$tmp_dir/$archive")
	[ "$expected" = "$actual" ] || die "checksum mismatch for $archive (expected $expected, got $actual)"

	tar -xzf "$tmp_dir/$archive" -C "$tmp_dir" \
		"golangci-lint-$GOLANGCI_LINT_BARE_VERSION-$platform/golangci-lint"
	mkdir -p "$TOOLS_DIR"
	mv "$tmp_dir/golangci-lint-$GOLANGCI_LINT_BARE_VERSION-$platform/golangci-lint" "$target"
	chmod +x "$target"

	info "installed $("$target" --version)"
}

main() {
	if [ "$#" -gt 0 ]; then
		tools="$*"
	else
		tools="golangci-lint"
	fi

	for tool in $tools; do
		case "$tool" in
		golangci-lint) install_golangci_lint ;;
		*) die "unknown tool '$tool' (known: golangci-lint)" ;;
		esac
	done
}

main "$@"
