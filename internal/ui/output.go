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

// Info は情報メッセージを青/シアン色の記号付きで出力します。
func Info(format string, a ...interface{}) {
    msg := fmt.Sprintf(format, a...)
    cyan.Print("ℹ ")
    fmt.Println(msg)
}

// Success は成功メッセージを緑色のチェックマーク付きで出力します。
func Success(format string, a ...interface{}) {
    msg := fmt.Sprintf(format, a...)
    green.Print("✔ ")
    fmt.Println(msg)
}

// Warn は警告メッセージを黄色の三角記号付きで出力します。
func Warn(format string, a ...interface{}) {
    msg := fmt.Sprintf(format, a...)
    yellow.Print("▲ ")
    fmt.Println(msg)
}

// Error はエラーメッセージを赤色のバツ印付きで標準エラー出力に出力します。
func Error(format string, a ...interface{}) {
    msg := fmt.Sprintf(format, a...)
    red.Print("✖ ")
    fmt.Fprintln(os.Stderr, msg)
}

// Step は処理ステップのセクションヘッダーを出力します。
func Step(format string, a ...interface{}) {
    msg := fmt.Sprintf(format, a...)
    cyan.Print("==> ")
    fmt.Println(msg)
}

// Dim は薄い灰色で補足テキストを出力します。
func Dim(format string, a ...interface{}) {
    msg := fmt.Sprintf(format, a...)
    gray.Println(msg)
}
