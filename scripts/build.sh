#!/bin/bash
# Build native Go application with whisper.cpp support

set -e

# FYNE_FONT should be a path to a .ttf font file (not a font name)
# The application will auto-detect Hebrew fonts if FYNE_FONT is not set
# For permanent solution, use font_bundle.sh to bundle fonts into the app

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

echo "Building native Go application with whisper.cpp support..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed."
    echo "Install with: brew install go (macOS) or download from https://go.dev/dl/"
    exit 1
fi

# Detect whisper.cpp installation
WHISPER_FOUND=0
OS_TYPE=$(uname)

echo "Detecting whisper.cpp for $OS_TYPE..."

# Check macOS Homebrew paths
if [ "$OS_TYPE" = "Darwin" ]; then
    WHISPER_CPP_PATH="/opt/homebrew/Cellar/whisper-cpp"
    if [ ! -d "$WHISPER_CPP_PATH" ]; then
        WHISPER_CPP_PATH="/usr/local/Cellar/whisper-cpp"
    fi

    if [ -d "$WHISPER_CPP_PATH" ]; then
        # Find latest version
        WHISPER_VERSION=$(ls -1 "$WHISPER_CPP_PATH" | sort -V | tail -1)
        WHISPER_PREFIX="$WHISPER_CPP_PATH/$WHISPER_VERSION/libexec"

        if [ -d "$WHISPER_PREFIX" ]; then
            echo "Found whisper.cpp at: $WHISPER_PREFIX"
            export CGO_CFLAGS="-I$WHISPER_PREFIX/include"
            export CGO_LDFLAGS="-L$WHISPER_PREFIX/lib -lwhisper -lggml -lggml-base -lm -lpthread"
            WHISPER_FOUND=1
        fi
    fi
# Check Linux system paths
elif [ "$OS_TYPE" = "Linux" ]; then
    # Check if whisper libraries are installed in standard locations
    if [ -f "/usr/local/lib/libwhisper.so" ] || [ -f "/usr/lib/libwhisper.so" ] || [ -f "/usr/local/lib/libwhisper.a" ]; then
        echo "Found whisper.cpp in system libraries"

        # Try pkg-config first
        if command -v pkg-config &> /dev/null && pkg-config --exists whisper 2>/dev/null; then
            echo "Using pkg-config for whisper.cpp"
            export CGO_CFLAGS="$(pkg-config --cflags whisper)"
            export CGO_LDFLAGS="$(pkg-config --libs whisper)"
        else
            # Fallback to standard paths
            echo "Using standard library paths"
            export CGO_CFLAGS="-I/usr/local/include -I/usr/include"
            export CGO_LDFLAGS="-L/usr/local/lib -L/usr/lib -lwhisper -lggml -lggml-base -lm -lpthread -ldl"
        fi
        WHISPER_FOUND=1
    fi
fi

if [ $WHISPER_FOUND -eq 0 ]; then
    echo ""
    echo "⚠️  WARNING: whisper.cpp not found"
    if [ "$OS_TYPE" = "Darwin" ]; then
        echo "Install with: brew install whisper-cpp"
    elif [ "$OS_TYPE" = "Linux" ]; then
        echo "Install whisper.cpp:"
        echo "  git clone https://github.com/ggerganov/whisper.cpp.git"
        echo "  cd whisper.cpp"
        echo "  make -j\$(nproc)"
        echo "  sudo make install"
        echo "  sudo ldconfig"
    fi
    echo ""
    echo "Building without native whisper.cpp support (app may not work)"
    echo ""
fi

# Initialize Go module if needed
if [ ! -f "go.mod" ]; then
    go mod init github.com/ivrit-ai/hebrew-transcription-native
fi

# Download dependencies
echo "Downloading dependencies..."
go mod tidy

# Build
echo "Building executable..."
# Build Gio-based GUI (pure Go with RTL support)
go build -o ivrit_ai ./cmd/ivrit_ai_gui

echo ""
echo "✅ Build complete!"
echo "Executable: $PROJECT_ROOT/ivrit_ai"
echo ""

if [ $WHISPER_FOUND -eq 1 ]; then
    echo "✅ Native whisper.cpp support enabled"
    echo ""
    echo "Models will be automatically downloaded on first use to:"
    echo "  ~/.cache/whisper/"
    echo ""
    echo "Run the application:"
    echo "  ./ivrit_ai           # GUI mode"
    echo "  ./ivrit_ai -help     # CLI mode"
else
    echo "⚠️  Built without whisper.cpp support"
    echo "The application requires whisper.cpp to function."
    echo ""
fi
