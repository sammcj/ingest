// web/crawler.go

package web

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/JohannesKaufmann/html-to-markdown/plugin"
	"github.com/PuerkitoBio/goquery"
	"github.com/bmatcuk/doublestar/v4"
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
	Depth       int
	StatusCode  int
	ContentType string
}

type Crawler struct {
	visited         map[string]bool
	visitedLock     sync.Mutex
	options         CrawlOptions
	converter       *md.Converter
	excludePatterns []string
}

func NewCrawler(options CrawlOptions) *Crawler {
	// Create a new converter with GitHub Flavoured Markdown support
	converter := md.NewConverter("", true, &md.Options{
		// Configure the converter to handle common edge cases
		StrongDelimiter: "**",
		EmDelimiter:     "*",
		LinkStyle:       "inlined",
		HeadingStyle:    "atx",
		HorizontalRule: "---",
		CodeBlockStyle: "fenced",
	})

	// Use GitHub Flavored Markdown plugins
	converter.Use(plugin.GitHubFlavored())

	// Configure the converter to handle specific elements
	converter.Keep("math", "script[type='math/tex']") // Keep math formulas
	converter.Remove("script", "style", "iframe", "noscript") // Remove unwanted elements

	return &Crawler{
		visited:   make(map[string]bool),
		options:   options,
		converter: converter,
	}
}

func (c *Crawler) SetExcludePatterns(patterns []string) {
	c.excludePatterns = patterns
}

func (c *Crawler) shouldExclude(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	path := parsedURL.Path
	if path == "" {
		path = "/"
	}

	for _, pattern := range c.excludePatterns {
		cleanPath := strings.TrimPrefix(path, "/")
		if match, _ := doublestar.Match(pattern, cleanPath); match {
			return true
		}
		if match, _ := doublestar.Match(pattern, path); match {
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

	client := &http.Client{
		Timeout: time.Duration(c.options.Timeout) * time.Second,
	}

	resp, err := client.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	c.markVisited(urlStr)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	// Extract title and links
	title := doc.Find("title").Text()
	links := c.extractLinks(doc, urlStr)

	// Convert HTML to Markdown
	markdown, err := c.converter.ConvertString(string(body))
	if err != nil {
		return nil, err
	}

	return &WebPage{
		URL:         urlStr,
		Content:     markdown,
		Title:       title,
		Links:       links,
		Depth:       depth,
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

func (c *Crawler) extractLinks(doc *goquery.Document, baseURL string) []string {
	var links []string
	baseURLParsed, err := url.Parse(baseURL)
	if err != nil {
		return links
	}

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			if absURL := c.resolveURL(baseURLParsed, href); absURL != "" {
				links = append(links, absURL)
			}
		}
	})

	return links
}

func (c *Crawler) resolveURL(base *url.URL, ref string) string {
	refURL, err := url.Parse(ref)
	if err != nil {
		return ""
	}

	resolvedURL := base.ResolveReference(refURL)
	if !strings.HasPrefix(resolvedURL.Scheme, "http") {
		return ""
	}

	return resolvedURL.String()
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

func (c *Crawler) Crawl(startURL string) ([]*WebPage, error) {
	var pages []*WebPage
	var pagesLock sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, c.options.ConcurrentJobs)

	var crawlPage func(urlStr string, depth int)
	crawlPage = func(urlStr string, depth int) {
		defer wg.Done()
		semaphore <- struct{}{} // Acquire
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
