package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// MistralTranslator handles translation using Mistral 8B via ollama
type MistralTranslator struct {
	ollamaURL string
	model     string
}

// NewMistralTranslator creates a new Mistral translator
func NewMistralTranslator() *MistralTranslator {
	return &MistralTranslator{
		ollamaURL: "http://localhost:11434/api/generate",
		model:     "mistral:latest",
	}
}

// OllamaRequest represents the request to ollama API
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaResponse represents the response from ollama API
type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// Translate translates text from Hebrew to target language
func (t *MistralTranslator) Translate(text string, targetLang string, progressCallback func(string)) (string, error) {
	if text == "" {
		return "", nil
	}

	// Build prompt
	langNames := map[string]string{
		"en": "English",
		"es": "Spanish",
		"fr": "French",
		"de": "German",
		"ar": "Arabic",
		"ru": "Russian",
		"zh": "Chinese",
	}

	langName := langNames[targetLang]
	if langName == "" {
		langName = targetLang
	}

	prompt := fmt.Sprintf(`Translate the following Hebrew text to %s. Only output the translation, nothing else. Keep the formatting the same including timecodes.

Hebrew text: %s

%s translation:`, langName, text, langName)

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Translating to %s...", langName))
	}

	// Make request to ollama
	reqBody := OllamaRequest{
		Model:  t.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(t.ollamaURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to connect to ollama: %v (is ollama running?)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	translation := strings.TrimSpace(ollamaResp.Response)
	return translation, nil
}

// TranslateSegments translates multiple segments
func (t *MistralTranslator) TranslateSegments(segments []Segment, targetLang string, progressCallback func(string), segmentCallback func(Segment)) ([]Segment, error) {
	translatedSegments := make([]Segment, len(segments))

	for i, seg := range segments {
		if progressCallback != nil {
			progressCallback(fmt.Sprintf("Translating segment %d/%d...", i+1, len(segments)))
		}

		translation, err := t.Translate(seg.Text, targetLang, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to translate segment %d: %v", i+1, err)
		}

		translatedSeg := Segment{
			Start:       seg.Start,
			End:         seg.End,
			Text:        translation,
			Original:    seg.Text, // Keep original Hebrew
			Translation: translation,
		}
		translatedSegments[i] = translatedSeg

		if segmentCallback != nil {
			segmentCallback(translatedSeg)
		}
	}

	return translatedSegments, nil
}
