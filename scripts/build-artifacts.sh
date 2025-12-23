#!/bin/bash
set -euo pipefail

# Build artifacts for release
# This script is called by relicta before creating a release

VERSION="${1:-$(git describe --tags --always --dirty)}"
COMMIT="${2:-$(git rev-parse --short HEAD)}"
BUILD_DATE="${3:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

LDFLAGS="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}"

echo "Building preflight ${VERSION} (${COMMIT})"

# Create dist directory
rm -rf dist
mkdir -p dist

# Build for each platform
platforms=(
    "linux/amd64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

for platform in "${platforms[@]}"; do
    os="${platform%/*}"
    arch="${platform#*/}"

    output_name="preflight"
    if [ "$os" = "windows" ]; then
        output_name="${output_name}.exe"
    fi

    echo "Building for ${os}/${arch}..."

    GOOS="$os" GOARCH="$arch" go build \
        -ldflags "${LDFLAGS}" \
        -o "dist/${output_name}" \
        ./cmd/preflight

    # Create archive
    archive_name="preflight-${os}-${arch}"
    cd dist

    if [ "$os" = "windows" ]; then
        zip "${archive_name}.zip" "${output_name}"
    else
        tar -czf "${archive_name}.tar.gz" "${output_name}"
    fi

    rm "${output_name}"
    cd ..
done

echo "Build complete. Artifacts:"
ls -la dist/
