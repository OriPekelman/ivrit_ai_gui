package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"unicode"
)

// Segment represents a transcribed segment with timing information
type Segment struct {
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
	Text        string  `json:"text"`
	Original    string  `json:"original,omitempty"`    // Original Hebrew text (if translated)
	Translation string  `json:"translation,omitempty"` // English translation (if requested)
	Speaker     int     `json:"speaker,omitempty"`     // Speaker ID (0, 1, 2, etc.) from tinydiarize
}

// TranscriptionEngine interface for different transcription backends
type TranscriptionEngine interface {
	Transcribe(audioPath string, modelID string, cpuThreads int, progressCallback func(string), segmentCallback func(Segment)) ([]Segment, error)
	SupportsModel(modelID string) bool
}

// ProgressCallback is called for status updates
type ProgressCallback func(message string, percentage int)

// SegmentCallback is called for each transcribed segment (for streaming)
type SegmentCallback func(segment Segment)

// IsVideoFile checks if a file is a video based on extension
func IsVideoFile(filePath string) bool {
	videoExts := map[string]bool{
		".mp4": true, ".avi": true, ".mov": true, ".mkv": true,
		".webm": true, ".flv": true, ".wmv": true, ".m4v": true, ".3gp": true,
	}
	ext := filepath.Ext(filePath)
	return videoExts[ext]
}

// ExtractAudioFromVideo extracts audio from video file using ffmpeg
func ExtractAudioFromVideo(videoPath string, progressCallback ProgressCallback) (string, error) {
	if progressCallback != nil {
		progressCallback("Extracting audio from video...", -1)
	}

	// Create temporary file for audio
	tempFile, err := os.CreateTemp("", "extracted_audio_*.wav")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close()

	// Use ffmpeg to extract audio
	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-vn",              // No video
		"-acodec", "pcm_s16le", // PCM 16-bit
		"-ar", "16000",      // 16kHz sample rate (optimal for Whisper)
		"-ac", "1",          // Mono
		"-y",                // Overwrite output file
		tempPath,
	)

	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Remove(tempPath)
		return "", fmt.Errorf("ffmpeg failed: %v (is ffmpeg installed?)", err)
	}

	return tempPath, nil
}

// FormatTimestamp formats seconds to SRT timestamp format
func FormatTimestamp(seconds float64, vtt bool) string {
	hours := int(seconds / 3600)
	minutes := int((int(seconds) % 3600) / 60)
	secs := int(seconds) % 60
	millis := int((seconds - float64(int(seconds))) * 1000)

	if vtt {
		return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, secs, millis)
	}
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secs, millis)
}

// FormatOutput formats segments into different output formats
func FormatOutput(segments []Segment, formatType string, includeOriginal bool) string {
	switch formatType {
	case "text":
		output := ""
		lastSpeaker := -1
		for _, seg := range segments {
			// Add speaker label if speaker changed
			speakerPrefix := ""
			if seg.Speaker != lastSpeaker {
				speakerPrefix = fmt.Sprintf("Speaker %d: ", seg.Speaker+1)
				lastSpeaker = seg.Speaker
			}

			// If both original and translation exist, show both
			if seg.Original != "" && seg.Translation != "" {
				// Add RTL markers for Hebrew text
				original := "\u202B" + seg.Original + "\u202C"
				output += speakerPrefix + original + "\n"
				if speakerPrefix != "" {
					output += "           " // Indent translation to align
				}
				output += seg.Translation + "\n\n"
			} else {
				// Just show text (either Hebrew or translated)
				text := seg.Text
				if containsHebrew(text) {
					text = "\u202B" + text + "\u202C"
				}
				output += speakerPrefix + text + "\n"
			}
		}
		return output

	case "json":
		// JSON output with separate original and translation fields
		output := "[\n"
		for i, seg := range segments {
			if i > 0 {
				output += ",\n"
			}
			output += "  {"
			output += fmt.Sprintf(`"start": %.2f, "end": %.2f, "speaker": %d`, seg.Start, seg.End, seg.Speaker+1)
			if seg.Translation != "" && seg.Original != "" {
				// Both original and translation present
				output += fmt.Sprintf(`, "original": "%s", "translation": "%s"`, seg.Original, seg.Translation)
			} else {
				// Just text (either Hebrew or English)
				output += fmt.Sprintf(`, "text": "%s"`, seg.Text)
			}
			output += "}"
		}
		output += "\n]"
		return output

	case "srt":
		output := ""
		lastSpeaker := -1
		for i, seg := range segments {
			start := FormatTimestamp(seg.Start, false)
			end := FormatTimestamp(seg.End, false)

			// Add speaker label if speaker changed
			speakerLabel := ""
			if seg.Speaker != lastSpeaker {
				speakerLabel = fmt.Sprintf("[Speaker %d] ", seg.Speaker+1)
				lastSpeaker = seg.Speaker
			}

			// If both original and translation exist, show both on separate lines
			if seg.Original != "" && seg.Translation != "" {
				output += fmt.Sprintf("%d\n%s --> %s\n%s%s\n%s\n\n", i+1, start, end, speakerLabel, seg.Original, seg.Translation)
			} else {
				output += fmt.Sprintf("%d\n%s --> %s\n%s%s\n\n", i+1, start, end, speakerLabel, seg.Text)
			}
		}
		return output

	case "vtt":
		output := "WEBVTT\n\n"
		lastSpeaker := -1
		for _, seg := range segments {
			start := FormatTimestamp(seg.Start, true)
			end := FormatTimestamp(seg.End, true)

			// Add speaker label if speaker changed
			speakerLabel := ""
			if seg.Speaker != lastSpeaker {
				speakerLabel = fmt.Sprintf("<v Speaker %d>", seg.Speaker+1)
				lastSpeaker = seg.Speaker
			}

			// If both original and translation exist, show both on separate lines
			if seg.Original != "" && seg.Translation != "" {
				output += fmt.Sprintf("%s --> %s\n%s%s\n%s\n\n", start, end, speakerLabel, seg.Original, seg.Translation)
			} else {
				output += fmt.Sprintf("%s --> %s\n%s%s\n\n", start, end, speakerLabel, seg.Text)
			}
		}
		return output

	default:
		return ""
	}
}

// GetOptimalCPUThreads returns optimal number of CPU threads
func GetOptimalCPUThreads() int {
	cpuCount := runtime.NumCPU()
	// Default to 8 threads (optimal for turbo model), but cap at available cores
	if cpuCount < 8 {
		return cpuCount
	}
	return 8
}

// containsHebrew checks if a string contains Hebrew characters
func containsHebrew(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Hebrew) {
			return true
		}
	}
	return false
}

