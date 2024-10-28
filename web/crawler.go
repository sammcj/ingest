// web/crawler.go

package web

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type CrawlOptions struct {
	MaxDepth       int
	AllowedDomains []string
	Timeout        int
	ConcurrentJobs int
}

type WebPage struct {
	URL         string
	Content     string
	Title       string
	Links       []string
	RawHTML     string
	Depth       int
	StatusCode  int
	ContentType string
}

type Crawler struct {
	visited         map[string]bool
	visitedLock     sync.Mutex
	options         CrawlOptions
	md              goldmark.Markdown
	excludePatterns []string // New field
}

func (c *Crawler) SetExcludePatterns(patterns []string) {
	c.excludePatterns = patterns
}

func (c *Crawler) shouldExclude(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	// Get the path relative to the domain
	path := parsedURL.Path
	if path == "" {
		path = "/"
	}

	// Check each exclude pattern
	for _, pattern := range c.excludePatterns {
		// Convert URL-style paths to glob patterns
		// Remove leading slash for matching
		cleanPath := strings.TrimPrefix(path, "/")

		// Match the pattern against the path
		if match, _ := doublestar.Match(pattern, cleanPath); match {
			return true
		}

		// Also try matching with a leading slash
		if match, _ := doublestar.Match(pattern, path); match {
			return true
		}
	}

	return false
}

func NewCrawler(options CrawlOptions) *Crawler {
	if options.ConcurrentJobs == 0 {
		options.ConcurrentJobs = 5
	}
	if options.Timeout == 0 {
		options.Timeout = 30
	}

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	return &Crawler{
		visited: make(map[string]bool),
		options: options,
		md:      md,
	}
}

func (c *Crawler) isAllowed(urlStr string) bool {
	if len(c.options.AllowedDomains) == 0 {
		return true
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	for _, domain := range c.options.AllowedDomains {
		if strings.Contains(parsedURL.Host, domain) {
			return true
		}
	}
	return false
}

func (c *Crawler) hasVisited(urlStr string) bool {
	c.visitedLock.Lock()
	defer c.visitedLock.Unlock()
	return c.visited[urlStr]
}

func (c *Crawler) markVisited(urlStr string) {
	c.visitedLock.Lock()
	defer c.visitedLock.Unlock()
	c.visited[urlStr] = true
}

func (c *Crawler) fetchPage(urlStr string, depth int) (*WebPage, error) {
	if depth > c.options.MaxDepth {
		return nil, nil
	}

	if c.hasVisited(urlStr) {
		return nil, nil
	}

	if !c.isAllowed(urlStr) {
		return nil, nil
	}

	if c.shouldExclude(urlStr) {
		return nil, nil
	}

	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	c.markVisited(urlStr)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	title := doc.Find("title").Text()
	links := c.extractLinks(doc, urlStr)

	// Convert HTML to Markdown
	var mdContent strings.Builder
	htmlContent, err := doc.Find("body").Html()
	if err != nil {
		return nil, err
	}
	if err := c.md.Convert([]byte(htmlContent), &mdContent); err != nil {
		return nil, err
	}

	return &WebPage{
		URL:         urlStr,
		Content:     mdContent.String(),
		Title:       title,
		Links:       links,
		RawHTML:     string(body),
		Depth:       depth,
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

func (c *Crawler) extractLinks(doc *goquery.Document, baseURL string) []string {
	var links []string
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			absoluteURL := c.resolveURL(baseURL, href)
			if absoluteURL != "" {
				links = append(links, absoluteURL)
			}
		}
	})
	return links
}

func (c *Crawler) resolveURL(base, ref string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		return ""
	}

	refURL, err := url.Parse(ref)
	if err != nil {
		return ""
	}

	resolvedURL := baseURL.ResolveReference(refURL)
	if !strings.HasPrefix(resolvedURL.Scheme, "http") {
		return ""
	}

	return resolvedURL.String()
}

func (c *Crawler) Crawl(startURL string) ([]*WebPage, error) {
	var pages []*WebPage
	var pagesLock sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, c.options.ConcurrentJobs)

	var crawlPage func(urlStr string, depth int)
	crawlPage = func(urlStr string, depth int) {
		defer wg.Done()
		semaphore <- struct{}{}        // Acquire
		defer func() { <-semaphore }() // Release

		page, err := c.fetchPage(urlStr, depth)
		if err != nil || page == nil {
			return
		}

		pagesLock.Lock()
		pages = append(pages, page)
		pagesLock.Unlock()

		if depth < c.options.MaxDepth {
			for _, link := range page.Links {
				if !c.hasVisited(link) && c.isAllowed(link) {
					wg.Add(1)
					go crawlPage(link, depth+1)
				}
			}
		}
	}

	wg.Add(1)
	go crawlPage(startURL, 0)
	wg.Wait()

	return pages, nil
}
