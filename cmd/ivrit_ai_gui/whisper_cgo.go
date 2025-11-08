package main

/*
// Cross-platform CGO configuration for whisper.cpp
// Supports multiple installation methods and platforms

// Note: whisper.cpp does not provide pkg-config files
// Use platform-specific paths below instead

// macOS with Homebrew (both Intel and Apple Silicon)
#cgo darwin CFLAGS: -I/opt/homebrew/include -I/usr/local/include
#cgo darwin LDFLAGS: -L/opt/homebrew/lib -L/usr/local/lib -lwhisper -lggml -lggml-base -lm -lpthread

// Linux (standard paths)
#cgo linux CFLAGS: -I/usr/include -I/usr/local/include
#cgo linux LDFLAGS: -L/usr/lib -L/usr/local/lib -lwhisper -lggml -lggml-base -lm -lpthread -ldl

// Windows (assuming whisper-cpp installed in standard location)
#cgo windows CFLAGS: -IC:/msys64/mingw64/include
#cgo windows LDFLAGS: -LC:/msys64/mingw64/lib -lwhisper -lggml -lggml-base

#include <stdlib.h>
#include <string.h>
#include "whisper.h"

// Forward declare callback wrappers
extern void whisper_new_segment_callback_go(struct whisper_context * ctx, struct whisper_state * state, int n_new, void * user_data);
extern void whisper_progress_callback_go(struct whisper_context * ctx, struct whisper_state * state, int progress, void * user_data);
*/
import "C"

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// Callback registry to pass Go functions to C callbacks
var (
	callbackRegistry      = make(map[uintptr]*transcriptionCallbacks)
	callbackRegistryMutex sync.RWMutex
	callbackIDCounter     uintptr = 1
)

type transcriptionCallbacks struct {
	progressCallback func(string)
	segmentCallback  func(Segment)
	ctx              *C.struct_whisper_context
	currentSpeaker   int    // Track current speaker for real-time callbacks
	progressPercent  *int32 // Atomic progress percentage (0-100)
}

//export whisper_new_segment_callback_go
func whisper_new_segment_callback_go(ctx *C.struct_whisper_context, state *C.struct_whisper_state, nNew C.int, userData unsafe.Pointer) {
	callbackID := uintptr(userData)

	// Get all C data FIRST, before locking
	nSegments := int(C.whisper_full_n_segments(ctx))
	if nSegments == 0 {
		return
	}

	lastIdx := nSegments - 1

	// Check for speaker turn (if this isn't the first segment)
	speakerTurn := false
	if lastIdx > 0 {
		speakerTurn = bool(C.whisper_full_get_segment_speaker_turn_next(ctx, C.int(lastIdx-1)))
	}

	t0 := C.whisper_full_get_segment_t0(ctx, C.int(lastIdx))
	t1 := C.whisper_full_get_segment_t1(ctx, C.int(lastIdx))
	textPtr := C.whisper_full_get_segment_text(ctx, C.int(lastIdx))
	text := C.GoString(textPtr)

	// NOW lock and update Go state
	callbackRegistryMutex.Lock()
	callbacks, ok := callbackRegistry[callbackID]
	if !ok || callbacks == nil || callbacks.segmentCallback == nil {
		callbackRegistryMutex.Unlock()
		return
	}

	if speakerTurn {
		callbacks.currentSpeaker++
	}

	segment := Segment{
		Start:   float64(t0) / 100.0,
		End:     float64(t1) / 100.0,
		Text:    text,
		Speaker: callbacks.currentSpeaker,
	}

	cb := callbacks.segmentCallback
	callbackRegistryMutex.Unlock()

	// Call callback outside of lock
	cb(segment)
}

//export whisper_progress_callback_go
func whisper_progress_callback_go(ctx *C.struct_whisper_context, state *C.struct_whisper_state, progress C.int, userData unsafe.Pointer) {
	callbackID := uintptr(userData)

	// CRITICAL: Only use atomic operations here - no allocations, no function calls
	// Any allocation can trigger GC which can move memory C code is using
	callbackRegistryMutex.RLock()
	callbacks, ok := callbackRegistry[callbackID]
	callbackRegistryMutex.RUnlock()

	if ok && callbacks != nil && callbacks.progressPercent != nil {
		// Atomic write only - safe from C callback
		atomic.StoreInt32(callbacks.progressPercent, int32(progress))
	}
}

// cachedModel wraps a whisper context with a mutex to ensure thread-safe access
type cachedModel struct {
	ctx   *C.struct_whisper_context
	mutex sync.Mutex // Protects concurrent calls to whisper_full
}

// Model cache to avoid reloading the same model
var (
	modelCache      = make(map[string]*cachedModel)
	modelCacheMutex sync.RWMutex
)

// Transcription result cache to avoid re-transcribing the same files
type transcriptionCacheKey struct {
	audioPath string
	modelID   string
}

var (
	transcriptionCache      = make(map[transcriptionCacheKey][]Segment)
	transcriptionCacheMutex sync.RWMutex
)

// WhisperCGOEngine implements TranscriptionEngine using direct cgo bindings
type WhisperCGOEngine struct {
	model     *cachedModel // Reference to cached model (includes mutex)
	modelPath string
	fromCache bool // Whether this engine is using a cached model
}

// NewWhisperCGOEngine creates a new whisper engine using direct cgo with model caching
func NewWhisperCGOEngine(modelPath string) (*WhisperCGOEngine, error) {
	if modelPath == "" {
		return nil, fmt.Errorf("model path required")
	}

	// Check if model exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("model file not found: %s", modelPath)
	}

	// Check cache first
	modelCacheMutex.RLock()
	cachedMdl, exists := modelCache[modelPath]
	modelCacheMutex.RUnlock()

	if exists && cachedMdl != nil {
		// Model already loaded, reuse it
		return &WhisperCGOEngine{
			model:     cachedMdl,
			modelPath: modelPath,
			fromCache: true,
		}, nil
	}

	// Convert Go string to C string
	cModelPath := C.CString(modelPath)
	defer C.free(unsafe.Pointer(cModelPath))

	// Load model with default params
	params := C.whisper_context_default_params()
	// Note: GPU support is automatically used if available in whisper.cpp build

	ctx := C.whisper_init_from_file_with_params(cModelPath, params)
	if ctx == nil {
		return nil, fmt.Errorf("failed to load model from %s", modelPath)
	}

	// Create cached model with mutex
	cachedMdl = &cachedModel{
		ctx: ctx,
	}

	// Store in cache
	modelCacheMutex.Lock()
	modelCache[modelPath] = cachedMdl
	modelCacheMutex.Unlock()

	return &WhisperCGOEngine{
		model:     cachedMdl,
		modelPath: modelPath,
		fromCache: false,
	}, nil
}

// SupportsModel checks if this engine supports the given model
func (e *WhisperCGOEngine) SupportsModel(modelID string) bool {
	return modelID == "large-v3" || modelID == "turbo" || modelID == "base"
}

// Transcribe transcribes audio using native whisper.cpp
func (e *WhisperCGOEngine) Transcribe(audioPath string, modelID string, cpuThreads int, progressCallback func(string), segmentCallback func(Segment)) ([]Segment, error) {
	return e.TranscribeWithTranslation(audioPath, modelID, cpuThreads, "", progressCallback, segmentCallback)
}

// TranscribeWithTranslation transcribes and optionally translates using whisper.cpp
// translateTo: target language code (e.g., "en"). whisper.cpp only supports translation to English.
func (e *WhisperCGOEngine) TranscribeWithTranslation(audioPath string, modelID string, cpuThreads int, translateTo string, progressCallback func(string), segmentCallback func(Segment)) ([]Segment, error) {
	if e.model == nil || e.model.ctx == nil {
		return nil, fmt.Errorf("whisper context not initialized")
	}

	// Check transcription cache first (only for non-translated transcriptions)
	if translateTo == "" {
		cacheKey := transcriptionCacheKey{audioPath: audioPath, modelID: modelID}
		transcriptionCacheMutex.RLock()
		cachedSegments, exists := transcriptionCache[cacheKey]
		transcriptionCacheMutex.RUnlock()

		if exists {
			if progressCallback != nil {
				progressCallback("Using cached transcription...")
			}
			// Call segment callbacks for cached results
			if segmentCallback != nil {
				for _, seg := range cachedSegments {
					segmentCallback(seg)
				}
			}
			if progressCallback != nil {
				progressCallback(fmt.Sprintf("Loaded cached transcription (%d segments)", len(cachedSegments)))
			}
			return cachedSegments, nil
		}
	}

	// Lock model for exclusive access during transcription (whisper_full is not thread-safe)
	e.model.mutex.Lock()
	defer e.model.mutex.Unlock()

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Transcribing with native whisper.cpp (model: %s)...", modelID))
	}

	// Load audio file (we need to convert to 16kHz mono PCM)
	// For now, use ffmpeg to convert if needed
	tempWav, err := prepareAudioFile(audioPath, progressCallback)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare audio: %v", err)
	}
	defer os.Remove(tempWav)

	// Read WAV file
	audioData, sampleRate, err := readWAVFile(tempWav)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio: %v", err)
	}

	if sampleRate != 16000 {
		return nil, fmt.Errorf("sample rate must be 16000 Hz, got %d", sampleRate)
	}

	// Set up whisper parameters
	params := C.whisper_full_default_params(C.WHISPER_SAMPLING_GREEDY)
	params.language = C.CString("he")
	defer C.free(unsafe.Pointer(params.language))
	params.n_threads = C.int(cpuThreads)
	// Enable translation if requested (whisper.cpp translates to English)
	params.translate = C.bool(translateTo == "en")
	params.print_progress = C.bool(false)
	params.print_special = C.bool(false)
	params.print_realtime = C.bool(false)
	params.print_timestamps = C.bool(true)
	// Enable tinydiarize for speaker detection
	params.tdrz_enable = C.bool(true)

	// Set up safe progress tracking using atomic variables
	// C callback writes to atomic (no allocations), Go goroutine reads and updates UI
	var progressPercent int32
	var callbackID uintptr

	if progressCallback != nil {
		callbackRegistryMutex.Lock()
		callbackID = callbackIDCounter
		callbackIDCounter++
		callbackRegistry[callbackID] = &transcriptionCallbacks{
			ctx:             e.model.ctx,
			progressPercent: &progressPercent,
		}
		callbackRegistryMutex.Unlock()

		// Set progress callback in params
		params.progress_callback = C.whisper_progress_callback(C.whisper_progress_callback_go)
		params.progress_callback_user_data = unsafe.Pointer(callbackID)
	}

	// Cleanup callback registration
	defer func() {
		if callbackID != 0 {
			callbackRegistryMutex.Lock()
			delete(callbackRegistry, callbackID)
			callbackRegistryMutex.Unlock()
		}
	}()

	// Convert audio to float32 samples
	samples := make([]float32, len(audioData)/2)
	for i := 0; i < len(samples); i++ {
		// Convert int16 to float32 (-1.0 to 1.0)
		sample := int16(audioData[i*2]) | int16(audioData[i*2+1])<<8
		samples[i] = float32(sample) / 32768.0
	}

	// Run inference
	if progressCallback != nil {
		progressCallback("Starting transcription...")
	}

	// Start progress polling goroutine (safe - only reads atomic, doesn't interact with C)
	done := make(chan bool, 1)
	if progressCallback != nil {
		go func() {
			ticker := time.NewTicker(200 * time.Millisecond) // Poll 5 times per second
			defer ticker.Stop()
			lastProgress := int32(-1)

			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					// Read progress from atomic (written by C callback)
					currentProgress := atomic.LoadInt32(&progressPercent)

					// Only update if progress changed
					if currentProgress != lastProgress {
						lastProgress = currentProgress
						if currentProgress > 0 && currentProgress <= 100 {
							progressCallback(fmt.Sprintf("Transcribing... %d%%", currentProgress))
						}
					}
				}
			}
		}()
	}

	// Call whisper_full - C callback updates atomic, goroutine reads it and updates UI
	result := C.whisper_full(e.model.ctx, params, (*C.float)(unsafe.Pointer(&samples[0])), C.int(len(samples)))

	// Stop progress polling
	if progressCallback != nil {
		close(done)
		time.Sleep(100 * time.Millisecond) // Give goroutine time to exit
	}

	if result != 0 {
		return nil, fmt.Errorf("whisper_full failed with code %d", result)
	}

	// Extract all segments
	segments := []Segment{}
	nSegments := int(C.whisper_full_n_segments(e.model.ctx))

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Processing %d segments...", nSegments))
	}

	// Track speaker changes using tinydiarize
	currentSpeaker := 0
	for i := 0; i < nSegments; i++ {
		// Check if this segment has a speaker turn (next segment switches speaker)
		if i > 0 {
			speakerTurn := C.whisper_full_get_segment_speaker_turn_next(e.model.ctx, C.int(i-1))
			if speakerTurn {
				currentSpeaker++
			}
		}

		t0 := C.whisper_full_get_segment_t0(e.model.ctx, C.int(i))
		t1 := C.whisper_full_get_segment_t1(e.model.ctx, C.int(i))
		textPtr := C.whisper_full_get_segment_text(e.model.ctx, C.int(i))
		// Convert C string to Go string - C.GoString handles UTF-8 correctly
		// whisper.cpp returns UTF-8 encoded text
		text := C.GoString(textPtr)

		// Ensure text is valid UTF-8 by converting to runes and back
		// This validates and fixes any encoding issues
		runes := []rune(text)
		text = string(runes)

		segment := Segment{
			Start:   float64(t0) / 100.0, // Convert from centiseconds to seconds
			End:     float64(t1) / 100.0,
			Text:    text,          // Store as UTF-8 string
			Speaker: currentSpeaker, // Speaker ID from tinydiarize
		}
		segments = append(segments, segment)

		// Call segment callback for UI updates (now safe - not in C callback context)
		if segmentCallback != nil {
			segmentCallback(segment)
		}
	}

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Transcription complete (%d segments)", len(segments)))
	}

	// Cache the transcription result (only for non-translated transcriptions)
	if translateTo == "" {
		cacheKey := transcriptionCacheKey{audioPath: audioPath, modelID: modelID}
		transcriptionCacheMutex.Lock()
		transcriptionCache[cacheKey] = segments
		transcriptionCacheMutex.Unlock()
	}

	return segments, nil
}

// Close releases resources (but keeps cached models)
func (e *WhisperCGOEngine) Close() {
	// Don't free cached models, they'll be reused
	// Only set model reference to nil to prevent double-free
	if !e.fromCache && e.model != nil && e.model.ctx != nil {
		// This was a non-cached model (shouldn't happen with current logic)
		C.whisper_free(e.model.ctx)
	}
	e.model = nil
}

// ClearModelCache clears all cached models (call on app shutdown)
func ClearModelCache() {
	modelCacheMutex.Lock()
	defer modelCacheMutex.Unlock()

	for path, cachedMdl := range modelCache {
		if cachedMdl != nil && cachedMdl.ctx != nil {
			// Lock before freeing (in case transcription is in progress)
			cachedMdl.mutex.Lock()
			C.whisper_free(cachedMdl.ctx)
			cachedMdl.mutex.Unlock()
		}
		delete(modelCache, path)
	}
}

// prepareAudioFile converts audio to 16kHz mono WAV using ffmpeg
func prepareAudioFile(audioPath string, progressCallback func(string)) (string, error) {
	if progressCallback != nil {
		progressCallback("Preparing audio file...")
	}

	// Create temporary WAV file
	tempFile, err := os.CreateTemp("", "whisper_audio_*.wav")
	if err != nil {
		return "", err
	}
	tempPath := tempFile.Name()
	tempFile.Close()

	// Use ffmpeg to convert
	cmd := exec.Command("ffmpeg",
		"-i", audioPath,
		"-ar", "16000",    // 16kHz sample rate
		"-ac", "1",        // Mono
		"-f", "wav",       // WAV format
		"-y",              // Overwrite
		tempPath,
	)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Remove(tempPath)
		return "", fmt.Errorf("ffmpeg conversion failed: %v", err)
	}

	return tempPath, nil
}

// readWAVFile reads a WAV file and returns PCM data and sample rate
func readWAVFile(wavPath string) ([]byte, int, error) {
	data, err := os.ReadFile(wavPath)
	if err != nil {
		return nil, 0, err
	}

	// Simple WAV parser (assumes standard 44-byte header)
	if len(data) < 44 {
		return nil, 0, fmt.Errorf("WAV file too short")
	}

	// Check RIFF header
	if string(data[0:4]) != "RIFF" {
		return nil, 0, fmt.Errorf("not a valid WAV file")
	}

	// Check WAVE format
	if string(data[8:12]) != "WAVE" {
		return nil, 0, fmt.Errorf("not a valid WAV file")
	}

	// Get sample rate (bytes 24-27, little-endian)
	sampleRate := int(data[24]) | int(data[25])<<8 | int(data[26])<<16 | int(data[27])<<24

	// Find data chunk
	dataOffset := 44
	for i := 12; i < len(data)-8; i++ {
		if string(data[i:i+4]) == "data" {
			dataOffset = i + 8
			break
		}
	}

	// Extract PCM data
	pcmData := data[dataOffset:]

	return pcmData, sampleRate, nil
}

// GetModelPath is now in model_download.go with auto-download support

