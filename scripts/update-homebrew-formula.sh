#!/bin/bash
# Update Homebrew formula with SHA256 hashes from a GitHub release
# Usage: ./scripts/update-homebrew-formula.sh <version>
# Example: ./scripts/update-homebrew-formula.sh 0.1.0

set -euo pipefail

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.1.0"
    exit 1
fi

REPO="felixgeelhaar/preflight"
TAP_REPO="felixgeelhaar/homebrew-tap"
FORMULA_PATH="Formula/preflight.rb"

echo "Updating Homebrew formula for preflight v${VERSION}..."

# Temporary directory for downloads
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# Download and calculate SHA256 for each platform
declare -A SHAS

for PLATFORM in "darwin-amd64" "darwin-arm64" "linux-amd64" "linux-arm64"; do
    URL="https://github.com/${REPO}/releases/download/v${VERSION}/preflight-${PLATFORM}.tar.gz"
    echo "Downloading ${PLATFORM}..."

    if ! curl -sL "$URL" -o "$TMPDIR/preflight-${PLATFORM}.tar.gz"; then
        echo "Error: Failed to download $URL"
        echo "Make sure the release v${VERSION} exists with all platform artifacts."
        exit 1
    fi

    SHA=$(shasum -a 256 "$TMPDIR/preflight-${PLATFORM}.tar.gz" | cut -d' ' -f1)
    SHAS[$PLATFORM]=$SHA
    echo "  SHA256: $SHA"
done

echo ""
echo "All SHA256 hashes calculated successfully."
echo ""

# Generate the formula content
FORMULA_CONTENT="# Homebrew formula for Preflight
# To install: brew tap felixgeelhaar/tap && brew install preflight
class Preflight < Formula
  desc \"Deterministic workstation compiler\"
  homepage \"https://github.com/felixgeelhaar/preflight\"
  version \"${VERSION}\"
  license \"MIT\"

  on_macos do
    if Hardware::CPU.arm?
      url \"https://github.com/felixgeelhaar/preflight/releases/download/v#{version}/preflight-darwin-arm64.tar.gz\"
      sha256 \"${SHAS[darwin-arm64]}\"
    else
      url \"https://github.com/felixgeelhaar/preflight/releases/download/v#{version}/preflight-darwin-amd64.tar.gz\"
      sha256 \"${SHAS[darwin-amd64]}\"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url \"https://github.com/felixgeelhaar/preflight/releases/download/v#{version}/preflight-linux-arm64.tar.gz\"
      sha256 \"${SHAS[linux-arm64]}\"
    else
      url \"https://github.com/felixgeelhaar/preflight/releases/download/v#{version}/preflight-linux-amd64.tar.gz\"
      sha256 \"${SHAS[linux-amd64]}\"
    end
  end

  def install
    bin.install \"preflight\"
  end

  test do
    assert_match version.to_s, shell_output(\"#{bin}/preflight version\")
  end
end"

echo "Generated formula:"
echo "=================="
echo "$FORMULA_CONTENT"
echo "=================="
echo ""

# Check if we should push to GitHub
read -p "Push to ${TAP_REPO}? (y/n) " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Updating formula in ${TAP_REPO}..."

    # Get current file SHA (needed for update)
    CURRENT_SHA=$(gh api "repos/${TAP_REPO}/contents/${FORMULA_PATH}" --jq '.sha' 2>/dev/null || echo "")

    # Encode content to base64
    ENCODED_CONTENT=$(echo "$FORMULA_CONTENT" | base64)

    if [[ -n "$CURRENT_SHA" ]]; then
        # Update existing file
        gh api -X PUT "repos/${TAP_REPO}/contents/${FORMULA_PATH}" \
            -f message="Update preflight formula to v${VERSION}" \
            -f content="$ENCODED_CONTENT" \
            -f sha="$CURRENT_SHA" \
            --silent
    else
        # Create new file
        gh api -X PUT "repos/${TAP_REPO}/contents/${FORMULA_PATH}" \
            -f message="Add preflight formula v${VERSION}" \
            -f content="$ENCODED_CONTENT" \
            --silent
    fi

    echo "Formula updated successfully!"
    echo ""
    echo "Users can now install with:"
    echo "  brew tap felixgeelhaar/tap"
    echo "  brew install preflight"
else
    echo "Skipped push to GitHub."
    echo ""
    echo "To manually update, copy the formula above to:"
    echo "  ${TAP_REPO}/${FORMULA_PATH}"
fi
