// pdf/pdf.go

package pdf

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ConvertPDFToMarkdown converts a PDF file to markdown format

func ConvertPDFToMarkdown(path string, isURL bool) (string, error) {
	var reader io.ReadCloser
	var err error

	if isURL {
		reader, err = downloadPDF(path)
		if err != nil {
			return "", fmt.Errorf("failed to download PDF: %w", err)
		}
		defer reader.Close()

		tempFile, err := os.CreateTemp("", "ingest-*.pdf")
		if err != nil {
			return "", fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		if _, err := io.Copy(tempFile, reader); err != nil {
			return "", fmt.Errorf("failed to save PDF: %w", err)
		}

		path = tempFile.Name()
	}

	// Open and read the PDF
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("# PDF Content: %s\n\n", filepath.Base(path)))

	// Extract text from each page
	totalPages := r.NumPage()
	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		page := r.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			return "", fmt.Errorf("failed to extract text from page %d: %w", pageNum, err)
		}

		// Clean and process the text
		cleanedText := cleanText(text)
		if cleanedText != "" {
			buf.WriteString(fmt.Sprintf("## Page %d\n\n", pageNum))
			buf.WriteString(cleanedText)
			buf.WriteString("\n\n")
		}
	}

	result := buf.String()
	if strings.TrimSpace(result) == strings.TrimSpace(fmt.Sprintf("# PDF Content: %s\n\n", filepath.Base(path))) {
		return "", fmt.Errorf("no text content could be extracted from PDF")
	}

	return result, nil
}

// IsPDF checks if a file is a PDF based on its content type or extension
func IsPDF(path string) (bool, error) {
	// Check if it's a URL
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		resp, err := http.Head(path)
		if err != nil {
			return false, fmt.Errorf("failed to check URL for PDF: %w", err)
		}
		defer resp.Body.Close()
		return resp.Header.Get("Content-Type") == "application/pdf", nil
	}

	// Check local file
	file, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read first 512 bytes to determine file type
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("failed to read file header: %w", err)
	}

	// Check file signature
	contentType := http.DetectContentType(buffer[:n])
	if contentType == "application/pdf" {
		return true, nil
	}

	// Also check file extension
	return strings.ToLower(filepath.Ext(path)) == ".pdf", nil
}

func downloadPDF(url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to download PDF: status code %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/pdf" {
		resp.Body.Close()
		return nil, fmt.Errorf("URL does not point to a PDF file")
	}

	return resp.Body, nil
}

func cleanText(text string) string {
	if strings.Contains(text, "%PDF-") || strings.Contains(text, "endobj") {
		// This appears to be raw PDF data rather than extracted text
		return ""
	}

	// Remove control characters except newlines and tabs
	text = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, text)

	// Split into lines and clean each line
	lines := strings.Split(text, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and lines that look like PDF syntax
		if line == "" ||
			strings.HasPrefix(line, "%") ||
			strings.HasPrefix(line, "/") ||
			strings.Contains(line, "obj") ||
			strings.Contains(line, "endobj") ||
			strings.Contains(line, "stream") {
			continue
		}

		cleanLines = append(cleanLines, line)
	}

	return strings.Join(cleanLines, "\n\n")
}
