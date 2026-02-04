package utils

import (
	"fmt"
	"strings"
	"time"
)

const (
	spinnerFrames = `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`
	barLength     = 30
)

var spinnerIdx = 0

// PrintProgress displays a formatted progress message with optional spinner
func PrintProgress(message string, withSpinner bool) {
	if withSpinner {
		frame := string(spinnerFrames[spinnerIdx%len(spinnerFrames)])
		spinnerIdx++
		fmt.Printf("\r%s %s ", frame, message)
	} else {
		fmt.Printf("\r%s", message)
	}
}

// PrintProgressBar displays a progress bar with percentage
func PrintProgressBar(message string, current, total int) {
	if total <= 0 {
		total = 1
	}
	percent := (current * 100) / total
	filled := (current * barLength) / total

	bar := "["
	for i := 0; i < barLength; i++ {
		if i < filled {
			bar += "="
		} else if i == filled {
			bar += ">"
		} else {
			bar += " "
		}
	}
	bar += "]"

	fmt.Printf("\r%s %s %3d%%", message, bar, percent)
}

// PrintProgressStep displays a step in a multi-step process
func PrintProgressStep(step, totalSteps int, message string) {
	fmt.Printf("\r[%d/%d] %s", step, totalSteps, message)
}

// ClearProgress clears the progress line
func ClearProgress() {
	fmt.Print("\r" + strings.Repeat(" ", 100) + "\r")
}

// PrintCompletionLine prints a completion message and clears progress
func PrintCompletionLine(message string) {
	ClearProgress()
	fmt.Printf("✓ %s\n", message)
}

// PrintErrorLine prints an error message
func PrintErrorLine(message string) {
	ClearProgress()
	fmt.Printf("✗ %s\n", message)
}

// SpinnerDelay returns a duration for smooth spinner animation
func SpinnerDelay() time.Duration {
	return 80 * time.Millisecond
}
