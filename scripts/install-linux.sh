#!/bin/bash
# Install Ivrit.AI Transcription on Linux
# This script installs the application, icon, and desktop entry

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Determine installation prefix
PREFIX="${PREFIX:-$HOME/.local}"

echo "Installing Ivrit.AI Transcription to $PREFIX"

# Create directories
mkdir -p "$PREFIX/bin"
mkdir -p "$PREFIX/share/applications"
mkdir -p "$PREFIX/share/icons/hicolor/256x256/apps"

# Install binary
if [ -f "$PROJECT_ROOT/ivrit_ai" ]; then
    echo "Installing binary..."
    cp "$PROJECT_ROOT/ivrit_ai" "$PREFIX/bin/ivrit_ai"
    chmod +x "$PREFIX/bin/ivrit_ai"
elif [ -f "$PROJECT_ROOT/ivrit_ai-linux-amd64" ]; then
    echo "Installing binary..."
    cp "$PROJECT_ROOT/ivrit_ai-linux-amd64" "$PREFIX/bin/ivrit_ai"
    chmod +x "$PREFIX/bin/ivrit_ai"
else
    echo "Error: Binary not found. Build the application first with ./scripts/build.sh"
    exit 1
fi

# Install icon
if [ -f "$PROJECT_ROOT/icons/icon.png" ]; then
    echo "Installing icon..."
    cp "$PROJECT_ROOT/icons/icon.png" "$PREFIX/share/icons/hicolor/256x256/apps/ivrit-ai-transcription.png"
elif [ -f "$PROJECT_ROOT/logo.png" ]; then
    echo "Installing icon..."
    cp "$PROJECT_ROOT/logo.png" "$PREFIX/share/icons/hicolor/256x256/apps/ivrit-ai-transcription.png"
fi

# Install desktop entry
if [ -f "$PROJECT_ROOT/ivrit-ai-transcription.desktop" ]; then
    echo "Installing desktop entry..."
    # Update Exec path in desktop file
    sed "s|^Exec=.*|Exec=$PREFIX/bin/ivrit_ai|" "$PROJECT_ROOT/ivrit-ai-transcription.desktop" > "$PREFIX/share/applications/ivrit-ai-transcription.desktop"
    chmod +x "$PREFIX/share/applications/ivrit-ai-transcription.desktop"
fi

# Update icon cache if using system directories
if [ "$PREFIX" = "/usr" ] || [ "$PREFIX" = "/usr/local" ]; then
    if command -v gtk-update-icon-cache &> /dev/null; then
        echo "Updating icon cache..."
        gtk-update-icon-cache -f -t "$PREFIX/share/icons/hicolor" || true
    fi
    if command -v update-desktop-database &> /dev/null; then
        echo "Updating desktop database..."
        update-desktop-database "$PREFIX/share/applications" || true
    fi
fi

echo ""
echo "✅ Installation complete!"
echo ""
echo "Run the application:"
echo "  ivrit_ai           # GUI mode"
echo "  ivrit_ai -help     # CLI mode"
echo ""
echo "Or find it in your application menu as 'Ivrit.AI Transcription'"
echo ""

# Add to PATH instructions if needed
if [[ ":$PATH:" != *":$PREFIX/bin:"* ]]; then
    echo "⚠️  Note: $PREFIX/bin is not in your PATH"
    echo "Add this line to your ~/.bashrc or ~/.zshrc:"
    echo "  export PATH=\"$PREFIX/bin:\$PATH\""
    echo ""
fi
