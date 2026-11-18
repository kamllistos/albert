#!/bin/sh
# Installs the latest Albert release into ~/.local/bin.
#   curl -fsSL https://raw.githubusercontent.com/kamllistos/albert/main/install.sh | sh
set -eu

REPO="kamllistos/albert"
API="${ALBERT_RELEASE_API:-https://api.github.com/repos/$REPO/releases/latest}"
BIN_DIR="$HOME/.local/bin"

die() {
	printf 'Albert install failed: %s\n' "$1" >&2
	exit 1
}

case "$(uname -s)" in
Darwin)
	os=darwin
	macos="$(sw_vers -productVersion)"
	[ "${macos%%.*}" -ge 12 ] || die "Albert needs macOS 12 (Monterey) or later; this Mac has macOS $macos."
	;;
Linux) os=linux ;;
*) die "Albert runs on macOS and Linux only." ;;
esac

case "$(uname -m)" in
arm64 | aarch64) arch=arm64 ;;
x86_64 | amd64) arch=amd64 ;;
*) die "this processor ($(uname -m)) isn't supported. Albert supports 64-bit ARM and Intel/AMD." ;;
esac

command -v curl >/dev/null 2>&1 || die "curl is needed but wasn't found."
if command -v shasum >/dev/null 2>&1; then
	sha256() { shasum -a 256 -c check.txt; }
elif command -v sha256sum >/dev/null 2>&1; then
	sha256() { sha256sum -c check.txt; }
else
	die "shasum or sha256sum is needed but neither was found."
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM HUP

archive="albert_${os}_${arch}.tar.gz"

printf 'Looking up the latest Albert release...\n'
status="$(curl -sSL -o "$tmp/release.json" -w '%{http_code}' "$API")" ||
	die "couldn't reach GitHub. Check your internet connection and try again."
case "$status" in
200) ;;
404) die "no Albert release was found. There may not be a published release yet." ;;
*) die "GitHub answered with status $status. Try again in a few minutes." ;;
esac

# Pick a download URL out of the release JSON by file name.
asset_url() {
	grep '"browser_download_url"' "$tmp/release.json" |
		sed -n 's/.*"browser_download_url": *"\([^"]*\)".*/\1/p' |
		grep "/$1\$" | head -n 1
}

archive_url="$(asset_url "$archive" || true)"
sums_url="$(asset_url checksums.txt || true)"
[ -n "$archive_url" ] || die "the latest release has no download for this Mac ($archive)."
[ -n "$sums_url" ] || die "the latest release has no checksums.txt, so the download can't be verified."

printf 'Downloading %s...\n' "$archive"
curl -fsSL -o "$tmp/$archive" "$archive_url" || die "couldn't download $archive."
curl -fsSL -o "$tmp/checksums.txt" "$sums_url" || die "couldn't download checksums.txt."

printf 'Checking the download...\n'
grep "  $archive\$" "$tmp/checksums.txt" >"$tmp/check.txt" ||
	die "checksums.txt has no entry for $archive."
(cd "$tmp" && sha256 >/dev/null 2>&1) ||
	die "the download doesn't match its checksum, so it was not installed."

tar -xzf "$tmp/$archive" -C "$tmp" albert || die "couldn't unpack $archive."
mkdir -p "$BIN_DIR"
mv -f "$tmp/albert" "$BIN_DIR/albert"
chmod 755 "$BIN_DIR/albert"

printf 'Installed %s\n' "$BIN_DIR/albert"

case ":$PATH:" in
*":$BIN_DIR:"*) ;;
*)
	printf '\n%s is not on your PATH yet. Add it by running:\n' "$BIN_DIR"
	# shellcheck disable=SC2016
	rc="$HOME/.zshrc"
	[ "$os" = linux ] && rc="$HOME/.bashrc"
	case "${SHELL:-}" in */bash) rc="$HOME/.bashrc" ;; */zsh) rc="$HOME/.zshrc" ;; esac
	printf '  echo '\''export PATH="$HOME/.local/bin:$PATH"'\'' >> %s\n' "$rc"
	printf 'then open a new terminal window.\n\n'
	;;
esac

printf 'Run: albert up\n'
