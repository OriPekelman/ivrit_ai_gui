#!/bin/bash
# Script to build and install whisper.cpp on Linux
# This script properly installs whisper.cpp since there's no 'make install' target

set -e

echo "Building and installing whisper.cpp..."

# Clone whisper.cpp if not already present
if [ ! -d "/tmp/whisper.cpp" ]; then
    echo "Cloning whisper.cpp..."
    git clone https://github.com/ggerganov/whisper.cpp.git /tmp/whisper.cpp
fi

cd /tmp/whisper.cpp

# Build with cmake
echo "Building whisper.cpp with cmake..."
cmake -B build -DCMAKE_BUILD_TYPE=Release -DBUILD_SHARED_LIBS=ON
cmake --build build -j$(nproc)

# Install libraries
echo "Installing libraries to /usr/local/lib..."
sudo mkdir -p /usr/local/lib
sudo cp -v build/src/libwhisper.so* /usr/local/lib/ 2>/dev/null || true
sudo cp -v build/ggml/src/libggml*.so* /usr/local/lib/ 2>/dev/null || true

# Install headers
echo "Installing headers to /usr/local/include..."
sudo mkdir -p /usr/local/include
sudo cp -v include/whisper.h /usr/local/include/
sudo cp -v ggml/include/*.h /usr/local/include/

# Update library cache
echo "Updating library cache..."
sudo ldconfig

echo ""
echo "✅ whisper.cpp installed successfully!"
echo ""
echo "Verify installation:"
echo "  ls -la /usr/local/lib/libwhisper.so*"
echo "  ls -la /usr/local/include/whisper.h"
echo ""
