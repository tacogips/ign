#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

usage() {
  cat <<'EOF'
Usage:
  scripts/render-homebrew-formula.sh <version> [output-file]

Reads archive checksums from:
  dist/homebrew/ign-<version>-<target>.tar.gz.sha256

Environment:
  IGN_RELEASE_DIR       Directory containing archives and .sha256 files.
  IGN_RELEASE_BASE_URL  Release URL base. Defaults to GitHub v<version>.

Example:
  scripts/build-homebrew-release.sh darwin-arm64 darwin-x64 linux-arm64 linux-x64
  scripts/render-homebrew-formula.sh <version> Formula/ign.rb
EOF
}

sha_for_target() {
  local version target release_dir sha_file
  version="$1"
  target="$2"
  release_dir="$3"
  sha_file="$release_dir/ign-$version-$target.tar.gz.sha256"

  if [[ ! -f "$sha_file" ]]; then
    printf 'missing checksum file: %s\n' "$sha_file" >&2
    return 1
  fi

  awk '{print $1}' "$sha_file"
}

main() {
  if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    usage
    return
  fi
  if [[ "${1:-}" == "" ]]; then
    usage
    return 2
  fi

  local version output release_dir release_base_url
  version="$1"
  output="${2:-$repo_root/Formula/ign.rb}"
  release_dir="${IGN_RELEASE_DIR:-$repo_root/dist/homebrew}"
  release_base_url="${IGN_RELEASE_BASE_URL:-https://github.com/tacogips/ign/releases/download/v$version}"

  local darwin_arm64_sha darwin_x64_sha linux_arm64_sha linux_x64_sha
  darwin_arm64_sha="$(sha_for_target "$version" darwin-arm64 "$release_dir")"
  darwin_x64_sha="$(sha_for_target "$version" darwin-x64 "$release_dir")"
  linux_arm64_sha="$(sha_for_target "$version" linux-arm64 "$release_dir")"
  linux_x64_sha="$(sha_for_target "$version" linux-x64 "$release_dir")"

  mkdir -p "$(dirname "$output")"
  cat > "$output" <<EOF
class Ign < Formula
  desc "Template-based code generation CLI tool"
  homepage "https://github.com/tacogips/ign"
  version "$version"
  license "MIT"

  livecheck do
    url :stable
    strategy :github_latest
  end

  on_macos do
    if Hardware::CPU.arm?
      url "$release_base_url/ign-$version-darwin-arm64.tar.gz"
      sha256 "$darwin_arm64_sha"
    else
      url "$release_base_url/ign-$version-darwin-x64.tar.gz"
      sha256 "$darwin_x64_sha"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "$release_base_url/ign-$version-linux-arm64.tar.gz"
      sha256 "$linux_arm64_sha"
    else
      url "$release_base_url/ign-$version-linux-x64.tar.gz"
      sha256 "$linux_x64_sha"
    end
  end

  def install
    bin.install "bin/ign"
  end

  test do
    assert_equal "$version", shell_output("#{bin}/ign version --short").strip
  end
end
EOF

  printf 'rendered %s\n' "$output"
}

main "$@"
