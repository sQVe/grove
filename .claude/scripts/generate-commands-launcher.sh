#!/bin/bash
# OS Detection and Command Generation Launcher
# This script detects the operating system and runs the appropriate command generation script
#
# Usage: ./generate-commands-launcher.sh <spec-name>

set -e

if [ -z "$1" ]; then
    echo "Error: Spec name is required"
    echo "Usage: ./generate-commands-launcher.sh <spec-name>"
    exit 1
fi

SPEC_NAME="$1"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Detect operating system
detect_os() {
    case "$(uname -s)" in
        CYGWIN*|MINGW*|MSYS*)
            echo "windows"
            ;;
        Darwin*)
            echo "macos"
            ;;
        Linux*)
            echo "linux"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

OS_TYPE=$(detect_os)

echo "Detected OS: $OS_TYPE"
echo "Generating commands for spec: $SPEC_NAME"

case "$OS_TYPE" in
    "windows")
        if [ -f "$SCRIPT_DIR/generate-commands.bat" ]; then
            cmd.exe /c "$SCRIPT_DIR/generate-commands.bat" "$SPEC_NAME"
        else
            echo "Error: Windows script not found at $SCRIPT_DIR/generate-commands.bat"
            exit 1
        fi
        ;;
    "macos"|"linux")
        if [ -f "$SCRIPT_DIR/generate-commands.sh" ]; then
            chmod +x "$SCRIPT_DIR/generate-commands.sh"
            "$SCRIPT_DIR/generate-commands.sh" "$SPEC_NAME"
        else
            echo "Error: Unix script not found at $SCRIPT_DIR/generate-commands.sh"
            exit 1
        fi
        ;;
    *)
        echo "Error: Unsupported operating system: $OS_TYPE"
        echo "Supported platforms: Windows, macOS, Linux"
        exit 1
        ;;
esac
