#!/bin/bash
# Quick script to install all Linux dependencies for building ivrit.ai GUI

set -e

echo "Installing Linux dependencies for ivrit.ai Hebrew Transcription..."

sudo apt-get update
sudo apt-get install -y \
    build-essential \
    pkg-config \
    git \
    cmake \
    g++ \
    ffmpeg \
    libgl-dev \
    libx11-dev \
    libxcursor-dev \
    libxrandr-dev \
    libxinerama-dev \
    libxi-dev \
    libxxf86vm-dev \
    libxfixes-dev \
    libx11-xcb-dev \
    libxcb1-dev \
    libwayland-dev \
    libwayland-egl-backend-dev \
    libxkbcommon-dev \
    libxkbcommon-x11-0 \
    libxkbcommon-x11-dev \
    libgtk-3-dev \
    libvulkan-dev \
    libegl1-mesa-dev

echo ""
echo "✅ All dependencies installed!"
echo ""
echo "Next steps:"
echo "1. Install whisper.cpp: ./install-whisper-linux.sh"
echo "2. Build the application: ./build.sh"
echo ""
