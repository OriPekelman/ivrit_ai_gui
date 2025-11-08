# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of ivrit.ai Hebrew Transcription
- Native desktop application with Gio UI framework
- CLI mode for automation and scripting
- Hebrew-optimized Whisper models from ivrit.ai
- Real-time progress tracking with ETA calculation
- Automatic speaker diarization using tinydiarize
- Multi-language translation via Mistral 8B (Ollama)
- Multiple export formats: Text, JSON, SRT, VTT
- Automatic model download and caching
- Transcription result caching for instant re-processing
- Video support with automatic audio extraction
- Cross-platform support: macOS, Linux, Windows
- Comprehensive documentation and testing suite

### Changed
- Standardized binary name to `ivrit_ai` across all platforms
- Updated all documentation to reflect consistent naming

### Fixed
- Thread-safe model context management
- Safe CGO callbacks with atomic-only writes
- Memory management for large files
