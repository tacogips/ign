# Homebrew Packaging

`ign` Homebrew releases install a standalone Go binary built with cross-compilation.
The published archive contains `bin/ign`. Homebrew does not need a runtime dependency
on Go.

Build release archives:

```bash
scripts/build-homebrew-release.sh darwin-arm64 darwin-x64 linux-arm64 linux-x64
```

The command writes archives and checksum files under `dist/homebrew/`:

```text
ign-<version>-darwin-arm64.tar.gz
ign-<version>-darwin-x64.tar.gz
ign-<version>-linux-arm64.tar.gz
ign-<version>-linux-x64.tar.gz
```

Create or update the GitHub release named `v<version>` with those archives:

```bash
gh release create "v<version>" \
  dist/homebrew/ign-<version>-darwin-arm64.tar.gz \
  dist/homebrew/ign-<version>-darwin-x64.tar.gz \
  dist/homebrew/ign-<version>-linux-arm64.tar.gz \
  dist/homebrew/ign-<version>-linux-x64.tar.gz \
  --repo tacogips/ign \
  --title "ign v<version>" \
  --notes ""
```

If the release already exists, upload or replace the assets with:

```bash
gh release upload "v<version>" \
  dist/homebrew/ign-<version>-darwin-arm64.tar.gz \
  dist/homebrew/ign-<version>-darwin-x64.tar.gz \
  dist/homebrew/ign-<version>-linux-arm64.tar.gz \
  dist/homebrew/ign-<version>-linux-x64.tar.gz \
  --repo tacogips/ign \
  --clobber
```

Then render the formula into the existing `tacogips/homebrew-tap` checkout:

```bash
scripts/render-homebrew-formula.sh <version> ../homebrew-tap/Formula/ign.rb
```

The Taskfile wrapper for that tap path is:

```bash
task homebrew:tap-formula -- <version>
```

For any other tap repository, run the render command from this repository and
write the generated formula into the tap's `Formula/ign.rb`.
Override `IGN_RELEASE_BASE_URL` when the archives are hosted somewhere
other than `https://github.com/tacogips/ign/releases/download/v<version>`.

Commit and push the tap change:

```bash
cd ../homebrew-tap
git add Formula/ign.rb README.md
git commit -m "chore: add ign formula"
git push origin main
```

After the tap commit is pushed, users can install with:

```bash
brew tap tacogips/tap
brew install ign
```

Smoke-test a local formula before upload by rendering into a temporary tap that
uses the local archive directory as its URL base:

```bash
brew tap-new local/ign-test
tap_root="$(brew --repository local/ign-test)"
IGN_RELEASE_BASE_URL="file://$PWD/dist/homebrew" \
  scripts/render-homebrew-formula.sh <version> "$tap_root/Formula/ign.rb"
brew install local/ign-test/ign
brew test local/ign-test/ign
brew uninstall ign
brew untap local/ign-test
```
