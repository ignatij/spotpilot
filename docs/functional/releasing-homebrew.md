# Releasing Spotpilot and Installing via Homebrew

## Background: what a Homebrew tap is

Homebrew installs software from "formulae" — small Ruby files that describe
where to download a binary and how to install it.  By default Homebrew only
looks in its own official repository.  A **tap** lets anyone publish extra
formulae from their own GitHub repository.

The naming rule is rigid:

- GitHub repository name must be `homebrew-<anything>`.
- Users add it with `brew tap <owner>/<anything>`.
- Users install from it with `brew install <owner>/<anything>/<formula>`.

For Spotpilot:

| Piece | Value |
|---|---|
| Tap repository on GitHub | `ignatij/homebrew-spotpilot` |
| Add tap | `brew tap ignatij/spotpilot` |
| Install | `brew install ignatij/spotpilot/spotpilot` |

Homebrew itself never touches your main code repository.  GoReleaser
automatically pushes the generated formula file into the tap repository
every time you cut a release.

---

## Step 1 — Create the tap repository

1. Go to [https://github.com/new](https://github.com/new).
2. Set the repository name **exactly** to `homebrew-spotpilot`.
3. Owner must be `ignatij` (matches `.goreleaser.yml`).
4. Leave it public (Homebrew installs require publicly reachable formula and release assets).
5. Add a short description, for example: *Homebrew formula for spotpilot*.
6. Do **not** tick "Add a README" — GoReleaser will create the formula file.
7. Click **Create repository**.

---

## Step 2 — Create a Personal Access Token for the tap

GoReleaser needs write access to push the formula into the tap repository.
This is separate from the `GITHUB_TOKEN` that GitHub Actions generates
automatically (that token only has access to the current repository).

1. Go to **GitHub → Settings → Developer settings → Personal access tokens →
   Fine-grained tokens → Generate new token**.
2. Set:
   - **Token name**: `homebrew-tap-spotpilot`
   - **Expiration**: 1 year (or no expiration)
   - **Resource owner**: `ignatij`
   - **Repository access**: Only selected — pick `homebrew-spotpilot`
   - **Permissions → Contents**: Read and write
3. Click **Generate token** and copy the value immediately (it is shown once).

---

## Step 3 — Add the token as a GitHub Actions secret

1. Open the main `spotpilot` repository on GitHub.
2. Go to **Settings → Secrets and variables → Actions → New repository secret**.
3. Name: `HOMEBREW_TAP_GITHUB_TOKEN`
4. Value: paste the token from step 2.
5. Click **Add secret**.

Your release workflow already references this secret:

```yaml
env:
  GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

---

## Step 4 — Cut a release tag

A release is triggered by pushing a version tag.

```sh
git tag v0.1.0
git push origin v0.1.0
```

What happens automatically:

1. GitHub Actions sees the `v*` tag and starts the release workflow.
2. GoReleaser builds binaries for macOS (amd64 + arm64) and Linux
   (amd64 + arm64).
3. GoReleaser creates a GitHub Release with the binaries and `checksums.txt`.
4. GoReleaser pushes a generated `spotpilot.rb` formula into
   `ignatij/homebrew-spotpilot`.

The formula file looks roughly like:

```ruby
class Spotpilot < Formula
  desc "Control Spotify from the command line — built for AI agents"
  homepage "https://github.com/ignatij/spotpilot"
  url "https://github.com/ignatij/spotpilot/releases/download/v0.1.0/spotpilot_darwin_arm64.tar.gz"
  sha256 "..."
  ...
  def install
    bin.install "spotpilot"
  end
end
```

---

## Step 5 — Install and test

```sh
# Add your tap (only needed once per machine)
brew tap ignatij/spotpilot

# Install
brew install ignatij/spotpilot/spotpilot

# Verify
spotpilot version
```

Expected output:

```json
{"ok":true,"command":"version","state":"ok","message":"spotpilot v0.1.0 (abc1234, 2026-03-17)","result":{"version":"v0.1.0","commit":"abc1234","build_date":"2026-03-17"}}
```

---

## Upgrading later

```sh
brew upgrade ignatij/spotpilot/spotpilot
```

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| Download failed for `.../releases/download/.../spotpilot_darwin_arm64.tar.gz` | Main repo or release assets are not publicly accessible | Make `ignatij/spotpilot` public and ensure the release is published (not draft), then reinstall |
| Release workflow fails with `401` pushing formula | Wrong token or missing secret | Re-check step 3 |
| `brew install` says formula not found | Tap not added | Run `brew tap ignatij/spotpilot` first |
| Binary architecture mismatch on Apple Silicon | Old `arch` setting | Ensure `.goreleaser.yml` includes `arm64` under `goarch` |
| `spotpilot version` shows `dev` | Binary not built by GoReleaser | Use the brew-installed binary, not a local `go build` |
