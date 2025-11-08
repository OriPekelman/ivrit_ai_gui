# Building from Source

Complete guide for building ivrit.ai Hebrew Transcription from source.

## Quick Start

**macOS or Linux:**
```bash
./scripts/build.sh
```

**For production releases:**
Use GitHub Actions (see below)

---

## Local Development Build

### Prerequisites

**All Platforms:**
- Go 1.21 or later
- Git

**macOS:**
```bash
brew install go whisper-cpp ffmpeg
```

**Linux (Ubuntu/Debian):**
```bash
# Install Go
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Install dependencies
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

# Build and install whisper.cpp
# Note: whisper.cpp doesn't have 'make install', we need to manually install
git clone https://github.com/ggerganov/whisper.cpp.git
cd whisper.cpp

# Build with cmake
cmake -B build -DCMAKE_BUILD_TYPE=Release -DBUILD_SHARED_LIBS=ON
cmake --build build -j$(nproc)

# Install libraries and headers
sudo mkdir -p /usr/local/lib /usr/local/include
sudo cp build/src/libwhisper.so* /usr/local/lib/
sudo cp build/ggml/src/libggml*.so* /usr/local/lib/
sudo cp include/whisper.h /usr/local/include/
sudo cp ggml/include/*.h /usr/local/include/
sudo ldconfig

# Or use the provided script:
# ./install-whisper-linux.sh
```

### Build the Application

```bash
# Clone repository
git clone <your-repo-url>
cd ivrit_ai_gui

# Build
./scripts/build.sh
```

**Output:** `./ivrit_ai`

### Test Your Build

```bash
# GUI mode
./ivrit_ai

# CLI mode
./ivrit_ai -help
./ivrit_ai -input audio.m4a
```

---

## Production Builds with GitHub Actions

The easiest way to create production binaries for all platforms.

### Trigger a Release Build

```bash
# Tag a version
git tag v1.0.0
git push origin v1.0.0

# GitHub Actions automatically builds:
# - macOS (Intel + Apple Silicon)
# - Linux (AMD64 + ARM64)
# - Windows (AMD64)
```

Binaries appear in GitHub Releases within 10-15 minutes.

### What Gets Built

| Platform | Architecture | Output |
|----------|-------------|--------|
| macOS | Intel (AMD64) | `.dmg` + binary |
| macOS | Apple Silicon (ARM64) | `.dmg` + binary |
| Linux | AMD64 | `.tar.gz` |
| Linux | ARM64 | `.tar.gz` |
| Windows | AMD64 | `.zip` |

---

## Advanced: Docker-Based Linux Builds

For cross-compiling Linux binaries from macOS with full CGO support.

### Prerequisites
- Docker Desktop installed and running
- At least 5GB free disk space

### Build with Docker

```bash
./scripts/build-linux.sh
```

This creates:
- `dist/ivrit_ai_gui-dev-linux-amd64.tar.gz`
- `dist/ivrit_ai_gui-dev-linux-arm64.tar.gz`

**Note:** First build takes 10-15 minutes (compiles whisper.cpp from source).
Subsequent builds are faster using Docker layer cache.

### How It Works

1. Uses Debian Bookworm base image
2. Installs build tools (gcc, cmake, etc.)
3. Compiles whisper.cpp from source
4. Builds Go application with CGO enabled
5. Exports binary with all dependencies

---

## Troubleshooting

### macOS: "whisper.cpp not found"

```bash
brew install whisper-cpp
```

### Linux: "whisper.cpp not found"

Check installation:
```bash
ls -la /usr/local/lib/libwhisper.*
ls -la /usr/local/include/whisper.h
```

If missing, reinstall whisper.cpp:
```bash
cd ~/whisper.cpp
make clean
make -j$(nproc)
sudo make install
sudo ldconfig
```

### Linux: "error while loading shared libraries: libwhisper.so"

Update library cache:
```bash
sudo ldconfig
```

Or add to library path:
```bash
export LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH
echo 'export LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH' >> ~/.bashrc
```

### Linux: "fatal error: vulkan/vulkan.h: No such file or directory"

Install Vulkan development headers:
```bash
sudo apt-get install -y libvulkan-dev
```

### Linux: "No CMAKE_CXX_COMPILER could be found"

Install C++ compiler:
```bash
sudo apt-get install -y g++
```

### Docker build fails: "no space left on device"

Clean Docker:
```bash
docker system prune -a
```

Or increase disk space in Docker Desktop → Settings → Resources.

### macOS: CGO errors

Install Command Line Tools:
```bash
xcode-select --install
```

---

## Build Artifacts

After a successful build, you'll have:

**Local build:**
- `ivrit_ai` - Main executable

**GitHub Actions:**
- Compressed archives for each platform
- DMG installers for macOS
- SHA256 checksums
- Automatic release notes

---

## Development Workflow

### Quick iteration
```bash
# Make code changes
vim cli.go

# Rebuild
./scripts/build.sh

# Test
./ivrit_ai -input test.m4a
```

### Testing CLI changes
```bash
./ivrit_ai -help
./ivrit_ai -input audio.m4a -format srt
```

### Testing GUI changes
```bash
./ivrit_ai
```

### Before committing
```bash
# Format code
go fmt ./...

# Check for issues
go vet ./...

# Build to verify
./scripts/build.sh
```

---

## Cross-Compilation Notes

### Why Docker for Linux?

Cross-compiling with CGO is challenging because:
- Need matching C libraries for target platform
- whisper.cpp requires native compilation
- GUI frameworks need platform-specific libraries

Docker solves this by building in a native Linux environment.

### Why GitHub Actions is Recommended?

- Professional build environment
- Builds all platforms in parallel
- Free for open source projects
- Automatic release creation
- Consistent, reproducible builds

---

## Platform-Specific Notes

### macOS

**Intel Macs:**
- Use `/usr/local/bin/brew` paths
- x86_64 architecture

**Apple Silicon Macs:**
- Use `/opt/homebrew` paths
- ARM64 architecture

The build script auto-detects your architecture.

### Linux

**Supported distributions:**
- Ubuntu 20.04+
- Debian 11+
- Fedora 35+
- Other glibc-based distros

**GUI requirements:**
- X11 or Wayland
- libGL and X11 development libraries

**CLI-only servers:**
- Don't need X11 libraries
- Smaller dependencies footprint

### Windows

Currently only supported via GitHub Actions.

For local Windows development:
- Use WSL2 with Linux instructions
- Or wait for native Windows build documentation

---

## Getting Help

**Build issues:** Open an issue on GitHub with:
- Output of `./build.sh`
- Operating system and version
- Go version (`go version`)
- Architecture (`uname -m`)

**Feature requests:** See [CONTRIBUTING.md](CONTRIBUTING.md)
