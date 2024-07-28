package utils

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/atotto/clipboard"
	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

func CopyToClipboard(rendered string) error {
	err := clipboard.WriteAll(rendered)
	if err != nil {
		return fmt.Errorf("failed to copy to clipboard: %v", err)
	}
	return nil
}

func WriteToFile(outputPath string, rendered string) error {
	err := ioutil.WriteFile(outputPath, []byte(rendered), 0644)
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

func PrintColoredMessage(symbol string, message string, messageColor color.Attribute) {
	white := color.New(color.FgWhite, color.Bold).SprintFunc()
	coloredMessage := color.New(messageColor).SprintFunc()

	fmt.Printf("%s%s%s %s\n", white("["), white(symbol), white("]"), coloredMessage(message))
}
