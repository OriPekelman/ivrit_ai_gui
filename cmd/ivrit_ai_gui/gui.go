package main

// Gio-based GUI implementation with proper Unicode/RTL support
// Gio is actively maintained and has better text rendering than Fyne

import (
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/sqweek/dialog"
)

// GioApp represents the Gio-based application
type GioApp struct {
	window *app.Window
	theme  *material.Theme

	// Widgets
	fileLabel         *widget.Label
	browseBtn         *widget.Clickable
	transcribeBtn     *widget.Clickable
	stopBtn           *widget.Clickable
	saveBtn           *widget.Clickable
	modelList         *widget.Enum
	formatList        *widget.Enum
	enableTranslation *widget.Bool // Enable translation checkbox
	translateLangList *widget.Enum // Target language for translation
	keepOriginal      *widget.Bool // Keep original Hebrew text checkbox

	// Credit links
	ivritLink    *widget.Clickable
	patreonLink  *widget.Clickable
	creditsLink  *widget.Clickable

	// Output
	outputEditor     *widget.Editor // Read-only editor for transcription

	// State
	audioFilePath      string
	transcriptionSegments []Segment
	originalSegments   []Segment
	workerRunning     bool
	workerMutex       sync.Mutex
	stopRequested     bool // Flag to stop transcription
	transcriptionStartTime int64
	audioDuration     float64

	// Status (protected by uiMutex)
	statusText      string
	timingText      string
	progressVisible bool
	uiMutex         sync.RWMutex // Protects statusText, timingText, outputEditor text
}

// NewGioApp creates a new Gio application
func NewGioApp(w *app.Window) *GioApp {
	th := material.NewTheme()
	// Use system fonts to get Hebrew support on macOS (SF Pro, Arial Hebrew, etc)
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	gioApp := &GioApp{
		window:            w,
		theme:             th,
		fileLabel:         &widget.Label{},
		browseBtn:         &widget.Clickable{},
		transcribeBtn:     &widget.Clickable{},
		stopBtn:           &widget.Clickable{},
		saveBtn:           &widget.Clickable{},
		modelList:         &widget.Enum{},
		formatList:        &widget.Enum{},
		enableTranslation: &widget.Bool{},
		translateLangList: &widget.Enum{},
		keepOriginal:      &widget.Bool{Value: true}, // Default to keeping original
		ivritLink:         &widget.Clickable{},
		patreonLink:       &widget.Clickable{},
		creditsLink:       &widget.Clickable{},
		outputEditor:      &widget.Editor{ReadOnly: true, SingleLine: false},
		statusText:        "Ready",
	}

	// Set defaults
	gioApp.modelList.Value = "turbo"
	gioApp.formatList.Value = "text"
	gioApp.translateLangList.Value = "en" // Default to English

	return gioApp
}

// Layout lays out the UI
func (a *GioApp) Layout(gtx layout.Context) layout.Dimensions {
	return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:    layout.Vertical,
			Spacing: layout.SpaceSides,
		}.Layout(gtx,
			// File selection
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, a.layoutFileSelection)
			}),

			// Options
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, a.layoutOptions)
			}),

			// Controls
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, a.layoutControls)
			}),

			// Status
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, a.layoutStatus)
			}),

			// Output (expands)
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return a.layoutOutput(gtx)
			}),

			// Credits
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, a.layoutCredits)
			}),
		)
	})
}

func (a *GioApp) layoutFileSelection(gtx layout.Context) layout.Dimensions {
	// Handle browse button clicks
	for a.browseBtn.Clicked(gtx) {
		go a.selectFile()
	}
	
	return layout.Flex{
		Axis:      layout.Horizontal,
		Spacing:   layout.SpaceBetween,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			fileName := a.audioFilePath
			if fileName == "" {
				fileName = "No file selected"
			} else {
				fileName = filepath.Base(fileName)
				if len(fileName) > 50 {
					fileName = fileName[:50] + "..."
				}
			}
			label := material.Label(a.theme, unit.Sp(14), fileName)
			return label.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(a.theme, a.browseBtn, "Choose audio file to transcribe")
			return btn.Layout(gtx)
		}),
	)
}

func (a *GioApp) layoutOptions(gtx layout.Context) layout.Dimensions {
	return layout.Flex{
		Axis:    layout.Vertical,
		Spacing: layout.SpaceStart,
	}.Layout(gtx,
		// Row 1: Model and Format
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis:    layout.Horizontal,
				Spacing: layout.SpaceStart,
			}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Label(a.theme, unit.Sp(14), "Model:").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.RadioButton(a.theme, a.modelList, "large-v3", "large-v3").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.RadioButton(a.theme, a.modelList, "turbo", "turbo").Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(24)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Label(a.theme, unit.Sp(14), "Format:").Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.RadioButton(a.theme, a.formatList, "text", "text").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.RadioButton(a.theme, a.formatList, "json", "json").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.RadioButton(a.theme, a.formatList, "srt", "srt").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.RadioButton(a.theme, a.formatList, "vtt", "vtt").Layout(gtx)
				}),
			)
		}),
		// Row 2: Translation (via Mistral 8B)
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis:    layout.Horizontal,
				Spacing: layout.SpaceStart,
				Alignment: layout.Middle,
			}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.CheckBox(a.theme, a.enableTranslation, "Enable Translation").Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if a.enableTranslation.Value {
						return layout.Flex{
							Axis:    layout.Horizontal,
							Spacing: layout.SpaceStart,
						}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return material.Label(a.theme, unit.Sp(14), "To:").Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return material.RadioButton(a.theme, a.translateLangList, "en", "English").Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return material.RadioButton(a.theme, a.translateLangList, "es", "Spanish").Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return material.RadioButton(a.theme, a.translateLangList, "fr", "French").Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return material.RadioButton(a.theme, a.translateLangList, "de", "German").Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return material.CheckBox(a.theme, a.keepOriginal, "Keep Hebrew").Layout(gtx)
							}),
						)
					}
					return layout.Dimensions{}
				}),
			)
		}),
	)
}

func (a *GioApp) layoutControls(gtx layout.Context) layout.Dimensions {
	// Handle button clicks
	for a.transcribeBtn.Clicked(gtx) {
		go a.startTranscription()
	}
	for a.stopBtn.Clicked(gtx) {
		go a.stopTranscription()
	}
	for a.saveBtn.Clicked(gtx) {
		go a.saveTranscription()
	}
	
	return layout.Flex{
		Axis:    layout.Horizontal,
		Spacing: layout.SpaceStart,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(a.theme, a.transcribeBtn, "Transcribe")
			btn.Background = color.NRGBA{R: 0, G: 122, B: 255, A: 255}
			return btn.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(a.theme, a.stopBtn, "Stop")
			btn.Background = color.NRGBA{R: 220, G: 53, B: 69, A: 255}
			return btn.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(a.theme, a.saveBtn, "Save As...")
			return btn.Layout(gtx)
		}),
	)
}

func (a *GioApp) layoutStatus(gtx layout.Context) layout.Dimensions {
	// Read UI state with lock to prevent data races
	a.uiMutex.RLock()
	statusText := a.statusText
	timingText := a.timingText
	a.uiMutex.RUnlock()

	return layout.Flex{
		Axis:    layout.Horizontal,
		Spacing: layout.SpaceBetween,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := material.Label(a.theme, unit.Sp(12), statusText)
			return label.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := material.Label(a.theme, unit.Sp(12), timingText)
			label.Color = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
			return label.Layout(gtx)
		}),
	)
}

func (a *GioApp) layoutOutput(gtx layout.Context) layout.Dimensions {
	// Output text area with RTL support
	// Gio's text shaper handles RTL automatically for Hebrew text
	ed := material.Editor(a.theme, a.outputEditor, "Transcription will appear here...")

	// Right-align Hebrew text
	a.uiMutex.RLock()
	currentText := a.outputEditor.Text()
	a.uiMutex.RUnlock()

	if containsHebrew(currentText) {
		ed.Editor.Alignment = text.End // Right-align for RTL Hebrew
	} else {
		ed.Editor.Alignment = text.Start // Left-align for LTR
	}

	// Wrap Layout call with panic recovery to prevent crashes with large text buffers
	var dims layout.Dimensions
	func() {
		defer func() {
			if r := recover(); r != nil {
				// If Layout panics (e.g., text buffer too large), show error in status
				fmt.Fprintf(os.Stderr, "Warning: Text layout error: %v\n", r)
				a.uiMutex.Lock()
				a.statusText = "Warning: Text display limited due to size"
				a.uiMutex.Unlock()
				// Return empty dimensions to prevent further crashes
				dims = layout.Dimensions{}
			}
		}()
		dims = ed.Layout(gtx)
	}()

	return dims
}

func (a *GioApp) layoutCredits(gtx layout.Context) layout.Dimensions {
	// Handle link clicks
	for a.ivritLink.Clicked(gtx) {
		go openURL("https://ivrit.ai")
	}
	for a.patreonLink.Clicked(gtx) {
		go openURL("https://www.patreon.com/ivrit_ai")
	}
	for a.creditsLink.Clicked(gtx) {
		go openURL("https://www.ivrit.ai/he/%d7%9e%d7%99-%d7%90%d7%a0%d7%97%d7%a0%d7%95/")
	}

	return layout.Flex{
		Axis:      layout.Horizontal,
		Spacing:   layout.SpaceStart,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := material.Label(a.theme, unit.Sp(10), "Powered by ")
			label.Color = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
			return label.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Clickable(gtx, a.ivritLink, func(gtx layout.Context) layout.Dimensions {
				label := material.Label(a.theme, unit.Sp(10), "ivrit.ai")
				label.Color = color.NRGBA{R: 0, G: 122, B: 255, A: 255} // Blue link color
				return label.Layout(gtx)
			})
			return btn
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := material.Label(a.theme, unit.Sp(10), " | ")
			label.Color = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
			return label.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Clickable(gtx, a.patreonLink, func(gtx layout.Context) layout.Dimensions {
				label := material.Label(a.theme, unit.Sp(10), "Support on Patreon")
				label.Color = color.NRGBA{R: 0, G: 122, B: 255, A: 255}
				return label.Layout(gtx)
			})
			return btn
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := material.Label(a.theme, unit.Sp(10), " | ")
			label.Color = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
			return label.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Clickable(gtx, a.creditsLink, func(gtx layout.Context) layout.Dimensions {
				label := material.Label(a.theme, unit.Sp(10), "Credits")
				label.Color = color.NRGBA{R: 0, G: 122, B: 255, A: 255}
				return label.Layout(gtx)
			})
			return btn
		}),
	)
}

// openURL opens a URL in the default browser
func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return
	}
	cmd.Run()
}

// selectFile opens native file dialog using cross-platform dialog library
func (a *GioApp) selectFile() {
	// Open file dialog with audio/video file filters
	filePath, err := dialog.File().
		Title("Choose audio or video file").
		Filter("Audio/Video Files", "mp3", "wav", "m4a", "aac", "flac", "ogg", "wma", "mp4", "avi", "mov", "mkv", "webm", "flv", "wmv", "m4v", "3gp").
		Filter("All Files", "*").
		Load()

	if err != nil {
		// User cancelled or error occurred
		if err.Error() != "Cancelled" {
			a.uiMutex.Lock()
			a.statusText = fmt.Sprintf("Error opening file dialog: %v", err)
			a.uiMutex.Unlock()
		}
		return
	}

	a.audioFilePath = filePath
	a.uiMutex.Lock()
	a.statusText = "File selected: " + filepath.Base(filePath)
	a.uiMutex.Unlock()

	// Get audio duration in background
	go func() {
		duration, err := getAudioDuration(filePath)
		if err == nil {
			a.audioDuration = duration
		}
	}()
}

// startTranscription starts transcription
func (a *GioApp) startTranscription() {
	if a.audioFilePath == "" {
		return
	}

	a.workerMutex.Lock()
	a.workerRunning = true
	a.stopRequested = false // Reset stop flag
	a.workerMutex.Unlock()

	a.uiMutex.Lock()
	a.statusText = "Transcribing..."
	a.transcriptionStartTime = time.Now().Unix()
	a.transcriptionSegments = nil // Clear previous transcription
	a.originalSegments = nil      // Clear previous original segments
	a.uiMutex.Unlock()

	go a.runTranscription()
}

// stopTranscription stops transcription
func (a *GioApp) stopTranscription() {
	a.workerMutex.Lock()
	a.stopRequested = true // Request stop
	a.workerMutex.Unlock()

	a.uiMutex.Lock()
	a.statusText = "Stopping..."
	a.uiMutex.Unlock()
}

// saveTranscription saves transcription using cross-platform dialog library
func (a *GioApp) saveTranscription() {
	if len(a.transcriptionSegments) == 0 {
		a.uiMutex.Lock()
		a.statusText = "No transcription to save"
		a.uiMutex.Unlock()
		return
	}

	format := a.formatList.Value
	ext := format
	filterName := "Text Files"
	if format == "json" {
		ext = "json"
		filterName = "JSON Files"
	} else if format == "srt" {
		ext = "srt"
		filterName = "SRT Subtitle Files"
	} else if format == "vtt" {
		ext = "vtt"
		filterName = "WebVTT Subtitle Files"
	} else {
		ext = "txt"
		filterName = "Text Files"
	}

	defaultName := "transcription." + ext

	// Open save dialog with appropriate file filter
	filePath, err := dialog.File().
		Title("Save transcription").
		Filter(filterName, ext).
		Filter("All Files", "*").
		SetStartFile(defaultName).
		Save()

	if err != nil {
		// User cancelled or error occurred
		if err.Error() != "Cancelled" {
			a.uiMutex.Lock()
			a.statusText = fmt.Sprintf("Error opening save dialog: %v", err)
			a.uiMutex.Unlock()
		}
		return
	}

	// Ensure file has the correct extension
	if !strings.HasSuffix(filePath, "."+ext) {
		filePath = filePath + "." + ext
	}

	outputText := FormatOutput(a.transcriptionSegments, format, false)
	if err := os.WriteFile(filePath, []byte(outputText), 0644); err != nil {
		a.uiMutex.Lock()
		a.statusText = fmt.Sprintf("Error saving file: %v", err)
		a.uiMutex.Unlock()
		return
	}

	a.uiMutex.Lock()
	a.statusText = "Transcription saved to " + filepath.Base(filePath)
	a.uiMutex.Unlock()
}

// appendSegment appends a segment (Gio handles RTL automatically)
func (a *GioApp) appendSegment(seg Segment) {
	// Check if stop was requested
	a.workerMutex.Lock()
	stopped := a.stopRequested
	a.workerMutex.Unlock()
	if stopped {
		return
	}

	a.transcriptionSegments = append(a.transcriptionSegments, seg)

	a.uiMutex.Lock()
	defer a.uiMutex.Unlock()

	// Gio's text shaper automatically handles RTL for Hebrew text!
	currentText := a.outputEditor.Text()
	format := a.formatList.Value

	// Limit text buffer to prevent crashes with very long transcriptions
	// Keep only last 20KB of text (conservative limit for Hebrew RTL text)
	const maxTextSize = 20000
	if len(currentText) > maxTextSize {
		// Keep only the last portion
		currentText = currentText[len(currentText)-maxTextSize:]
		// Try to start from a newline
		if idx := strings.Index(currentText, "\n"); idx > 0 {
			currentText = currentText[idx+1:]
		}
		// Prepend truncation notice
		currentText = "[...earlier text truncated for display...]\n\n" + currentText
	}

	var newText string
	switch format {
	case "text":
		newText = currentText + seg.Text + "\n"
	case "json":
		newText = currentText + fmt.Sprintf(`{"start": %.2f, "end": %.2f, "text": "%s"}`+"\n", seg.Start, seg.End, seg.Text)
	case "srt":
		start := FormatTimestamp(seg.Start, false)
		end := FormatTimestamp(seg.End, false)
		segmentNum := len(a.transcriptionSegments)
		newText = currentText + fmt.Sprintf("%d\n%s --> %s\n%s\n\n", segmentNum, start, end, seg.Text)
	case "vtt":
		start := FormatTimestamp(seg.Start, true)
		end := FormatTimestamp(seg.End, true)
		newText = currentText + fmt.Sprintf("%s --> %s\n%s\n\n", start, end, seg.Text)
	}

	// Safe text update with error recovery
	defer func() {
		if r := recover(); r != nil {
			// If SetText panics, just skip this update
			fmt.Fprintf(os.Stderr, "Warning: Failed to update text display: %v\n", r)
		}
	}()

	a.outputEditor.SetText(newText)

	// Update timing
	if a.transcriptionStartTime > 0 {
		elapsed := time.Now().Unix() - a.transcriptionStartTime
		elapsedSeconds := float64(elapsed)
		a.timingText = fmt.Sprintf("Elapsed: %.1fs", elapsedSeconds)
		if a.audioDuration > 0 && elapsedSeconds > 0 {
			speed := elapsedSeconds / a.audioDuration
			a.timingText += fmt.Sprintf(" | Speed: %.2fx", speed)
		}
	}
}

// transcriptionComplete handles completion
func (a *GioApp) transcriptionComplete(segments []Segment) {
	a.uiMutex.Lock()
	defer a.uiMutex.Unlock()

	a.progressVisible = false
	a.statusText = "Transcription complete"
	a.transcriptionSegments = segments

	format := a.formatList.Value
	finalOutput := FormatOutput(segments, format, false)

	// Gio handles RTL automatically - no manual markers needed!
	a.outputEditor.SetText(finalOutput)

	// Calculate timing
	elapsed := time.Now().Unix() - a.transcriptionStartTime
	elapsedSeconds := float64(elapsed)
	a.timingText = fmt.Sprintf("Elapsed: %.1fs", elapsedSeconds)
	if a.audioDuration > 0 {
		speed := elapsedSeconds / a.audioDuration
		a.timingText += fmt.Sprintf(" | Speed: %.2fx", speed)
	}
}

// runTranscription runs the transcription (ported from Qt/Fyne version)
func (a *GioApp) runTranscription() {
	defer func() {
		a.workerMutex.Lock()
		a.workerRunning = false
		a.workerMutex.Unlock()
	}()
	
	// Get options
	modelID := a.modelList.Value
	enableTranslation := a.enableTranslation.Value
	targetLang := a.translateLangList.Value
	keepOriginal := a.keepOriginal.Value
	audioPath := a.audioFilePath

	// Use optimal CPU threads
	cpuThreads := GetOptimalCPUThreads()
	
	// Channels for communication
	progressChan := make(chan string, 10)
	segmentChan := make(chan Segment, 100)
	doneChan := make(chan []Segment, 1)
	errorChan := make(chan string, 1)
	
	// Progress callback with ETA calculation
	progressCallback := func(msg string) {
		// Extract percentage from message if present (e.g., "Transcribing... 45%")
		var enhancedMsg string
		if strings.Contains(msg, "%") {
			// Try to parse percentage
			var percent int
			if _, err := fmt.Sscanf(msg, "Transcribing... %d%%", &percent); err == nil && percent > 0 && percent <= 100 {
				// Calculate ETA
				elapsed := time.Since(time.Unix(a.transcriptionStartTime, 0))
				if percent > 0 {
					// Estimate total time = elapsed / (percent/100)
					estimatedTotal := time.Duration(float64(elapsed) / (float64(percent) / 100.0))
					remaining := estimatedTotal - elapsed

					// Format remaining time
					if remaining > 0 {
						mins := int(remaining.Minutes())
						secs := int(remaining.Seconds()) % 60
						if mins > 0 {
							enhancedMsg = fmt.Sprintf("Transcribing... %d%% (ETA: %dm %ds)", percent, mins, secs)
						} else {
							enhancedMsg = fmt.Sprintf("Transcribing... %d%% (ETA: %ds)", percent, secs)
						}
					} else {
						enhancedMsg = msg
					}
				} else {
					enhancedMsg = msg
				}
			} else {
				enhancedMsg = msg
			}
		} else {
			enhancedMsg = msg
		}

		select {
		case progressChan <- enhancedMsg:
		default:
		}
	}
	
	// Segment callback
	segmentCallback := func(seg Segment) {
		select {
		case segmentChan <- seg:
		default:
		}
	}
	
	// Handle UI updates
	go func() {
		for {
			select {
			case msg := <-progressChan:
				a.uiMutex.Lock()
				a.statusText = msg
				a.uiMutex.Unlock()
				a.window.Invalidate() // Force UI redraw
			case seg := <-segmentChan:
				a.appendSegment(seg)
			case errMsg := <-errorChan:
				a.uiMutex.Lock()
				a.statusText = "Error: " + errMsg
				currentText := a.outputEditor.Text()
				a.outputEditor.SetText(currentText + "\n[Error: " + errMsg + "]\n")
				a.uiMutex.Unlock()
				return
			case segments := <-doneChan:
				a.transcriptionComplete(segments)
				return
			}
		}
	}()
	
	// Transcribe using native whisper.cpp
	go func() {
		// Get model path
		modelPath, modelErr := GetModelPath(modelID, func(msg string, pct int) {
			select {
			case progressChan <- msg:
			default:
			}
		})
		
		if modelErr != nil {
			errorChan <- modelErr.Error()
			return
		}
		
		// Use native whisper.cpp
		engine, engineErr := NewWhisperCGOEngine(modelPath)
		if engineErr != nil {
			errorChan <- fmt.Sprintf("Failed to initialize whisper engine: %v", engineErr)
			return
		}
		defer engine.Close()
		
		// Step 1: Transcribe in Hebrew (no whisper translation)
		progressCallback("Transcribing in Hebrew...")
		segments, err := engine.Transcribe(audioPath, modelID, cpuThreads, progressCallback, segmentCallback)
		if err != nil {
			errorChan <- err.Error()
			return
		}
		a.originalSegments = segments

		// Check if stop was requested after transcription
		a.workerMutex.Lock()
		stopped := a.stopRequested
		a.workerMutex.Unlock()
		if stopped {
			a.uiMutex.Lock()
			a.statusText = "Stopped"
			a.uiMutex.Unlock()
			doneChan <- segments
			return
		}

		// Step 2: Translate using Mistral if requested
		if enableTranslation {
			progressCallback(fmt.Sprintf("Translating to %s using Mistral 8B...", targetLang))
			translator := NewMistralTranslator()

			translatedSegments, transErr := translator.TranslateSegments(segments, targetLang, progressCallback, nil)
			if transErr != nil {
				errorChan <- fmt.Sprintf("Translation failed: %v", transErr)
				return
			}

			// Check if stop was requested during translation
			a.workerMutex.Lock()
			stopped := a.stopRequested
			a.workerMutex.Unlock()
			if stopped {
				a.uiMutex.Lock()
				a.statusText = "Stopped"
				a.uiMutex.Unlock()
				doneChan <- segments
				return
			}

			// If keep original is disabled, only show translation
			if !keepOriginal {
				for i := range translatedSegments {
					translatedSegments[i].Text = translatedSegments[i].Translation
					translatedSegments[i].Original = ""
				}
			} else {
				// Keep both original and translation
				for i := range translatedSegments {
					translatedSegments[i].Text = translatedSegments[i].Translation
				}
			}

			segments = translatedSegments
		}

		doneChan <- segments
	}()
}


