package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// getAudioDuration gets the duration of an audio/video file using ffprobe
func getAudioDuration(filePath string) (float64, error) {
	// Use ffprobe to get duration
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		filePath,
	)
	
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	
	// Parse JSON output
	var data struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	
	if err := json.Unmarshal(output, &data); err != nil {
		return 0, err
	}
	
	if data.Format.Duration == "" {
		return 0, fmt.Errorf("duration not found")
	}
	
	duration, err := strconv.ParseFloat(data.Format.Duration, 64)
	if err != nil {
		return 0, err
	}
	
	return duration, nil
}

