package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// CLIMode runs the application in command-line mode
func CLIMode() {
	// Define command-line flags
	audioFile := flag.String("input", "", "Input audio/video file path (required)")
	outputFile := flag.String("output", "", "Output file path (default: transcription.txt)")
	modelID := flag.String("model", "turbo", "Model to use: large-v3, turbo, or base")
	format := flag.String("format", "text", "Output format: text, json, srt, or vtt")
	translate := flag.Bool("translate", false, "Translate to English using Mistral 8B")
	targetLang := flag.String("lang", "en", "Target language for translation: en, es, fr, de")
	keepOriginal := flag.Bool("keep-original", true, "Keep original Hebrew text when translating")
	cpuThreads := flag.Int("threads", 0, "Number of CPU threads (0 = auto)")
	help := flag.Bool("help", false, "Show help message")

	flag.Parse()

	// Show help
	if *help || *audioFile == "" {
		fmt.Println("ivrit.ai Hebrew Transcription CLI")
		fmt.Println("\nUsage:")
		fmt.Printf("  %s -input <audio-file> [options]\n\n", os.Args[0])
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Printf("  %s -input recording.m4a\n", os.Args[0])
		fmt.Printf("  %s -input video.mp4 -model large-v3 -format srt -output subtitles.srt\n", os.Args[0])
		fmt.Printf("  %s -input audio.wav -translate -lang en -keep-original=false\n", os.Args[0])
		if *audioFile == "" {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Validate input file
	if _, err := os.Stat(*audioFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input file does not exist: %s\n", *audioFile)
		os.Exit(1)
	}

	// Auto-detect output file name if not specified
	if *outputFile == "" {
		ext := "txt"
		switch *format {
		case "json":
			ext = "json"
		case "srt":
			ext = "srt"
		case "vtt":
			ext = "vtt"
		}
		base := filepath.Base(*audioFile)
		*outputFile = base[:len(base)-len(filepath.Ext(base))] + "_transcription." + ext
	}

	// Validate model
	validModels := map[string]bool{"large-v3": true, "turbo": true, "base": true}
	if !validModels[*modelID] {
		fmt.Fprintf(os.Stderr, "Error: Invalid model '%s'. Valid options: large-v3, turbo, base\n", *modelID)
		os.Exit(1)
	}

	// Validate format
	validFormats := map[string]bool{"text": true, "json": true, "srt": true, "vtt": true}
	if !validFormats[*format] {
		fmt.Fprintf(os.Stderr, "Error: Invalid format '%s'. Valid options: text, json, srt, vtt\n", *format)
		os.Exit(1)
	}

	// Determine CPU threads
	threads := *cpuThreads
	if threads == 0 {
		threads = GetOptimalCPUThreads()
	}

	fmt.Printf("Starting transcription...\n")
	fmt.Printf("  Input:  %s\n", *audioFile)
	fmt.Printf("  Output: %s\n", *outputFile)
	fmt.Printf("  Model:  %s\n", *modelID)
	fmt.Printf("  Format: %s\n", *format)
	fmt.Printf("  Threads: %d\n", threads)
	if *translate {
		fmt.Printf("  Translation: Enabled (target: %s, keep original: %v)\n", *targetLang, *keepOriginal)
	}
	fmt.Println()

	// Progress callback
	progressCallback := func(msg string, pct int) {
		if pct >= 0 {
			fmt.Printf("\r%s (%d%%)  ", msg, pct)
		} else {
			fmt.Printf("\r%s  ", msg)
		}
	}

	// Get model path (will auto-download if needed)
	modelPath, err := GetModelPath(*modelID, progressCallback)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError getting model: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()

	// Initialize whisper engine
	engine, err := NewWhisperCGOEngine(modelPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing whisper engine: %v\n", err)
		os.Exit(1)
	}
	defer engine.Close()

	// Transcribe
	segments, err := engine.Transcribe(*audioFile, *modelID, threads, func(msg string) {
		fmt.Printf("\r%s", msg)
	}, nil)

	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError during transcription: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nTranscription complete (%d segments)\n", len(segments))

	// Translate if requested
	if *translate {
		fmt.Printf("Translating to %s...\n", *targetLang)
		translator := NewMistralTranslator()

		translatedSegments, err := translator.TranslateSegments(segments, *targetLang, func(msg string) {
			fmt.Printf("\r%s", msg)
		}, nil)

		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError during translation: %v\n", err)
			os.Exit(1)
		}

		// Handle keep original setting
		if !*keepOriginal {
			for i := range translatedSegments {
				translatedSegments[i].Text = translatedSegments[i].Translation
				translatedSegments[i].Original = ""
			}
		} else {
			for i := range translatedSegments {
				translatedSegments[i].Text = translatedSegments[i].Translation
			}
		}

		segments = translatedSegments
		fmt.Println("\nTranslation complete")
	}

	// Format output
	outputText := FormatOutput(segments, *format, *keepOriginal)

	// Write to file
	if err := os.WriteFile(*outputFile, []byte(outputText), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Saved to: %s\n", *outputFile)
}
