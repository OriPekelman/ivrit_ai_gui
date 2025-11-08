package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// ModelInfo represents a HuggingFace model
type ModelInfo struct {
	ID            string `json:"id"`
	File          string `json:"file"`
	LocalFileName string `json:"localFileName,omitempty"`
	Description   string `json:"description,omitempty"`
	URL           string `json:"url,omitempty"`
}

// ModelsConfig represents the models configuration file
type ModelsConfig struct {
	Models map[string]ModelInfo `json:"models"`
}

// loadModelsConfig loads model configuration from JSON file if it exists
func loadModelsConfig() map[string]ModelInfo {
	// Try to load from multiple locations
	configPaths := []string{
		"models.json",                           // Current directory
		filepath.Join(".", "models.json"),       // Explicit current dir
		filepath.Join("..", "models.json"),      // Parent directory
		filepath.Join(os.Getenv("HOME"), ".config", "ivrit-ai", "models.json"), // User config
	}

	for _, configPath := range configPaths {
		if data, err := os.ReadFile(configPath); err == nil {
			var config ModelsConfig
			if err := json.Unmarshal(data, &config); err == nil && len(config.Models) > 0 {
				return config.Models
			}
		}
	}

	// Return default hardcoded models if config file not found
	return map[string]ModelInfo{
		"large-v3": {
			ID:            "ivrit-ai/whisper-large-v3-ggml",
			File:          "ggml-model.bin",
			LocalFileName: "ggml-large-v3-ivrit.bin",
			Description:   "Ivrit.ai Large v3 - Best quality for Hebrew",
		},
		"turbo": {
			ID:            "ivrit-ai/whisper-large-v3-turbo-ggml",
			File:          "ggml-model.bin",
			LocalFileName: "ggml-large-v3-turbo-ivrit.bin",
			Description:   "Ivrit.ai Turbo - Faster with good quality",
		},
		"base": {
			ID:            "ggerganov/whisper.cpp",
			File:          "ggml-base.bin",
			LocalFileName: "ggml-base.bin",
			Description:   "Base model - Fast but lower quality",
		},
	}
}

// GetModelPath returns the path to a whisper model file, downloading if needed
func GetModelPath(modelID string, progressCallback func(string, int)) (string, error) {
	// Load model configuration from JSON file or use defaults
	modelMap := loadModelsConfig()

	modelInfo, exists := modelMap[modelID]
	if !exists || modelInfo.ID == "" {
		return "", fmt.Errorf("unsupported model: %s", modelID)
	}

	// Check common model locations
	homeDir, _ := os.UserHomeDir()
	// Use configured local filename or fall back to original file name
	localFileName := modelInfo.LocalFileName
	if localFileName == "" {
		localFileName = modelInfo.File
	}

	possiblePaths := []string{
		filepath.Join(homeDir, ".cache", "whisper", localFileName),
		filepath.Join(homeDir, ".cache", "whisper", modelID+".bin"),
		filepath.Join(homeDir, ".local", "share", "whisper", localFileName),
		filepath.Join("/usr/local/share/whisper", localFileName),
		filepath.Join(".", "models", localFileName),
		filepath.Join(".", localFileName),
	}

	// Check for existing model
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			if progressCallback != nil {
				progressCallback(fmt.Sprintf("Found model at: %s", path), -1)
			}
			return path, nil
		}
	}

	// Model not found - try to download
	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Downloading %s from ivrit.ai...", modelID), 0)
	}

	// Download from HuggingFace
	cacheDir := filepath.Join(homeDir, ".cache", "whisper")
	os.MkdirAll(cacheDir, 0755)
	modelPath := filepath.Join(cacheDir, localFileName)

	err := downloadModelFromHuggingFace(modelInfo.ID, modelInfo.File, modelPath, progressCallback)
	if err != nil {
		// For base model, try direct download as fallback
		if modelID == "base" {
			if progressCallback != nil {
				progressCallback("HuggingFace download failed, trying direct download...", -1)
			}
			err = downloadModelDirect(modelInfo.File, modelPath, progressCallback)
		}

		if err != nil {
			return "", fmt.Errorf(
				"failed to download model: %v\n"+
					"Please manually download from:\n"+
					"  https://huggingface.co/%s\n"+
					"Then place %s in: %s",
				err, modelInfo.ID, localFileName, cacheDir)
		}
	}

	if progressCallback != nil {
		progressCallback("Model downloaded successfully", 100)
	}

	return modelPath, nil
}

// downloadModelFromHuggingFace downloads a model from HuggingFace
func downloadModelFromHuggingFace(repoID, fileName, destPath string, progressCallback func(string, int)) error {
	// HuggingFace API endpoint
	url := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", repoID, fileName)

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Get content length
	contentLength := resp.ContentLength
	if contentLength == 0 {
		contentLength = -1 // Unknown size
	}

	// Create destination file
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Download with progress
	buffer := make([]byte, 32*1024) // 32KB chunks
	var downloaded int64

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			written, writeErr := out.Write(buffer[:n])
			if writeErr != nil {
				return writeErr
			}
			downloaded += int64(written)

			// Update progress
			if progressCallback != nil && contentLength > 0 {
				percent := int((downloaded * 100) / contentLength)
				mbDownloaded := float64(downloaded) / (1024 * 1024)
				mbTotal := float64(contentLength) / (1024 * 1024)
				progressCallback(fmt.Sprintf("Downloading: %.1fMB / %.1fMB", mbDownloaded, mbTotal), percent)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// downloadModelDirect downloads from ggml.ggerganov.com (direct download)
func downloadModelDirect(fileName, destPath string, progressCallback func(string, int)) error {
	// Map file names to direct download URLs
	urlMap := map[string]string{
		"ggml-large-v3.bin": "https://ggml.ggerganov.com/models/whisper/ggml-large-v3.bin",
		"ggml-base.bin":       "https://ggml.ggerganov.com/models/whisper/ggml-base.bin",
	}

	url, ok := urlMap[fileName]
	if !ok {
		return fmt.Errorf("no direct download URL for %s", fileName)
	}

	// Download
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	contentLength := resp.ContentLength
	if contentLength == 0 {
		contentLength = -1
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	buffer := make([]byte, 32*1024)
	var downloaded int64

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			written, writeErr := out.Write(buffer[:n])
			if writeErr != nil {
				return writeErr
			}
			downloaded += int64(written)

			if progressCallback != nil && contentLength > 0 {
				percent := int((downloaded * 100) / contentLength)
				mbDownloaded := float64(downloaded) / (1024 * 1024)
				mbTotal := float64(contentLength) / (1024 * 1024)
				progressCallback(fmt.Sprintf("Downloading: %.1fMB / %.1fMB", mbDownloaded, mbTotal), percent)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

