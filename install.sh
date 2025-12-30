#!/bin/sh
set -e

# Grove installer
# Usage: curl -fsSL https://raw.githubusercontent.com/sQVe/grove/main/install.sh | sh

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-}"

repo="sQVe/grove"

main() {
  case "${1:-}" in
    -h | --help)
      show_help
      exit 0
      ;;
  esac

  check_dependencies

  os=$(detect_os)
  arch=$(detect_arch)

  if [ -z "${os}" ] || [ -z "${arch}" ]; then
    echo "Error: Unsupported platform: $(uname -s)/$(uname -m)" >&2
    exit 1
  fi

  version=$(get_version)
  if [ -z "${version}" ]; then
    echo "Error: Could not determine latest version" >&2
    echo "  Try: VERSION=v1.0.0 ./install.sh" >&2
    echo "  Or check: https://github.com/${repo}/releases" >&2
    exit 1
  fi

  check_existing_installation "${version}"

  echo "Installing grove ${version} (${os}/${arch})..."

  tmpdir=$(mktemp -d)
  trap 'rm -rf "${tmpdir}"' EXIT INT TERM

  archive="grove_${version#v}_${os}_${arch}.tar.gz"
  url="https://github.com/${repo}/releases/download/${version}/${archive}"
  checksums_url="https://github.com/${repo}/releases/download/${version}/checksums.txt"

  echo "Downloading..."
  download "${url}" "${tmpdir}/${archive}"
  download "${checksums_url}" "${tmpdir}/checksums.txt"

  echo "Verifying checksum..."
  verify_checksum "${tmpdir}/${archive}" "${tmpdir}/checksums.txt" "${archive}"

  echo "Extracting..."
  if ! tar -xzf "${tmpdir}/${archive}" -C "${tmpdir}"; then
    echo "Error: Failed to extract archive" >&2
    exit 1
  fi

  if [ ! -f "${tmpdir}/grove" ]; then
    echo "Error: Binary not found in archive" >&2
    exit 1
  fi

  if [ ! -w "${INSTALL_DIR}" ]; then
    echo "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo install -m 755 "${tmpdir}/grove" "${INSTALL_DIR}/grove"
  else
    install -m 755 "${tmpdir}/grove" "${INSTALL_DIR}/grove"
  fi

  echo "Installed grove to ${INSTALL_DIR}/grove"

  case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
      echo ""
      echo "Warning: ${INSTALL_DIR} is not in your PATH"
      echo "  Add it: export PATH=\"\${PATH}:${INSTALL_DIR}\""
      ;;
  esac

  echo ""
  echo "Run 'grove --help' to get started"
}

show_help() {
  cat << 'EOF'
Grove Installer

Usage:
  curl -fsSL https://raw.githubusercontent.com/sQVe/grove/main/install.sh | sh

Options:
  -h, --help    Show this help

Environment Variables:
  VERSION       Install specific version (default: latest)
  INSTALL_DIR   Installation directory (default: /usr/local/bin)

Examples:
  # Install latest
  curl -fsSL https://... | sh

  # Install specific version
  VERSION=v1.0.0 curl -fsSL https://... | sh

  # Custom directory
  INSTALL_DIR=~/.local/bin curl -fsSL https://... | sh
EOF
}

check_dependencies() {
  missing=""
  for cmd in curl tar awk grep; do
    if ! command -v "${cmd}" > /dev/null 2>&1; then
      missing="${missing} ${cmd}"
    fi
  done

  if [ -n "${missing}" ]; then
    echo "Error: Missing required tools:${missing}" >&2
    exit 1
  fi
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

get_version() {
  if [ -n "${VERSION}" ]; then
    echo "${VERSION}"
    return
  fi

  curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" 2> /dev/null | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
}

check_existing_installation() {
  target_version="$1"

  if [ -x "${INSTALL_DIR}/grove" ]; then
    current=$("${INSTALL_DIR}/grove" --version 2> /dev/null | awk '{print $NF}' || echo "unknown")
    if [ "v${current}" = "${target_version}" ] || [ "${current}" = "${target_version}" ]; then
      echo "grove ${target_version} is already installed"
      exit 0
    fi
    echo "Upgrading grove ${current} -> ${target_version#v}"
  fi
}

download() {
  url="$1"
  output="$2"

  if [ -t 1 ]; then
    if ! curl -fL --progress-bar "${url}" -o "${output}"; then
      echo "Error: Failed to download ${url}" >&2
      exit 1
    fi
  else
    if ! curl -fsSL "${url}" -o "${output}"; then
      echo "Error: Failed to download ${url}" >&2
      exit 1
    fi
  fi

  if [ ! -s "${output}" ]; then
    echo "Error: Downloaded file is empty: ${url}" >&2
    exit 1
  fi
}

verify_checksum() {
  file="$1"
  checksums="$2"
  filename="$3"

  expected=$(grep -F "${filename}" "${checksums}" | awk '{print $1}')
  if [ -z "${expected}" ]; then
    echo "Error: Checksum not found for ${filename}" >&2
    exit 1
  fi

  if command -v sha256sum > /dev/null 2>&1; then
    actual=$(sha256sum "${file}" | awk '{print $1}')
  elif command -v shasum > /dev/null 2>&1; then
    actual=$(shasum -a 256 "${file}" | awk '{print $1}')
  elif command -v openssl > /dev/null 2>&1; then
    actual=$(openssl dgst -sha256 "${file}" | awk '{print $NF}')
  else
    echo "Error: Checksum verification requires sha256sum, shasum, or openssl" >&2
    exit 1
  fi

  if [ "${expected}" != "${actual}" ]; then
    echo "Error: Checksum mismatch" >&2
    echo "  Expected: ${expected}" >&2
    echo "  Actual:   ${actual}" >&2
    exit 1
  fi
}

main "$@"
