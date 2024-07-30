package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"unsafe"

	"github.com/atotto/clipboard"
	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	"github.com/schollz/progressbar/v3"
)

type termSize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func CopyToClipboard(rendered string) error {
	err := clipboard.WriteAll(rendered)
	if err != nil {
		return fmt.Errorf("failed to copy to clipboard: %v", err)
	}
	return nil
}

func WriteToFile(outputPath string, rendered string) error {
	err := os.WriteFile(outputPath, []byte(rendered), 0644)
	if err != nil {
		return fmt.Errorf("failed to write to file: %v", err)
	}
	fmt.Printf("%s Prompt written to file: %s\n", color.GreenString("✓"), outputPath)
	return nil
}

func SetupSpinner(message string) *progressbar.ProgressBar {
	return progressbar.NewOptions(-1,
		progressbar.OptionSetDescription(message),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
}

func Label(path string) string {
	if path == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "."
		}
		return wd
	}
	return path
}

func PrintColouredMessage(symbol string, message string, messageColor color.Attribute) {
	white := color.New(color.FgWhite, color.Bold).SprintFunc()
	colouredMessage := color.New(messageColor).SprintFunc()

	fmt.Printf("%s%s%s %s\n", white("["), white(symbol), white("]"), colouredMessage(message))
}

func EnsureConfigDirectories() error {
	home, err := homedir.Dir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDirs := []struct {
		path string
		desc string
	}{
		{filepath.Join(home, ".config", "ingest", "patterns", "exclude"), "Add .glob files here containing glob matches to exclude additional patterns."},
		{filepath.Join(home, ".config", "ingest", "patterns", "templates"), "Add go templates with the extension .tmpl here for different output formats."},
	}

	for _, dir := range configDirs {
		if err := os.MkdirAll(dir.path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir.path, err)
		}

		readmePath := filepath.Join(dir.path, "README.md")
		if _, err := os.Stat(readmePath); os.IsNotExist(err) {
			content := fmt.Sprintf("# %s\n\n%s", filepath.Base(dir.path), dir.desc)
			if err := os.WriteFile(readmePath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to create README.md in %s: %w", dir.path, err)
			}
		}
	}

	return nil
}

func FormatNumber(n int) string {
	in := strconv.Itoa(n)
	out := make([]byte, len(in)+(len(in)-2+int(in[0]/'0'))/3)
	if in[0] == '-' {
		in, out[0] = in[1:], '-'
	}

	for i, j, k := len(in)-1, len(out)-1, 0; ; i, j = i-1, j-1 {
		out[j] = in[i]
		if i == 0 {
			return string(out)
		}
		if k++; k == 3 {
			j, k = j-1, 0
			out[j] = ','
		}
	}
}

func GetTerminalWidth() int {
	ws := &termSize{}
	retCode, _, _ := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 {
		return 100
	}
	return int(ws.Col)
}
