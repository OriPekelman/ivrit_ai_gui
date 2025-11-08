package main

import (
	"log"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/op"
)

func main() {
	// Check if running in CLI mode (any command-line arguments provided)
	if len(os.Args) > 1 {
		// Check if the first arg is a flag (starts with -)
		if os.Args[1][0] == '-' {
			CLIMode()
			return
		}
	}

	// Run GUI mode
	go func() {
		w := new(app.Window)
		w.Option(app.Title("ivrit.ai - Hebrew Audio Transcription"))
		w.Option(app.Size(900, 700))
		if err := run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(w *app.Window) error {
	var ops op.Ops
	gioApp := NewGioApp(w)

	// Ticker for UI refresh during transcription (avoids CGO thread safety issues)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		// Check for timer tick
		select {
		case <-ticker.C:
			// Periodic refresh while transcription is running
			w.Invalidate()
		default:
			// Continue to event handling
		}

		// Handle window events
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			gioApp.Layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}
