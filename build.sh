#!/bin/bash

# Create build directory
mkdir -p build

# Platforms to build for
PLATFORMS=("windows/amd64" "windows/386" "linux/amd64" "linux/386" "darwin/amd64" "darwin/arm64")

# Build version from git tag or default
VERSION=${1:-"1.0.0"}

# Build date
BUILD_DATE=$(date '+%Y-%m-%d_%H:%M:%S')

for PLATFORM in "${PLATFORMS[@]}"
do
    # Split platform into OS and ARCH
    IFS="/" read -r -a platform_split <<< "$PLATFORM"
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    
    # Set output name based on OS
    if [ "$GOOS" = "windows" ]; then
        output_name="hl7_processor_${GOOS}_${GOARCH}.exe"
    else
        output_name="hl7_processor_${GOOS}_${GOARCH}"
    fi

    # Build binary
    echo "Building for $GOOS/$GOARCH..."
    env GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-X main.Version=$VERSION -X main.BuildDate=$BUILD_DATE" -o "build/$output_name" .
    
    # Create zip archive
    if [ "$GOOS" = "windows" ]; then
        zip -j "build/${output_name%.exe}.zip" "build/$output_name"
    else
        tar -czf "build/${output_name}.tar.gz" -C build "$output_name"
    fi
    
    # Cleanup binary
    rm "build/$output_name"
done

echo "Build complete! Check the build directory for compiled binaries."