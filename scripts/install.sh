#!/bin/sh
set -eu

REPO="${MAILJAIL_REPO:-vnedyalk0v/MailJail}"
VERSION="${MAILJAIL_VERSION:-latest}"
INSTALL_DIR="${MAILJAIL_INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="mailjail"

fail() {
	printf '%s\n' "error: $*" >&2
	exit 1
}

need_cmd() {
	command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

fetch_url() {
	url="$1"
	dest="$2"

	if command -v fetch >/dev/null 2>&1; then
		fetch -o "$dest" "$url"
		return 0
	fi

	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$url" -o "$dest"
		return 0
	fi

	fail "neither fetch nor curl is available"
}

detect_arch() {
	arch="$(uname -m)"
	case "$arch" in
		amd64|x86_64)
			printf 'amd64\n'
			;;
		aarch64|arm64)
			printf 'arm64\n'
			;;
		*)
			fail "unsupported architecture: $arch"
			;;
	esac
}

resolve_version() {
	if [ "$VERSION" != "latest" ]; then
		printf '%s\n' "$VERSION"
		return 0
	fi

	api_url="https://api.github.com/repos/${REPO}/releases/latest"
	response_file="$tmpdir/latest-release.json"
	fetch_url "$api_url" "$response_file"

	version="$(sed -n 's/^[[:space:]]*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' "$response_file" | head -n 1)"
	[ -n "$version" ] || fail "failed to resolve latest release tag from ${api_url}"
	printf '%s\n' "$version"
}

verify_checksum() {
	archive="$1"
	checksums="$2"

	expected="$(awk -v file="$(basename "$archive")" '$2 == file { print $1 }' "$checksums")"
	[ -n "$expected" ] || fail "no checksum entry found for $(basename "$archive")"

	if command -v sha256 >/dev/null 2>&1; then
		actual="$(sha256 -q "$archive")"
	elif command -v sha256sum >/dev/null 2>&1; then
		actual="$(sha256sum "$archive" | awk '{print $1}')"
	else
		fail "neither sha256 nor sha256sum is available"
	fi

	[ "$expected" = "$actual" ] || fail "checksum mismatch for $(basename "$archive")"
}

need_cmd uname
need_cmd tar
need_cmd install
need_cmd awk
need_cmd sed

[ "$(uname -s)" = "FreeBSD" ] || fail "this installer is intended for FreeBSD hosts"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT INT TERM

arch="$(detect_arch)"
version="$(resolve_version)"
artifact="mailjail_${version}_freebsd_${arch}.tar.gz"
checksums="mailjail_${version}_checksums.txt"
base_url="https://github.com/${REPO}/releases/download/${version}"

archive_path="${tmpdir}/${artifact}"
checksums_path="${tmpdir}/${checksums}"

printf '%s\n' "Installing ${BINARY_NAME} ${version} for freebsd/${arch}"
fetch_url "${base_url}/${artifact}" "$archive_path"
fetch_url "${base_url}/${checksums}" "$checksums_path"
verify_checksum "$archive_path" "$checksums_path"

tar -xzf "$archive_path" -C "$tmpdir"
binary_path="${tmpdir}/mailjail_${version}_freebsd_${arch}"
[ -f "$binary_path" ] || fail "extracted binary not found: $binary_path"

install -d "$INSTALL_DIR"
install -m 0755 "$binary_path" "${INSTALL_DIR}/${BINARY_NAME}"

printf '%s\n' "Installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"
printf '%s\n' "Run '${BINARY_NAME} version' to verify the installation."
