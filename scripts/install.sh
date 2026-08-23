#!/usr/bin/env sh
set -eu

GO_BIN="${GO_BIN:-go}"
PREFIX="${PREFIX:-$HOME/.local}"
BIN_DIR="$PREFIX/bin"

if ! command -v "$GO_BIN" >/dev/null 2>&1; then
  echo "go was not found. Set GO_BIN=/path/to/go or install Go first." >&2
  exit 1
fi

if [ ! -f main.go ]; then
  echo "main.go not found — attempting 'go install' fallback..." >&2
  if "$GO_BIN" install -ldflags "-X github.com/ryanrodrigues25200525-svg/Apple-music-cli/cmd.Version=dev" github.com/ryanrodrigues25200525-svg/Apple-music-cli@latest 2>/dev/null; then
    # go install places binary in GOPATH/bin or GOBIN; copy to BIN_DIR if different
    INSTALLED_BIN="$("$GO_BIN" env GOPATH)/bin/mu"
    if [ -f "$INSTALLED_BIN" ] && [ "$INSTALLED_BIN" != "$BIN_DIR/mu" ]; then
      mkdir -p "$BIN_DIR"
      cp "$INSTALLED_BIN" "$BIN_DIR/mu"
    fi
    echo "Installed mu to $BIN_DIR/mu (via go install)"
    echo "Run: mu doctor"
    exit 0
  fi
  echo "Failed. Run this script from the checked-out repo (./scripts/install.sh)" >&2
  echo "or manually: go install github.com/ryanrodrigues25200525-svg/Apple-music-cli@latest" >&2
  exit 1
fi

mkdir -p "$BIN_DIR"
"$GO_BIN" build -ldflags "-X github.com/ryanrodrigues25200525-svg/Apple-music-cli/cmd.Version=dev" -o "$BIN_DIR/mu" main.go

echo "Installed mu to $BIN_DIR/mu"
echo "Run: mu doctor"
