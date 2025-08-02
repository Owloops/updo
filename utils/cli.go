package utils

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type CLI struct{}

var Log = CLI{}

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
		fmt.Printf("\r%s [░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░] %d/%d (0.0%%)", prefix, current, total)
		return
	}

	percent := float64(current) / float64(total) * 100
	barWidth := 40
	filledWidth := int(float64(barWidth) * float64(current) / float64(total))

	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)
	fmt.Printf("\r%s [%s] %d/%d (%.1f%%)", prefix, bar, current, total, percent)

	if current == total {
		fmt.Println()
	}
}

func (c CLI) ProgressWithStatus(current, total int, prefix, status string) {
	percent := float64(current) / float64(total) * 100
	barWidth := 30
	filledWidth := int(float64(barWidth) * float64(current) / float64(total))

	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)
	fmt.Printf("\r%s [%s] %d/%d (%.1f%%) - %s", prefix, bar, current, total, percent, status)

	if current == total {
		fmt.Println()
	}
}

func (c CLI) Spinner(message string, stopCh <-chan bool) {
	spinChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0

	for {
		select {
		case <-stopCh:
			fmt.Printf("\r%s\n", message)
			return
		default:
			fmt.Printf("\r%s %s", spinChars[i%len(spinChars)], message)
			i++
			time.Sleep(100 * time.Millisecond)
		}
	}
}
