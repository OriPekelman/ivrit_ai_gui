package main

import (
	"strings"
	"testing"
)

// Test containsHebrew function
func TestContainsHebrew(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Hebrew text", "שלום עולם", true},
		{"English text", "Hello world", false},
		{"Mixed text", "Hello שלום", true},
		{"Empty string", "", false},
		{"Numbers", "12345", false},
		{"Hebrew with punctuation", "שלום, מה שלומך?", true},
		{"English with punctuation", "Hello, how are you?", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsHebrew(tt.input)
			if result != tt.expected {
				t.Errorf("containsHebrew(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Test IsVideoFile function
func TestIsVideoFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{"MP4 file", "test.mp4", true},
		{"AVI file", "test.avi", true},
		{"MOV file", "test.mov", true},
		{"MKV file", "test.mkv", true},
		{"M4V file", "video.m4v", true},
		{"MP3 audio", "audio.mp3", false},
		{"WAV audio", "audio.wav", false},
		{"M4A audio", "audio.m4a", false},
		{"No extension", "testfile", false},
		{"Path with video ext", "/path/to/video.mp4", true},
		{"Upper case ext", "VIDEO.MP4", false}, // Extensions are case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsVideoFile(tt.filePath)
			if result != tt.expected {
				t.Errorf("IsVideoFile(%q) = %v, expected %v", tt.filePath, result, tt.expected)
			}
		})
	}
}

// Test FormatTimestamp function
func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		seconds  float64
		vtt      bool
		expected string
	}{
		{"Zero seconds SRT", 0.0, false, "00:00:00,000"},
		{"Zero seconds VTT", 0.0, true, "00:00:00.000"},
		{"One minute SRT", 60.0, false, "00:01:00,000"},
		{"One minute VTT", 60.0, true, "00:01:00.000"},
		{"One hour SRT", 3600.0, false, "01:00:00,000"},
		{"One hour VTT", 3600.0, true, "01:00:00.000"},
		{"With milliseconds SRT", 123.456, false, "00:02:03,456"},
		{"With milliseconds VTT", 123.456, true, "00:02:03.456"},
		{"Complex time SRT", 3723.789, false, "01:02:03,789"},
		{"Complex time VTT", 3723.789, true, "01:02:03.789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTimestamp(tt.seconds, tt.vtt)
			if result != tt.expected {
				t.Errorf("FormatTimestamp(%v, %v) = %q, expected %q", tt.seconds, tt.vtt, result, tt.expected)
			}
		})
	}
}

// Test FormatOutput function with text format
func TestFormatOutputText(t *testing.T) {
	segments := []Segment{
		{Start: 0.0, End: 2.5, Text: "שלום עולם", Speaker: 0},
		{Start: 2.5, End: 5.0, Text: "מה שלומך?", Speaker: 1},
		{Start: 5.0, End: 7.5, Text: "בסדר גמור", Speaker: 0},
	}

	output := FormatOutput(segments, "text", false)

	// Check that output contains speaker labels
	if !strings.Contains(output, "Speaker 1:") {
		t.Error("Output should contain 'Speaker 1:'")
	}
	if !strings.Contains(output, "Speaker 2:") {
		t.Error("Output should contain 'Speaker 2:'")
	}

	// Check that Hebrew text is present (with RTL markers)
	if !strings.Contains(output, "שלום עולם") {
		t.Error("Output should contain Hebrew text")
	}
}

// Test FormatOutput with translation
func TestFormatOutputTextWithTranslation(t *testing.T) {
	segments := []Segment{
		{
			Start:       0.0,
			End:         2.5,
			Text:        "Hello world",
			Original:    "שלום עולם",
			Translation: "Hello world",
			Speaker:     0,
		},
	}

	output := FormatOutput(segments, "text", true)

	// Should contain both original and translation
	if !strings.Contains(output, "שלום עולם") {
		t.Error("Output should contain original Hebrew text")
	}
	if !strings.Contains(output, "Hello world") {
		t.Error("Output should contain translation")
	}
}

// Test FormatOutput with JSON format
func TestFormatOutputJSON(t *testing.T) {
	segments := []Segment{
		{Start: 0.0, End: 2.5, Text: "שלום", Speaker: 0},
		{Start: 2.5, End: 5.0, Text: "עולם", Speaker: 1},
	}

	output := FormatOutput(segments, "json", false)

	// Check JSON structure
	if !strings.HasPrefix(output, "[") || !strings.HasSuffix(output, "]") {
		t.Error("JSON output should be wrapped in array brackets")
	}

	// Check for required fields
	if !strings.Contains(output, `"start":`) {
		t.Error("JSON should contain start field")
	}
	if !strings.Contains(output, `"end":`) {
		t.Error("JSON should contain end field")
	}
	if !strings.Contains(output, `"speaker":`) {
		t.Error("JSON should contain speaker field")
	}
	if !strings.Contains(output, `"text":`) {
		t.Error("JSON should contain text field")
	}
}

// Test FormatOutput with SRT format
func TestFormatOutputSRT(t *testing.T) {
	segments := []Segment{
		{Start: 0.0, End: 2.5, Text: "שלום עולם", Speaker: 0},
		{Start: 2.5, End: 5.0, Text: "מה שלומך?", Speaker: 0},
	}

	output := FormatOutput(segments, "srt", false)

	// Check for SRT structure
	if !strings.Contains(output, "1\n") {
		t.Error("SRT should start with subtitle number")
	}

	// Check for timestamp format (SRT uses comma for milliseconds)
	if !strings.Contains(output, "00:00:00,000 --> 00:00:02,500") {
		t.Error("SRT should contain proper timestamp format with comma")
	}

	// Check for text content
	if !strings.Contains(output, "שלום עולם") {
		t.Error("SRT should contain Hebrew text")
	}
}

// Test FormatOutput with VTT format
func TestFormatOutputVTT(t *testing.T) {
	segments := []Segment{
		{Start: 0.0, End: 2.5, Text: "שלום עולם", Speaker: 0},
	}

	output := FormatOutput(segments, "vtt", false)

	// Check for VTT header
	if !strings.HasPrefix(output, "WEBVTT\n\n") {
		t.Error("VTT should start with WEBVTT header")
	}

	// Check for timestamp format (VTT uses period for milliseconds)
	if !strings.Contains(output, "00:00:00.000 --> 00:00:02.500") {
		t.Error("VTT should contain proper timestamp format with period")
	}
}

// Test GetOptimalCPUThreads
func TestGetOptimalCPUThreads(t *testing.T) {
	threads := GetOptimalCPUThreads()

	// Should return a positive number
	if threads <= 0 {
		t.Errorf("GetOptimalCPUThreads() returned %d, expected positive number", threads)
	}

	// Should not exceed 8 (as per the implementation)
	if threads > 8 {
		t.Errorf("GetOptimalCPUThreads() returned %d, expected <= 8", threads)
	}
}

// Benchmark for containsHebrew
func BenchmarkContainsHebrew(b *testing.B) {
	testString := "שלום עולם - Hello world - שלום"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		containsHebrew(testString)
	}
}

// Benchmark for FormatTimestamp
func BenchmarkFormatTimestamp(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FormatTimestamp(3723.789, false)
	}
}
