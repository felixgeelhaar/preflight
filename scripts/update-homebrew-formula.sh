#!/bin/bash
# Update Homebrew formula with SHA256 hashes from a release
# Usage: ./scripts/update-homebrew-formula.sh v0.1.0

set -euo pipefail

VERSION="${1:-}"
FORMULA_PATH="homebrew/preflight.rb"
REPO="felixgeelhaar/preflight"

if [[ -z "$VERSION" ]]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.1.0"
    exit 1
fi

# Strip 'v' prefix for version number
VERSION_NUM="${VERSION#v}"

echo "Updating Homebrew formula for version $VERSION_NUM..."

# Download checksums file from release
CHECKSUMS_URL="https://github.com/$REPO/releases/download/$VERSION/checksums.txt"
echo "Fetching checksums from: $CHECKSUMS_URL"

CHECKSUMS=$(curl -sL "$CHECKSUMS_URL")
if [[ -z "$CHECKSUMS" ]]; then
    echo "Error: Could not fetch checksums from release"
    exit 1
fi

echo "Checksums:"
echo "$CHECKSUMS"
echo

# Extract SHA256 for each platform
get_sha256() {
    local pattern="$1"
    echo "$CHECKSUMS" | grep "$pattern" | awk '{print $1}'
}

SHA_DARWIN_ARM64=$(get_sha256 "darwin-arm64")
SHA_DARWIN_AMD64=$(get_sha256 "darwin-amd64")
SHA_LINUX_ARM64=$(get_sha256 "linux-arm64")
SHA_LINUX_AMD64=$(get_sha256 "linux-amd64")

echo "Updating formula..."
echo "  darwin-arm64: $SHA_DARWIN_ARM64"
echo "  darwin-amd64: $SHA_DARWIN_AMD64"
echo "  linux-arm64:  $SHA_LINUX_ARM64"
echo "  linux-amd64:  $SHA_LINUX_AMD64"

# Update version
sed -i '' "s/version \"[^\"]*\"/version \"$VERSION_NUM\"/" "$FORMULA_PATH"

# Update SHA256 hashes
sed -i '' "s/PLACEHOLDER_SHA256_DARWIN_ARM64/$SHA_DARWIN_ARM64/" "$FORMULA_PATH"
sed -i '' "s/PLACEHOLDER_SHA256_DARWIN_AMD64/$SHA_DARWIN_AMD64/" "$FORMULA_PATH"
sed -i '' "s/PLACEHOLDER_SHA256_LINUX_ARM64/$SHA_LINUX_ARM64/" "$FORMULA_PATH"
sed -i '' "s/PLACEHOLDER_SHA256_LINUX_AMD64/$SHA_LINUX_AMD64/" "$FORMULA_PATH"

# Also update any existing hashes (for subsequent releases)
if [[ -n "$SHA_DARWIN_ARM64" ]]; then
    # For updates after first release, replace the actual hashes
    sed -i '' "/darwin-arm64/{ n; s/sha256 \"[a-f0-9]*\"/sha256 \"$SHA_DARWIN_ARM64\"/; }" "$FORMULA_PATH" 2>/dev/null || true
fi

echo
echo "Formula updated successfully!"
echo
echo "Next steps:"
echo "1. Review changes: git diff $FORMULA_PATH"
echo "2. Copy to homebrew-tap repository"
echo "3. Commit and push to homebrew-tap"
