package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadModelsConfig tests loading default model configuration
func TestLoadModelsConfig(t *testing.T) {
	models := loadModelsConfig()

	// Should have at least the default models
	expectedModels := []string{"large-v3", "turbo", "base"}

	for _, modelID := range expectedModels {
		model, exists := models[modelID]
		if !exists {
			t.Errorf("Expected model %q to exist in configuration", modelID)
			continue
		}

		// Validate required fields
		if model.ID == "" {
			t.Errorf("Model %q should have an ID", modelID)
		}
		if model.File == "" {
			t.Errorf("Model %q should have a File", modelID)
		}
	}
}

// TestLoadCustomModelsConfig tests loading custom model configuration from file
func TestLoadCustomModelsConfig(t *testing.T) {
	// Create a temporary models.json file
	customConfig := ModelsConfig{
		Models: map[string]ModelInfo{
			"test-model": {
				ID:            "test-org/test-model",
				File:          "test-model.bin",
				LocalFileName: "test-custom.bin",
				Description:   "Test model",
			},
		},
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "ivrit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write custom config
	configPath := filepath.Join(tmpDir, "models.json")
	configData, err := json.MarshalIndent(customConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Load config
	models := loadModelsConfig()

	// Should have the custom model
	testModel, exists := models["test-model"]
	if !exists {
		t.Error("Custom model should be loaded from config file")
		return
	}

	if testModel.ID != "test-org/test-model" {
		t.Errorf("Expected ID 'test-org/test-model', got %q", testModel.ID)
	}

	if testModel.LocalFileName != "test-custom.bin" {
		t.Errorf("Expected LocalFileName 'test-custom.bin', got %q", testModel.LocalFileName)
	}
}

// TestModelInfoValidation tests ModelInfo struct validation
func TestModelInfoValidation(t *testing.T) {
	tests := []struct {
		name      string
		modelInfo ModelInfo
		valid     bool
	}{
		{
			name: "Valid model",
			modelInfo: ModelInfo{
				ID:            "org/model",
				File:          "model.bin",
				LocalFileName: "local.bin",
			},
			valid: true,
		},
		{
			name: "Missing ID",
			modelInfo: ModelInfo{
				File:          "model.bin",
				LocalFileName: "local.bin",
			},
			valid: false,
		},
		{
			name: "Missing File",
			modelInfo: ModelInfo{
				ID:            "org/model",
				LocalFileName: "local.bin",
			},
			valid: false,
		},
		{
			name: "LocalFileName optional",
			modelInfo: ModelInfo{
				ID:   "org/model",
				File: "model.bin",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.modelInfo.ID != "" && tt.modelInfo.File != ""
			if isValid != tt.valid {
				t.Errorf("Model validation = %v, expected %v", isValid, tt.valid)
			}
		})
	}
}

// TestDefaultModelConfiguration tests that default models are properly configured
func TestDefaultModelConfiguration(t *testing.T) {
	models := loadModelsConfig()

	// Test large-v3 model
	largeV3, exists := models["large-v3"]
	if !exists {
		t.Fatal("large-v3 model should exist in defaults")
	}

	if largeV3.ID != "ivrit-ai/whisper-large-v3-ggml" {
		t.Errorf("large-v3 ID = %q, expected 'ivrit-ai/whisper-large-v3-ggml'", largeV3.ID)
	}

	if largeV3.LocalFileName != "ggml-large-v3-ivrit.bin" {
		t.Errorf("large-v3 LocalFileName = %q, expected 'ggml-large-v3-ivrit.bin'", largeV3.LocalFileName)
	}

	// Test turbo model
	turbo, exists := models["turbo"]
	if !exists {
		t.Fatal("turbo model should exist in defaults")
	}

	if turbo.ID != "ivrit-ai/whisper-large-v3-turbo-ggml" {
		t.Errorf("turbo ID = %q, expected 'ivrit-ai/whisper-large-v3-turbo-ggml'", turbo.ID)
	}

	// Test base model
	base, exists := models["base"]
	if !exists {
		t.Fatal("base model should exist in defaults")
	}

	if base.ID != "ggerganov/whisper.cpp" {
		t.Errorf("base ID = %q, expected 'ggerganov/whisper.cpp'", base.ID)
	}
}

// TestJSONUnmarshal tests that ModelsConfig can be properly unmarshaled
func TestJSONUnmarshal(t *testing.T) {
	jsonData := `{
		"models": {
			"test": {
				"id": "org/model",
				"file": "model.bin",
				"localFileName": "local.bin",
				"description": "Test model"
			}
		}
	}`

	var config ModelsConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(config.Models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(config.Models))
	}

	testModel, exists := config.Models["test"]
	if !exists {
		t.Fatal("test model should exist after unmarshal")
	}

	if testModel.Description != "Test model" {
		t.Errorf("Expected description 'Test model', got %q", testModel.Description)
	}
}

// TestJSONMarshal tests that ModelsConfig can be properly marshaled
func TestJSONMarshal(t *testing.T) {
	config := ModelsConfig{
		Models: map[string]ModelInfo{
			"test": {
				ID:          "org/model",
				File:        "model.bin",
				Description: "Test model",
			},
		},
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Unmarshal back to verify
	var decoded ModelsConfig
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(decoded.Models) != 1 {
		t.Errorf("Expected 1 model after round-trip, got %d", len(decoded.Models))
	}
}
