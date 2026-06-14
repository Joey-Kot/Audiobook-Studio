#!/usr/bin/env sh
set -eu

missing=0

check_library() {
  name="$1"
  if ! ldconfig -p 2>/dev/null | grep -q "$name"; then
    echo "Missing runtime library: $name"
    missing=1
  fi
}

check_library "libgtk-3.so"
check_library "libwebkit2gtk-4.0.so"

if [ "$missing" -eq 0 ]; then
  echo "Linux GUI runtime dependencies look available."
  exit 0
fi

cat <<'EOF'

Install the runtime packages for your distribution, then start Audiobook-Studio again.

Common package names:
  Ubuntu/Debian: sudo apt-get install libgtk-3-0 libwebkit2gtk-4.0-37
  Fedora:        sudo dnf install gtk3 webkit2gtk4.0
  Arch:          sudo pacman -S gtk3 webkit2gtk-4.1

Package names can vary by distribution release.
EOF

exit 1

