#!/bin/bash
# Generate Windows resources (icon and manifest)
# This script uses go-winres to embed icons into Windows executables

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

echo "Generating Windows resources..."

# Install go-winres if not present
if ! command -v go-winres &> /dev/null; then
    echo "Installing go-winres..."
    go install github.com/tc-hib/go-winres@latest
fi

# Generate .syso file from winres.json
cd cmd/ivrit_ai_gui
go-winres make

echo "✅ Windows resources generated: cmd/ivrit_ai_gui/rsrc_windows_amd64.syso"
echo "   This file will be automatically included in Windows builds"
