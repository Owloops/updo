package utils

import (
	"fmt"
	"os"
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
