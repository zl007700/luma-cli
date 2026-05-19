#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

version="local"
install=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --install|install)
      install=1
      ;;
    *)
      version="$1"
      ;;
  esac
  shift
done

out_dir="build/local"
out="$out_dir/luma-cli"
ldflags="-X github.com/luma-cli/lumer-cli/internal/commands.version=$version"

mkdir -p "$out_dir"

echo "Building luma-cli $version..."
go build -ldflags "$ldflags" -o "$out" .
chmod +x "$out"

echo "Built: $(pwd)/$out"

if [[ "$install" -eq 1 ]]; then
  npm_root="$(npm root -g)"
  target_dir="$npm_root/@lumageo/luma-cli/bin"
  target="$target_dir/luma-cli"

  mkdir -p "$target_dir"
  cp "$out" "$target"
  chmod +x "$target"

  echo "Installed local binary: $target"
  echo "Try: luma-cli version"
fi
