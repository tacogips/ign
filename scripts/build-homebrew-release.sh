#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

detect_target() {
  local kernel arch
  kernel="$(uname -s)"
  arch="$(uname -m)"

  case "$kernel:$arch" in
    Darwin:arm64) printf '%s\n' "darwin-arm64" ;;
    Darwin:x86_64) printf '%s\n' "darwin-x64" ;;
    Linux:aarch64 | Linux:arm64) printf '%s\n' "linux-arm64" ;;
    Linux:x86_64 | Linux:amd64) printf '%s\n' "linux-x64" ;;
    *)
      printf 'unsupported local platform: %s/%s\n' "$kernel" "$arch" >&2
      return 1
      ;;
  esac
}

usage() {
  cat <<'EOF'
Usage:
  scripts/build-homebrew-release.sh [target ...]

Targets:
  darwin-arm64  darwin-x64  linux-arm64  linux-x64

Environment:
  IGN_VERSION       Override package version used in archive names.
  IGN_RELEASE_DIR   Output directory. Defaults to dist/homebrew.

Examples:
  scripts/build-homebrew-release.sh
  scripts/build-homebrew-release.sh darwin-arm64 linux-x64
EOF
}

validate_target() {
  case "$1" in
    darwin-arm64 | darwin-x64 | linux-arm64 | linux-x64) ;;
    *)
      printf 'unsupported target: %s\n' "$1" >&2
      usage >&2
      return 1
      ;;
  esac
}

write_sha256() {
  local file
  file="$1"

  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file"
    return
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file"
    return
  fi

  printf 'missing checksum tool: expected shasum or sha256sum\n' >&2
  return 1
}

package_version() {
  if [[ -n "${IGN_VERSION:-}" ]]; then
    printf '%s\n' "$IGN_VERSION"
    return
  fi

  tr -d '[:space:]' < "$repo_root/internal/build/VERSION"
}

go_platform() {
  case "$1" in
    darwin-arm64) printf '%s\n' "darwin arm64" ;;
    darwin-x64) printf '%s\n' "darwin amd64" ;;
    linux-arm64) printf '%s\n' "linux arm64" ;;
    linux-x64) printf '%s\n' "linux amd64" ;;
  esac
}

build_target() {
  local version target release_dir work_dir archive binary goos goarch
  version="$1"
  target="$2"
  release_dir="$3"
  work_dir="$release_dir/work/ign-$version-$target"
  archive="$release_dir/ign-$version-$target.tar.gz"
  binary="$work_dir/bin/ign"
  read -r goos goarch <<< "$(go_platform "$target")"

  rm -rf "$work_dir" "$archive" "$archive.sha256"
  mkdir -p "$work_dir/bin"

  (
    cd "$repo_root"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build \
      -trimpath \
      -ldflags "-s -w -X github.com/tacogips/ign/internal/build.version=$version" \
      -o "$binary" \
      ./cmd/ign
  )

  chmod 0755 "$binary"
  cp "$repo_root/README.md" "$work_dir/README.md"

  tar -C "$work_dir" -czf "$archive" .
  write_sha256 "$archive" > "$archive.sha256"

  printf 'built %s\n' "$archive"
  cat "$archive.sha256"
}

main() {
  if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    usage
    return
  fi

  local version release_dir
  version="$(package_version)"
  release_dir="${IGN_RELEASE_DIR:-$repo_root/dist/homebrew}"
  mkdir -p "$release_dir"

  local -a targets
  if [[ "$#" -eq 0 ]]; then
    targets=("$(detect_target)")
  else
    targets=("$@")
  fi

  local target
  for target in "${targets[@]}"; do
    validate_target "$target"
    build_target "$version" "$target" "$release_dir"
  done

  printf '\nRender a formula after all platform archives exist:\n'
  printf '  scripts/render-homebrew-formula.sh %s\n' "$version"
}

main "$@"
