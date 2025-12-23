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

# Remove 'v' prefix if present
VERSION="${VERSION#v}"

REPO="felixgeelhaar/preflight"
TAP_REPO="felixgeelhaar/homebrew-tap"
FORMULA_PATH="Formula/preflight.rb"

echo "Updating Homebrew formula for preflight v${VERSION}..."

# Temporary directory for downloads
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# Download and calculate SHA256 for each platform
echo "Downloading darwin-amd64..."
curl -sL "https://github.com/${REPO}/releases/download/v${VERSION}/preflight-darwin-amd64.tar.gz" -o "$TMPDIR/preflight-darwin-amd64.tar.gz"
SHA_DARWIN_AMD64=$(shasum -a 256 "$TMPDIR/preflight-darwin-amd64.tar.gz" | cut -d' ' -f1)
echo "  SHA256: $SHA_DARWIN_AMD64"

echo "Downloading darwin-arm64..."
curl -sL "https://github.com/${REPO}/releases/download/v${VERSION}/preflight-darwin-arm64.tar.gz" -o "$TMPDIR/preflight-darwin-arm64.tar.gz"
SHA_DARWIN_ARM64=$(shasum -a 256 "$TMPDIR/preflight-darwin-arm64.tar.gz" | cut -d' ' -f1)
echo "  SHA256: $SHA_DARWIN_ARM64"

echo "Downloading linux-amd64..."
curl -sL "https://github.com/${REPO}/releases/download/v${VERSION}/preflight-linux-amd64.tar.gz" -o "$TMPDIR/preflight-linux-amd64.tar.gz"
SHA_LINUX_AMD64=$(shasum -a 256 "$TMPDIR/preflight-linux-amd64.tar.gz" | cut -d' ' -f1)
echo "  SHA256: $SHA_LINUX_AMD64"

echo "Downloading linux-arm64..."
curl -sL "https://github.com/${REPO}/releases/download/v${VERSION}/preflight-linux-arm64.tar.gz" -o "$TMPDIR/preflight-linux-arm64.tar.gz"
SHA_LINUX_ARM64=$(shasum -a 256 "$TMPDIR/preflight-linux-arm64.tar.gz" | cut -d' ' -f1)
echo "  SHA256: $SHA_LINUX_ARM64"

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
      sha256 \"${SHA_DARWIN_ARM64}\"
    else
      url \"https://github.com/felixgeelhaar/preflight/releases/download/v#{version}/preflight-darwin-amd64.tar.gz\"
      sha256 \"${SHA_DARWIN_AMD64}\"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url \"https://github.com/felixgeelhaar/preflight/releases/download/v#{version}/preflight-linux-arm64.tar.gz\"
      sha256 \"${SHA_LINUX_ARM64}\"
    else
      url \"https://github.com/felixgeelhaar/preflight/releases/download/v#{version}/preflight-linux-amd64.tar.gz\"
      sha256 \"${SHA_LINUX_AMD64}\"
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

# Check for non-interactive mode
if [[ "${NONINTERACTIVE:-}" == "1" ]] || [[ ! -t 0 ]]; then
    REPLY="y"
else
    read -p "Push to ${TAP_REPO}? (y/n) " -n 1 -r
    echo ""
fi

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
