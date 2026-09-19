package ui

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	green  = color.New(color.FgGreen, color.Bold)
	yellow = color.New(color.FgYellow, color.Bold)
	red    = color.New(color.FgRed, color.Bold)
	cyan   = color.New(color.FgCyan, color.Bold)
	gray   = color.New(color.FgHiBlack)
)

// Info prints an informational message with a blue/cyan bullet.
func Info(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	cyan.Print("ℹ ")
	fmt.Println(msg)
}

// Success prints a success message with a green checkmark.
func Success(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	green.Print("✔ ")
	fmt.Println(msg)
}

// Warn prints a warning message with a yellow exclamation mark.
func Warn(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	yellow.Print("▲ ")
	fmt.Println(msg)
}

// Error prints an error message with a red cross.
func Error(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	red.Print("✖ ")
	fmt.Fprintln(os.Stderr, msg)
}

// Step prints a section step.
func Step(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	cyan.Print("==> ")
	fmt.Println(msg)
}

// Dim prints dim/grayed text.
func Dim(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	gray.Println(msg)
}
