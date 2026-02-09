#!/bin/sh
set -eu

REPO="vchilikov/takeout-fix"
API_URL="https://api.github.com/repos/${REPO}/releases/latest"
CWD=$(pwd)
OS=""
ARCH=""
TAG=""
ASSET_NAME=""
TMP_DIR=""
ASSET_PATH=""
CHECKSUM_PATH=""
RUN_BIN=""

log() {
	printf '%s\n' "$*"
}

fail() {
	printf 'Error: %s\n' "$*" >&2
	exit 1
}

cleanup() {
	if [ -n "$RUN_BIN" ] && [ -f "$RUN_BIN" ]; then
		rm -f "$RUN_BIN"
	fi
	if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
		rm -rf "$TMP_DIR"
	fi
}

command_exists() {
	command -v "$1" >/dev/null 2>&1
}

require_cmd() {
	if ! command_exists "$1"; then
		fail "required command not found: $1"
	fi
}

install_exiftool_macos() {
	if ! command_exists brew; then
		fail "exiftool is missing and Homebrew is not installed. Install exiftool manually and rerun."
	fi
	brew install exiftool
}

install_exiftool_linux() {
	if command_exists apt-get; then
		if [ "$(id -u)" -eq 0 ]; then
			apt-get install -y libimage-exiftool-perl
		elif command_exists sudo; then
			sudo apt-get install -y libimage-exiftool-perl
		else
			fail "sudo is required to install exiftool via apt-get."
		fi
		return
	fi

	if command_exists dnf; then
		if [ "$(id -u)" -eq 0 ]; then
			dnf install -y perl-Image-ExifTool
		elif command_exists sudo; then
			sudo dnf install -y perl-Image-ExifTool
		else
			fail "sudo is required to install exiftool via dnf."
		fi
		return
	fi

	if command_exists pacman; then
		if [ "$(id -u)" -eq 0 ]; then
			pacman -S --noconfirm perl-image-exiftool
		elif command_exists sudo; then
			sudo pacman -S --noconfirm perl-image-exiftool
		else
			fail "sudo is required to install exiftool via pacman."
		fi
		return
	fi

	fail "no supported package manager found (apt-get, dnf, pacman). Install exiftool manually and rerun."
}

ensure_exiftool() {
	if command_exists exiftool; then
		return
	fi

	log "exiftool not found in PATH. Installing..."
	case "$OS" in
		darwin)
			install_exiftool_macos
			;;
		linux)
			install_exiftool_linux
			;;
		*)
			fail "unsupported OS for automatic exiftool installation: $OS"
			;;
	esac

	if ! command_exists exiftool; then
		fail "exiftool installation did not make it available in PATH."
	fi
}

detect_platform() {
	uname_s=$(uname -s)
	case "$uname_s" in
		Darwin)
			OS="darwin"
			;;
		Linux)
			OS="linux"
			;;
		*)
			fail "unsupported OS: $uname_s"
			;;
	esac

	uname_m=$(uname -m)
	case "$uname_m" in
		x86_64 | amd64)
			ARCH="amd64"
			;;
		arm64 | aarch64)
			ARCH="arm64"
			;;
		*)
			fail "unsupported architecture: $uname_m"
			;;
	esac
}

resolve_tag() {
	release_json=$(curl -fsSL \
		-H "Accept: application/vnd.github+json" \
		-H "User-Agent: takeoutfix-installer" \
		"$API_URL")

	TAG=$(printf '%s\n' "$release_json" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)
	if [ -z "$TAG" ]; then
		fail "failed to resolve latest release tag from GitHub API."
	fi
}

download_assets() {
	ASSET_NAME="takeoutfix_${TAG}_${OS}_${ARCH}.tar.gz"
	checksums_name="checksums.txt"
	base_url="https://github.com/${REPO}/releases/download/${TAG}"

	require_cmd mktemp
	TMP_DIR=$(mktemp -d "${CWD}/.takeoutfix-installer.XXXXXX")
	ASSET_PATH="${TMP_DIR}/${ASSET_NAME}"
	CHECKSUM_PATH="${TMP_DIR}/${checksums_name}"

	log "Downloading ${ASSET_NAME}..."
	curl -fsSL -o "$ASSET_PATH" "${base_url}/${ASSET_NAME}" || fail "failed to download ${ASSET_NAME}"
	curl -fsSL -o "$CHECKSUM_PATH" "${base_url}/${checksums_name}" || fail "failed to download ${checksums_name}"
}

sha256_file() {
	file="$1"
	if command_exists sha256sum; then
		sha256sum "$file" | awk '{print $1}'
		return
	fi
	if command_exists shasum; then
		shasum -a 256 "$file" | awk '{print $1}'
		return
	fi
	fail "sha256 tool not found. Install sha256sum or shasum."
}

verify_checksum() {
	expected=$(awk -v asset="$ASSET_NAME" '
		{
			hash = $1
			name = $2
			if (name == "") {
				next
			}
			if (substr(name, 1, 1) == "*") {
				name = substr(name, 2)
			}
			if (name == asset) {
				print tolower(hash)
				exit
			}
		}
	' "$CHECKSUM_PATH")
	if [ -z "$expected" ]; then
		fail "checksum entry for ${ASSET_NAME} not found."
	fi

	actual=$(sha256_file "$ASSET_PATH" | tr 'A-F' 'a-f')
	if [ "$actual" != "$expected" ]; then
		fail "checksum mismatch for ${ASSET_NAME}."
	fi
}

extract_binary() {
	if [ ! -f "$ASSET_PATH" ]; then
		fail "archive not found: $ASSET_PATH"
	fi

	case "$ASSET_NAME" in
		*.tar.gz)
			require_cmd tar
			tar -xzf "$ASSET_PATH" -C "$TMP_DIR"
			;;
		*.zip)
			require_cmd unzip
			unzip -q "$ASSET_PATH" -d "$TMP_DIR"
			;;
		*)
			fail "unsupported archive extension: $ASSET_NAME"
			;;
	esac

	src_bin="${TMP_DIR}/takeoutfix"
	if [ ! -f "$src_bin" ]; then
		src_bin=$(find "$TMP_DIR" -type f -name takeoutfix -print | head -n 1 || true)
	fi
	if [ -z "$src_bin" ] || [ ! -f "$src_bin" ]; then
		fail "takeoutfix binary was not found in downloaded archive."
	fi

	RUN_BIN="${CWD}/.takeoutfix-run-$$-$(date +%s)"
	cp "$src_bin" "$RUN_BIN"
	chmod +x "$RUN_BIN"
}

run_takeoutfix() {
	log "Running TakeoutFix in: ${CWD}"
	set +e
	"$RUN_BIN" --workdir "$CWD"
	status=$?
	set -e
	return "$status"
}

trap cleanup EXIT INT TERM

require_cmd curl
detect_platform
ensure_exiftool
resolve_tag
download_assets
verify_checksum
extract_binary
run_takeoutfix
