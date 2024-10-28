// web/integration.go

package web

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/sammcj/ingest/filesystem"
)

type CrawlResult struct {
	TreeString string
	Files      []filesystem.FileInfo
}

func ProcessWebURL(urlStr string, options CrawlOptions, excludePatterns []string) (*CrawlResult, error) {
	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if !strings.HasPrefix(parsedURL.Scheme, "http") {
		return nil, fmt.Errorf("URL must start with http:// or https://")
	}

	// Initialize crawler
	crawler := NewCrawler(options)
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

	// Generate tree representation
	treeString := generateWebTree(pages)

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
