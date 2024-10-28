// web/integration.go

package web

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/sammcj/ingest/filesystem"
	"github.com/sammcj/ingest/pdf"
)

type CrawlResult struct {
	TreeString string
	Files      []filesystem.FileInfo
}

func ProcessWebURL(urlStr string, options CrawlOptions, excludePatterns []string) (*CrawlResult, error) {
	// Check if URL points to a PDF
	isPDF, err := pdf.IsPDF(urlStr)
	if err != nil {
		return nil, fmt.Errorf("error checking PDF: %w", err)
	}

	if isPDF {
		content, err := pdf.ConvertPDFToMarkdown(urlStr, true)
		if err != nil {
			return nil, fmt.Errorf("error converting PDF: %w", err)
		}

		return &CrawlResult{
			TreeString: fmt.Sprintf("PDF Document: %s", urlStr),
			Files: []filesystem.FileInfo{{
				Path:      urlStr,
				Extension: ".md",
				Code:      content,
			}},
		}, nil
	}

	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if !strings.HasPrefix(parsedURL.Scheme, "http") {
		return nil, fmt.Errorf("URL must start with http:// or https://")
	}

	// Initialize crawler with the start URL
	crawler := NewCrawler(options, urlStr)
	crawler.SetExcludePatterns(excludePatterns)

	// Perform crawl
	pages, err := crawler.Crawl(urlStr)
	if err != nil {
		return nil, fmt.Errorf("crawl failed: %w", err)
	}

	// Convert crawled pages to FileInfo format
	var files []filesystem.FileInfo
	for _, page := range pages {
		// Skip pages with no content or error status codes
		if page.StatusCode != 200 || page.Content == "" {
			continue
		}

		files = append(files, filesystem.FileInfo{
			Path:      page.URL,
			Extension: ".md",
			Code:      page.Content,
		})
	}

	// Generate tree representation, but only if we have more than one page
	var treeString string
	if len(files) > 1 {
		treeString = generateWebTree(pages)
	} else if len(files) == 1 {
		treeString = fmt.Sprintf("Web Page: %s", files[0].Path)
	}

	// If we're crawling a specific page, only return that page's content
	if parsedURL.Path != "/" && parsedURL.Path != "" {
		for _, file := range files {
			fileURL, err := url.Parse(file.Path)
			if err != nil {
				continue
			}
			// Find the exact matching path (ignoring trailing slashes)
			if strings.TrimSuffix(fileURL.Path, "/") == strings.TrimSuffix(parsedURL.Path, "/") {
				return &CrawlResult{
					TreeString: fmt.Sprintf("Web Page: %s", file.Path),
					Files:      []filesystem.FileInfo{file},
				}, nil
			}
		}
	}

	return &CrawlResult{
		TreeString: treeString,
		Files:      files,
	}, nil
}

func generateWebTree(pages []*WebPage) string {
	var builder strings.Builder
	builder.WriteString("Web Crawl Structure:\n")

	// Create a map of depth to pages
	depthMap := make(map[int][]*WebPage)
	for _, page := range pages {
		if page.StatusCode == 200 && page.Content != "" {
			depthMap[page.Depth] = append(depthMap[page.Depth], page)
		}
	}

	// Build the tree structure with indentation
	for depth := 0; depth <= len(depthMap); depth++ {
		if pages, ok := depthMap[depth]; ok {
			for _, page := range pages {
				indent := strings.Repeat("  ", depth)
				urlPath := getURLPath(page.URL)
				builder.WriteString(fmt.Sprintf("%s├── %s\n", indent, urlPath))
			}
		}
	}

	return builder.String()
}

func getURLPath(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}

	path := parsedURL.Path
	if path == "" || path == "/" {
		return parsedURL.Host
	}

	return filepath.Join(parsedURL.Host, path)
}
