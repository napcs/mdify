package sitemap

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/html/charset"
)

type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

type Logger interface {
	Printf(format string, v ...interface{})
}

// Sitemap represents a sitemap XML structure
type Sitemap struct {
	XMLName xml.Name `xml:"urlset"`
	URLs    []URL    `xml:"url"`
}

type URL struct {
	Loc string `xml:"loc"`
}

type Service struct {
	client HTTPClient
	logger Logger
}

// NewService creates a new sitemap service
func NewService(client HTTPClient, logger Logger) *Service {
	return &Service{
		client: client,
		logger: logger,
	}
}

// FetchSitemap fetches and parses a sitemap from the given URL
func (s *Service) FetchSitemap(sitemapURL string) (*Sitemap, error) {
	s.logger.Printf("Fetching sitemap: %s", sitemapURL)

	resp, err := s.client.Get(sitemapURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sitemap: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("sitemap request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read sitemap response: %w", err)
	}

	var sitemap Sitemap
	decoder := xml.NewDecoder(strings.NewReader(string(body)))
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(&sitemap); err != nil {
		// If it's not a valid sitemap format (RSS, HTML), return empty sitemap instead of error
		if strings.Contains(err.Error(), "expected element type") {
			s.logger.Printf("Warning: Document is not a sitemap format, found 0 URLs")
			return &Sitemap{URLs: []URL{}}, nil
		}
		return nil, fmt.Errorf("failed to parse sitemap XML: %w", err)
	}

	s.logger.Printf("Found %d URLs in sitemap", len(sitemap.URLs))
	return &sitemap, nil
}

// FilterURLs filters URLs by a path filter
func (s *Service) FilterURLs(sitemap *Sitemap, pathFilter string) []string {
	var filteredURLs []string

	for _, url := range sitemap.URLs {
		if pathFilter == "" || strings.Contains(url.Loc, pathFilter) {
			filteredURLs = append(filteredURLs, url.Loc)
		}
	}

	if pathFilter != "" {
		s.logger.Printf("Filtered to %d URLs matching path filter '%s'", len(filteredURLs), pathFilter)
	}

	return filteredURLs
}

// GetURLsFromSitemap fetches a sitemap and returns filtered URLs
func (s *Service) GetURLsFromSitemap(sitemapURL, pathFilter string) ([]string, error) {
	sitemap, err := s.FetchSitemap(sitemapURL)
	if err != nil {
		return nil, err
	}

	return s.FilterURLs(sitemap, pathFilter), nil
}
