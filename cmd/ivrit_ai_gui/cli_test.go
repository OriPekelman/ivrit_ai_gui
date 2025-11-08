package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLIHelp tests that the CLI shows help when -help flag is used
func TestCLIHelp(t *testing.T) {
	// Skip if binary doesn't exist
	binaryPath := "./ivrit_ai"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Binary not built yet. Run ./build.sh first.")
	}

	cmd := exec.Command(binaryPath, "-help")
	output, err := cmd.CombinedOutput()

	// Should exit with code 0 for help
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 0 {
				t.Errorf("Expected exit code 0 for -help, got %d", exitErr.ExitCode())
			}
		}
	}

	outputStr := string(output)

	// Check that help contains expected information
	expectedStrings := []string{
		"ivrit.ai Hebrew Transcription CLI",
		"Usage:",
		"-input",
		"-output",
		"-model",
		"-format",
		"Examples:",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Help output should contain %q", expected)
		}
	}
}

// TestCLINoInput tests that CLI exits with error when no input is provided
func TestCLINoInput(t *testing.T) {
	binaryPath := "./ivrit_ai"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Binary not built yet. Run ./build.sh first.")
	}

	cmd := exec.Command(binaryPath, "-model", "turbo")
	output, err := cmd.CombinedOutput()

	// Should exit with non-zero code
	if err == nil {
		t.Error("Expected error when no input file specified")
	}

	outputStr := string(output)
	// Should show help when no input
	if !strings.Contains(outputStr, "Usage:") {
		t.Error("Should show usage when no input file specified")
	}
}

// TestCLIInvalidFile tests that CLI exits with error for non-existent file
func TestCLIInvalidFile(t *testing.T) {
	binaryPath := "./ivrit_ai"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Binary not built yet. Run ./build.sh first.")
	}

	cmd := exec.Command(binaryPath, "-input", "nonexistent_file.mp3")
	output, err := cmd.CombinedOutput()

	// Should exit with error
	if err == nil {
		t.Error("Expected error for non-existent input file")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "does not exist") {
		t.Errorf("Expected 'does not exist' error, got: %s", outputStr)
	}
}

// TestCLIInvalidModel tests that CLI validates model names
func TestCLIInvalidModel(t *testing.T) {
	binaryPath := "./ivrit_ai"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Binary not built yet. Run ./build.sh first.")
	}

	// Create a temporary test file
	tmpFile, err := os.CreateTemp("", "test_audio_*.wav")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cmd := exec.Command(binaryPath, "-input", tmpFile.Name(), "-model", "invalid-model")
	output, err := cmd.CombinedOutput()

	// Should exit with error
	if err == nil {
		t.Error("Expected error for invalid model")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Invalid model") {
		t.Errorf("Expected 'Invalid model' error, got: %s", outputStr)
	}
}

// TestCLIInvalidFormat tests that CLI validates output formats
func TestCLIInvalidFormat(t *testing.T) {
	binaryPath := "./ivrit_ai"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Binary not built yet. Run ./build.sh first.")
	}

	// Create a temporary test file
	tmpFile, err := os.CreateTemp("", "test_audio_*.wav")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cmd := exec.Command(binaryPath, "-input", tmpFile.Name(), "-format", "invalid-format")
	output, err := cmd.CombinedOutput()

	// Should exit with error
	if err == nil {
		t.Error("Expected error for invalid format")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Invalid format") {
		t.Errorf("Expected 'Invalid format' error, got: %s", outputStr)
	}
}

// TestAutoOutputFilename tests automatic output filename generation
func TestAutoOutputFilename(t *testing.T) {
	tests := []struct {
		name          string
		inputFile     string
		format        string
		expectedExt   string
	}{
		{"Text format", "test.mp3", "text", ".txt"},
		{"JSON format", "test.wav", "json", ".json"},
		{"SRT format", "test.m4a", "srt", ".srt"},
		{"VTT format", "test.mp4", "vtt", ".vtt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the auto-filename logic from cli.go
			ext := "txt"
			switch tt.format {
			case "json":
				ext = "json"
			case "srt":
				ext = "srt"
			case "vtt":
				ext = "vtt"
			}
			base := filepath.Base(tt.inputFile)
			outputFile := base[:len(base)-len(filepath.Ext(base))] + "_transcription." + ext

			if !strings.HasSuffix(outputFile, tt.expectedExt) {
				t.Errorf("Expected output file to have extension %s, got %s", tt.expectedExt, outputFile)
			}

			if !strings.Contains(outputFile, "_transcription") {
				t.Error("Output filename should contain '_transcription'")
			}
		})
	}
}

// TestModelValidation tests model name validation
func TestModelValidation(t *testing.T) {
	validModels := map[string]bool{"large-v3": true, "turbo": true, "base": true}

	tests := []struct {
		model string
		valid bool
	}{
		{"large-v3", true},
		{"turbo", true},
		{"base", true},
		{"invalid", false},
		{"TURBO", false},      // Case sensitive
		{"large", false},      // Incomplete name
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			isValid := validModels[tt.model]
			if isValid != tt.valid {
				t.Errorf("Model %q validation = %v, expected %v", tt.model, isValid, tt.valid)
			}
		})
	}
}

// TestFormatValidation tests output format validation
func TestFormatValidation(t *testing.T) {
	validFormats := map[string]bool{"text": true, "json": true, "srt": true, "vtt": true}

	tests := []struct {
		format string
		valid  bool
	}{
		{"text", true},
		{"json", true},
		{"srt", true},
		{"vtt", true},
		{"txt", false},
		{"TEXT", false},       // Case sensitive
		{"xml", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			isValid := validFormats[tt.format]
			if isValid != tt.valid {
				t.Errorf("Format %q validation = %v, expected %v", tt.format, isValid, tt.valid)
			}
		})
	}
}
