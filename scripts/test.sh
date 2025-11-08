#!/bin/bash
# Run tests with proper CGO flags for whisper.cpp

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

# Detect whisper.cpp installation (same logic as build.sh)
WHISPER_FOUND=0
OS_TYPE=$(uname)

if [ "$OS_TYPE" = "Darwin" ]; then
    WHISPER_CPP_PATH="/opt/homebrew/Cellar/whisper-cpp"
    if [ ! -d "$WHISPER_CPP_PATH" ]; then
        WHISPER_CPP_PATH="/usr/local/Cellar/whisper-cpp"
    fi

    if [ -d "$WHISPER_CPP_PATH" ]; then
        WHISPER_VERSION=$(ls -1 "$WHISPER_CPP_PATH" | sort -V | tail -1)
        WHISPER_PREFIX="$WHISPER_CPP_PATH/$WHISPER_VERSION/libexec"

        if [ -d "$WHISPER_PREFIX" ]; then
            export CGO_CFLAGS="-I$WHISPER_PREFIX/include"
            export CGO_LDFLAGS="-L$WHISPER_PREFIX/lib -lwhisper -lggml -lggml-base -lm -lpthread"
            WHISPER_FOUND=1
        fi
    fi
elif [ "$OS_TYPE" = "Linux" ]; then
    if [ -f "/usr/local/lib/libwhisper.so" ] || [ -f "/usr/lib/libwhisper.so" ] || [ -f "/usr/local/lib/libwhisper.a" ]; then
        if command -v pkg-config &> /dev/null && pkg-config --exists whisper 2>/dev/null; then
            export CGO_CFLAGS="$(pkg-config --cflags whisper)"
            export CGO_LDFLAGS="$(pkg-config --libs whisper)"
        else
            export CGO_CFLAGS="-I/usr/local/include -I/usr/include"
            export CGO_LDFLAGS="-L/usr/local/lib -L/usr/lib -lwhisper -lggml -lggml-base -lm -lpthread -ldl"
        fi
        WHISPER_FOUND=1
    fi
fi

if [ $WHISPER_FOUND -eq 0 ]; then
    echo "⚠️  Warning: whisper.cpp not found"
    echo "Tests requiring CGO may fail"
    echo ""
fi

# Run tests with proper flags
echo "Running tests..."
go test -v ./cmd/ivrit_ai_gui "$@"
