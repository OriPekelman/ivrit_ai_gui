#!/bin/bash
# Build Linux binaries using Docker for proper cross-compilation
# Supports: linux/amd64, linux/arm64

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

VERSION="${VERSION:-dev}"
OUTPUT_DIR="dist"

echo -e "${BLUE}🚀 Building ivrit.ai Hebrew Transcription for Linux${NC}"
echo -e "${BLUE}====================================================${NC}"
echo ""

# Check if Docker is installed and running
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed${NC}"
    echo "Install Docker Desktop from: https://www.docker.com/products/docker-desktop"
    exit 1
fi

if ! docker info &> /dev/null; then
    echo -e "${RED}❌ Docker daemon is not running${NC}"
    echo "Please start Docker Desktop"
    exit 1
fi

echo -e "${GREEN}✅ Docker is ready${NC}"
echo ""

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Build function
build_linux() {
    local ARCH=$1
    local PLATFORM="linux/${ARCH}"

    echo -e "${BLUE}📦 Building for ${PLATFORM}...${NC}"

    # Build the binary using Docker
    docker buildx build \
        --platform "${PLATFORM}" \
        --file Dockerfile.linux-build \
        --target binary \
        --output "type=local,dest=${OUTPUT_DIR}/linux-${ARCH}" \
        --build-arg VERSION="${VERSION}" \
        .

    if [ $? -eq 0 ]; then
        local BINARY_PATH="${OUTPUT_DIR}/linux-${ARCH}/ivrit_ai"
        if [ -f "${BINARY_PATH}" ]; then
            # Make executable
            chmod +x "${BINARY_PATH}"

            local SIZE=$(du -h "${BINARY_PATH}" | cut -f1)
            echo -e "${GREEN}✅ Built: ${BINARY_PATH} (${SIZE})${NC}"

            # Create tarball
            local ARCHIVE_NAME="ivrit_ai-${VERSION}-linux-${ARCH}.tar.gz"
            tar -czf "${OUTPUT_DIR}/${ARCHIVE_NAME}" \
                -C "${OUTPUT_DIR}/linux-${ARCH}" ivrit_ai \
                -C ../.. README.md LICENSE models.json

            echo -e "${GREEN}📦 Created: ${OUTPUT_DIR}/${ARCHIVE_NAME}${NC}"
            echo ""
        else
            echo -e "${RED}❌ Binary not found after build${NC}"
            return 1
        fi
    else
        echo -e "${RED}❌ Build failed for ${PLATFORM}${NC}"
        return 1
    fi
}

# Check if buildx is available
if ! docker buildx version &> /dev/null; then
    echo -e "${YELLOW}⚠️  Docker buildx not available, using regular build${NC}"
    echo "For multi-arch builds, update Docker Desktop to latest version"
fi

# Clean previous builds
echo -e "${YELLOW}🧹 Cleaning previous builds...${NC}"
rm -rf "${OUTPUT_DIR}"
mkdir -p "${OUTPUT_DIR}"
echo ""

# Build for different architectures
echo -e "${BLUE}🏗️  Building for all Linux architectures...${NC}"
echo ""

# Build for AMD64 (x86_64)
build_linux "amd64"

# Build for ARM64 (aarch64)
build_linux "arm64"

# Create checksums
echo -e "${BLUE}🔒 Creating checksums...${NC}"
cd "${OUTPUT_DIR}"
if ls *.tar.gz 1> /dev/null 2>&1; then
    shasum -a 256 *.tar.gz > checksums-${VERSION}.txt
    echo -e "${GREEN}✅ Checksums created${NC}"
fi
cd ..

echo ""
echo -e "${GREEN}✅ Build complete!${NC}"
echo ""
echo -e "${BLUE}📦 Release artifacts in: ${OUTPUT_DIR}/${NC}"
ls -lh "${OUTPUT_DIR}/" 2>/dev/null || true
echo ""
echo -e "${YELLOW}Testing binaries:${NC}"
echo "  1. Extract: tar -xzf ${OUTPUT_DIR}/ivrit_ai-${VERSION}-linux-amd64.tar.gz"
echo "  2. Run: ./ivrit_ai -help"
echo ""
