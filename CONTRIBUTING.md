# Contributing to ivrit.ai Hebrew Transcription

Thank you for your interest in contributing! This document provides guidelines for contributing to the project.

## Code of Conduct

Be respectful, inclusive, and constructive. We're here to build great tools for the Hebrew language community.

## How to Contribute

### Reporting Bugs

Before creating a bug report:
1. Check if the issue already exists
2. Ensure you're using the latest version
3. Collect relevant information:
   - OS and version
   - Audio file length and format
   - Model used
   - Complete error message/stack trace

Create an issue with:
- Clear title describing the problem
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs or screenshots

### Suggesting Features

Feature requests are welcome! Please:
1. Check if the feature is already requested
2. Explain the use case clearly
3. Describe the expected behavior
4. Consider implementation complexity

### Pull Requests

#### Before Starting

1. Check existing issues and PRs
2. For large changes, open an issue first to discuss
3. Fork the repository
4. Create a feature branch from `main`

#### Development Setup

```bash
# Clone your fork
git clone https://github.com/OriPekelman/ivrit_ai_gui.git
cd ivrit_ai_gui

# Install dependencies
brew install whisper-cpp ffmpeg  # macOS
go mod download

# Build
./scripts/build.sh

# Test
go test ./...
```

#### Code Guidelines

**Go Style**
- Follow standard Go formatting (`go fmt`)
- Use meaningful variable names
- Add comments for complex logic
- Keep functions focused and small

**CGO Safety**
- **CRITICAL**: Never allocate memory in C callbacks
- Use atomic variables for C→Go communication
- No nested C→Go→C calls
- Test with large files (30+ minutes)

**UI Updates**
- Use mutex protection for shared state
- Force window invalidation after state changes
- Test with Hebrew text (RTL)

**Commits**
- Use clear, descriptive commit messages
- Reference issues (e.g., "Fixes #123")
- Keep commits focused (one logical change per commit)

#### Testing

```bash
# Run tests
go test ./...

# Test with different file sizes
# - Small: < 1 minute
# - Medium: 5-10 minutes
# - Large: 20-30 minutes

# Test all models
# - turbo
# - large-v3
# - base

# Test features
# - Transcription caching (re-transcribe same file)
# - Speaker diarization
# - Translation (requires Ollama)
# - All export formats (text, JSON, SRT, VTT)
```

#### Pull Request Process

1. Update documentation if needed
2. Add tests for new features
3. Ensure all tests pass
4. Create PR with:
   - Clear description of changes
   - Link to related issues
   - Screenshots for UI changes

### Documentation

Documentation improvements are always welcome:
- Fix typos or unclear instructions
- Add examples
- Improve technical explanations
- Translate to other languages

## Project Structure

```
.
├── cmd/ivrit_ai_gui/
│   ├── main.go               # Application entry point
│   ├── gui.go                # Gio UI implementation
│   ├── cli.go                # CLI mode implementation
│   ├── whisper_cgo.go        # CGO bindings to whisper.cpp
│   ├── transcription.go      # Core transcription logic
│   ├── model_download.go     # Model management
│   ├── mistral_translation.go # Translation via Mistral
│   └── audio_utils.go        # Audio file utilities
├── scripts/
│   ├── build.sh              # Development build script
│   ├── build-linux.sh        # Docker-based Linux builds
│   └── test.sh               # Test runner with CGO support
├── models.json               # Model configuration
├── README.md
├── BUILDING.md
├── TESTING.md
└── CONTRIBUTING.md
```

## Architecture

### Threading Model

```
Main Thread (UI)
  ↓
Worker Goroutine (Transcription)
  ↓
CGO → whisper_full() [C code, blocks]
  ↓
Progress Callback (C) → Atomic Write Only
  ↓
Polling Goroutine → Read Atomic → Update UI
```

### Key Principles

1. **Thread Safety**: Model contexts protected by mutex
2. **CGO Safety**: No allocations in C callbacks, atomic-only
3. **Caching**: Models cached globally, results cached per file+model
4. **Progress**: Atomic variables + polling, not direct callbacks

## Release Process

Maintainers only:

1. Create a version tag (e.g., `v1.0.0`)
2. Push the tag to GitHub: `git push origin v1.0.0`
3. GitHub Actions automatically builds all platform binaries
4. Review and publish the auto-generated release

## Questions?

- **Issues**: https://github.com/OriPekelman/ivrit_ai_gui/issues
- **Discussions**: https://github.com/OriPekelman/ivrit_ai_gui/discussions
- **ivrit.ai**: https://ivrit.ai

Thank you for contributing! 🙏
