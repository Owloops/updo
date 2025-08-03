package utils

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	_defaultBarWidth = 40
	_statusBarWidth  = 30
	_spinnerInterval = 100 * time.Millisecond
)

var (
	_spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
)

type CLI struct{}

var _log = CLI{}

var Log = _log

func NewCLI() CLI {
	return CLI{}
}

func (c CLI) Error(msg string) {
	fmt.Fprintf(os.Stderr, "✗ %s\n", msg)
}

func (c CLI) Warn(msg string) {
	fmt.Printf("! %s\n", msg)
}

func (c CLI) Info(msg string) {
	fmt.Printf("• %s\n", msg)
}

func (c CLI) Success(msg string) {
	fmt.Printf("✓ %s\n", msg)
}

func (c CLI) Plain(msg string) {
	fmt.Println(msg)
}

func (c CLI) Region(region string) string {
	return fmt.Sprintf("[%s]", region)
}

func (c CLI) Progress(current, total int, prefix string) {
	if total == 0 {
		emptyBar := strings.Repeat("░", _defaultBarWidth)
		fmt.Printf("\r%s [%s] %d/%d (0.0%%)", prefix, emptyBar, current, total)
		return
	}

	percent := float64(current) / float64(total) * 100
	filledWidth := int(float64(_defaultBarWidth) * float64(current) / float64(total))

	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", _defaultBarWidth-filledWidth)
	fmt.Printf("\r%s [%s] %d/%d (%.1f%%)", prefix, bar, current, total, percent)

	if current == total {
		fmt.Println()
	}
}

func (c CLI) ProgressWithStatus(current, total int, prefix, status string) {
	percent := float64(current) / float64(total) * 100
	filledWidth := int(float64(_statusBarWidth) * float64(current) / float64(total))

	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", _statusBarWidth-filledWidth)
	fmt.Printf("\r%s [%s] %d/%d (%.1f%%) - %s", prefix, bar, current, total, percent, status)

	if current == total {
		fmt.Println()
	}
}

func (c CLI) Spinner(message string, stopCh <-chan bool) {
	i := 0

	for {
		select {
		case <-stopCh:
			fmt.Printf("\r%s\n", message)
			return
		default:
			fmt.Printf("\r%s %s", _spinnerChars[i%len(_spinnerChars)], message)
			i++
			time.Sleep(_spinnerInterval)
		}
	}
}
