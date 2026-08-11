# Homebrew Packaging

Homebrew releases install a standalone Go binary from GitHub Release assets.
The published archive contains `bin/ign`.

Build release archives:

```bash
scripts/build-homebrew-release.sh darwin-arm64 darwin-x64 linux-arm64 linux-x64
```

The command writes archives and checksum files under `dist/homebrew/`.

Create or update the GitHub release named `v<version>` with those archives, then
render the formula into the tap checkout:

```bash
scripts/render-homebrew-formula.sh <version> ../homebrew-tap/Formula/ign.rb
```

For the default mise task wrappers:

```bash
mise run build:homebrew -- darwin-arm64 darwin-x64 linux-arm64 linux-x64
mise run homebrew:formula -- <version>
mise run homebrew:tap-formula -- <version>
```

Verify from the tap checkout:

```bash
ruby -c Formula/ign.rb
brew audit --strict ign || brew audit --strict --formula ign
brew install tacogips/tap/ign
brew test tacogips/tap/ign
```
