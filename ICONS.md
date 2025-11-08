# Application Icons

This document explains how icons are configured for the Hebrew Transcription application across different platforms.

## Icon Files

All icon files are located in the `icons/` directory:

- **`icons/icon.png`** - Source PNG (256x256) for Linux
- **`icons/icon.ico`** - Windows icon (multiple sizes: 16, 32, 48, 64, 96, 128, 256)
- **`icons/icon.icns`** - macOS icon set
- **`icons/icon.iconset/`** - Temporary iconset directory (not in git)

## Platform-Specific Implementation

### Windows

**How it works:**
- Windows executables embed icons using a compiled resource file (`.syso`)
- The resource is defined in `cmd/ivrit_ai_gui/winres.json`
- During build, `go-winres` generates `rsrc_windows_amd64.syso`
- Go build automatically includes `.syso` files in the executable

**Configuration:**
- Icon: `cmd/ivrit_ai_gui/winres.json` → references `icons/icon.ico`
- Manifest: Includes app metadata and DPI awareness settings
- Version info: Company name, product version, copyright

**Build process:**
```bash
# Automatic in GitHub Actions
# Manual build:
./scripts/generate-windows-resources.sh
go build -o app.exe ./cmd/ivrit_ai_gui
```

### macOS

**How it works:**
- macOS apps use `.icns` files in the app bundle
- Icon specified in `Info.plist` via `CFBundleIconFile`
- The `.app` bundle structure includes the icon in `Resources/`

**Configuration:**
- Icon: `IvritAI.app/Contents/Resources/icon.icns`
- Info.plist: `.github/workflows/release.yml` (macOS build section)

**Build process:**
```bash
# Creates app bundle with icon
# Defined in .github/workflows/release.yml
# Produces: ivrit_ai-macos.dmg
```

### Linux

**How it works:**
- Desktop files (`.desktop`) reference icon names
- Icons installed to `~/.local/share/icons/hicolor/256x256/apps/`
- System looks up icons by name from standard directories

**Configuration:**
- Desktop file: `ivrit-ai-transcription.desktop`
- Icon name: `ivrit-ai-transcription` (references PNG file)
- Icon file: `icons/icon.png`

**Installation:**
```bash
# Automatic install script
./scripts/install-linux.sh

# Manual install
cp icons/icon.png ~/.local/share/icons/hicolor/256x256/apps/ivrit-ai-transcription.png
cp ivrit-ai-transcription.desktop ~/.local/share/applications/
```

## Regenerating Icons

If you update `logo.png`, regenerate all icon formats:

### Prerequisites

**macOS:**
```bash
brew install imagemagick
# iconutil is built-in
```

**Linux:**
```bash
sudo apt-get install imagemagick
```

### Generate All Icons

```bash
# Convert to Windows .ico
magick logo.png -resize 256x256 -define icon:auto-resize=256,128,96,64,48,32,16 icons/icon.ico

# Create macOS iconset
mkdir -p icons/icon.iconset
for size in 16 32 128 256 512; do
  magick logo.png -resize ${size}x${size} icons/icon.iconset/icon_${size}x${size}.png
done
for size in 32 64 256 512; do
  magick logo.png -resize ${size}x${size} icons/icon.iconset/icon_$((size/2))x$((size/2))@2x.png
done
iconutil -c icns icons/icon.iconset -o icons/icon.icns

# Copy for Linux
cp logo.png icons/icon.png

# Regenerate Windows resources
./scripts/generate-windows-resources.sh
```

## Testing Icons

### Windows
- Build the executable and check icon in Windows Explorer
- Right-click → Properties should show icon and version info

### macOS
- Open the DMG and check the app icon
- Drag to Applications and verify it appears with the correct icon

### Linux
- Install using `./scripts/install-linux.sh`
- Check application menu for "Ivrit.AI Transcription" with icon
- Run `gtk-launch ivrit-ai-transcription` to test

## Icon Design Guidelines

**Current icon:** Hebrew letter Yod (י) with "עברים, דברו עברית" (Hebrews, speak Hebrew)

**Recommendations:**
- Keep design recognizable at 16x16 pixels
- Use high contrast for small sizes
- Test on light and dark backgrounds
- Ensure Hebrew text is readable at larger sizes

## Files Reference

```
.
├── icons/
│   ├── icon.png          # Linux (256x256)
│   ├── icon.ico          # Windows (multi-size)
│   ├── icon.icns         # macOS (multi-size)
│   └── icon.iconset/     # macOS build temp (not in git)
├── cmd/ivrit_ai_gui/
│   ├── winres.json       # Windows resource configuration
│   └── rsrc_*.syso       # Generated Windows resource (not in git)
├── ivrit-ai-transcription.desktop  # Linux desktop entry
├── scripts/
│   ├── generate-windows-resources.sh
│   └── install-linux.sh
└── .github/workflows/
    └── release.yml       # Includes icon in all builds
```

## Troubleshooting

### Windows icon not showing
- Ensure `go-winres` is installed: `go install github.com/tc-hib/go-winres@latest`
- Regenerate resources: `./scripts/generate-windows-resources.sh`
- Check `.syso` file exists: `ls -la cmd/ivrit_ai_gui/rsrc_*.syso`

### macOS icon not showing
- Verify icon is in bundle: `ls -la IvritAI.app/Contents/Resources/`
- Check Info.plist has `CFBundleIconFile` key
- Clear icon cache: `sudo rm -rf /Library/Caches/com.apple.iconservices.store`

### Linux icon not showing
- Install to proper location: `~/.local/share/icons/hicolor/256x256/apps/`
- Update icon cache: `gtk-update-icon-cache -f -t ~/.local/share/icons/hicolor`
- Update desktop database: `update-desktop-database ~/.local/share/applications`

## Resources

- [Windows Resource Format](https://github.com/tc-hib/go-winres)
- [macOS App Bundle Guidelines](https://developer.apple.com/library/archive/documentation/CoreFoundation/Conceptual/CFBundles/BundleTypes/BundleTypes.html)
- [freedesktop.org Icon Theme Specification](https://specifications.freedesktop.org/icon-theme-spec/icon-theme-spec-latest.html)
