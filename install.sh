#!/bin/sh
set -e

# Grove installer
# Usage: curl -fsSL https://raw.githubusercontent.com/sQVe/grove/main/install.sh | sh

REPO="sQVe/grove"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

main() {
  os=$(detect_os)
  arch=$(detect_arch)

  if [ -z "${os}" ] || [ -z "${arch}" ]; then
    echo "Error: Unsupported platform: $(uname -s)/$(uname -m)" >&2
    exit 1
  fi

  version=$(get_latest_version)
  if [ -z "${version}" ]; then
    echo "Error: Could not determine latest version" >&2
    exit 1
  fi

  echo "Installing grove ${version} (${os}/${arch})..."

  tmpdir=$(mktemp -d)
  trap 'rm -rf "$tmpdir"' EXIT

  archive="grove_${version#v}_${os}_${arch}.tar.gz"
  url="https://github.com/${REPO}/releases/download/${version}/${archive}"
  checksums_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"

  echo "Downloading ${url}..."
  download "${url}" "${tmpdir}/${archive}"
  download "${checksums_url}" "${tmpdir}/checksums.txt"

  echo "Verifying checksum..."
  verify_checksum "${tmpdir}/${archive}" "${tmpdir}/checksums.txt" "${archive}"

  echo "Extracting..."
  tar -xzf "${tmpdir}/${archive}" -C "${tmpdir}"

  if [ ! -w "${INSTALL_DIR}" ]; then
    echo "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo install -m 755 "${tmpdir}/grove" "${INSTALL_DIR}/grove"
  else
    install -m 755 "${tmpdir}/grove" "${INSTALL_DIR}/grove"
  fi

  echo "Installed grove to ${INSTALL_DIR}/grove"
  echo "Run 'grove --help' to get started"
}

detect_os() {
  case "$(uname -s)" in
    Linux*) echo "linux" ;;
    Darwin*) echo "darwin" ;;
    *) echo "" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64 | amd64) echo "amd64" ;;
    arm64 | aarch64) echo "arm64" ;;
    *) echo "" ;;
  esac
}

get_latest_version() {
  if command -v curl > /dev/null 2>&1; then
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
  elif command -v wget > /dev/null 2>&1; then
    wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
  fi
}

download() {
  url="$1"
  output="$2"
  if command -v curl > /dev/null 2>&1; then
    curl -fsSL "${url}" -o "${output}"
  elif command -v wget > /dev/null 2>&1; then
    wget -q "${url}" -O "${output}"
  else
    echo "Error: curl or wget required" >&2
    exit 1
  fi
}

verify_checksum() {
  file="$1"
  checksums="$2"
  filename="$3"

  expected=$(grep "${filename}" "${checksums}" | awk '{print $1}')
  if [ -z "${expected}" ]; then
    echo "Error: Checksum not found for ${filename}" >&2
    exit 1
  fi

  if command -v sha256sum > /dev/null 2>&1; then
    actual=$(sha256sum "${file}" | awk '{print $1}')
  elif command -v shasum > /dev/null 2>&1; then
    actual=$(shasum -a 256 "${file}" | awk '{print $1}')
  else
    echo "Warning: Cannot verify checksum (sha256sum/shasum not found)" >&2
    return 0
  fi

  if [ "${expected}" != "${actual}" ]; then
    echo "Error: Checksum mismatch" >&2
    echo "  Expected: ${expected}" >&2
    echo "  Actual:   ${actual}" >&2
    exit 1
  fi
}

main
