// pdf/pdf.go

package pdf

import (
	"bytes"
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

		// Create a temporary file to store the PDF
		tempFile, err := os.CreateTemp("", "ingest-*.pdf")
		if err != nil {
			return "", fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		// Copy the downloaded PDF to the temp file
		if _, err := io.Copy(tempFile, reader); err != nil {
			return "", fmt.Errorf("failed to save PDF: %w", err)
		}

		path = tempFile.Name()
	}

	// Open the PDF file
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# PDF Content: %s\n\n", filepath.Base(path)))

	// Read each page
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

		// Add page header and content
		buf.WriteString(fmt.Sprintf("## Page %d\n\n", pageNum))
		buf.WriteString(cleanText(text))
		buf.WriteString("\n\n")
	}

	return buf.String(), nil
}

// IsPDF checks if a file is a PDF based on its content type or extension
func IsPDF(path string) (bool, error) {
	// Check if it's a URL
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		resp, err := http.Head(path)
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()
		return resp.Header.Get("Content-Type") == "application/pdf", nil
	}

	// Check local file
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Read first 512 bytes to determine file type
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return false, err
	}

	// Check file signature
	contentType := http.DetectContentType(buffer)
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
	// Remove excessive whitespace
	text = strings.ReplaceAll(text, "\r", "")
	text = strings.TrimSpace(text)

	// Normalize line endings
	lines := strings.Split(text, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n\n")
}
