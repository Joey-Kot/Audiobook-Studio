#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_OS=windows TARGET_ARCH=amd64 "$SCRIPT_DIR/build-ffmpeg-static.sh"
