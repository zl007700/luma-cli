#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

version="local"

# Auto-derive version from the most recent git tag (e.g. "0.0.19-11-g654a5b7")
# so the output reflects how far this build is from the last release. The
# leading "v" is stripped to match the package.json convention. Falls back
# silently to "local" when not in a git checkout or no annotated tags exist;
# any positional argument below overrides this.
if desc="$(git describe --tags --long 2>/dev/null)"; then
  version="${desc#v}"
fi

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

# Capture build metadata so the resulting --version output reflects the actual
# source. Falls back to "unknown" when git is unavailable or the working tree
# is not a checkout (e.g. a tarball extraction).
commit="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
build_date="$(date -u +%Y-%m-%d)"

ldflags="-X github.com/luma-cli/lumer-cli/internal/commands.version=$version -X github.com/luma-cli/lumer-cli/internal/commands.commit=$commit -X github.com/luma-cli/lumer-cli/internal/commands.buildDate=$build_date"

mkdir -p "$out_dir"

echo "Building luma-cli $version (commit $commit, built $build_date)..."
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
