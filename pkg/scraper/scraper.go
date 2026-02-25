package scraper

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
)

// HTTPClient interface for making HTTP requests
type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

// FileSystem interface for file operations
type FileSystem interface {
	Create(name string) (io.WriteCloser, error)
	MkdirAll(path string, perm int) error
}

// Sleeper interface for time delays
type Sleeper interface {
	Sleep(duration time.Duration)
}

// Logger interface for logging
type Logger interface {
	Printf(format string, v ...interface{})
}

// Config holds configuration for the scraper
type Config struct {
	Timeout    time.Duration
	MaxRetries int
	Workers    int
}

// Service provides web scraping functionality
type Service struct {
	client    HTTPClient
	converter *md.Converter
	fs        FileSystem
	sleeper   Sleeper
	logger    Logger
	config    Config
}

// Job represents a scraping job
type Job struct {
	URL      string
	Selector string
	Output   string
}

// Result represents the result of a scraping job
type Result struct {
	URL         string
	Success     bool
	Error       error
	OutputPath  string
}

// NewService creates a new scraper service
func NewService(client HTTPClient, fs FileSystem, sleeper Sleeper, logger Logger, config Config) *Service {
	converter := md.NewConverter("", true, nil)
	
	return &Service{
		client:    client,
		converter: converter,
		fs:        fs,
		sleeper:   sleeper,
		logger:    logger,
		config:    config,
	}
}

// FetchWithRetries fetches a URL with retry logic and exponential backoff
func (s *Service) FetchWithRetries(url string) (*http.Response, error) {
	var lastErr error
	
	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoffDuration := time.Duration(1<<uint(attempt-1)) * time.Second
			s.logger.Printf("Retrying %s in %v (attempt %d/%d)", url, backoffDuration, attempt+1, s.config.MaxRetries+1)
			s.sleeper.Sleep(backoffDuration)
		}

		resp, err := s.client.Get(url)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		if resp.StatusCode == 404 {
			resp.Body.Close()
			return nil, fmt.Errorf("404 not found: %s", url)
		}

		resp.Body.Close()
		lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}

	return nil, fmt.Errorf("failed after %d retries: %w", s.config.MaxRetries+1, lastErr)
}

// ExtractContent extracts content from HTML using a CSS selector and converts to markdown
func (s *Service) ExtractContent(htmlContent, selector string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	selection := doc.Find(selector)
	if selection.Length() == 0 {
		return "", fmt.Errorf("selector '%s' matched no elements", selector)
	}

	html, err := selection.Html()
	if err != nil {
		return "", fmt.Errorf("failed to extract HTML: %w", err)
	}

	markdown, err := s.converter.ConvertString(html)
	if err != nil {
		return "", fmt.Errorf("failed to convert to markdown: %w", err)
	}

	return markdown, nil
}

// ScrapeURL scrapes a single URL and returns the markdown content
func (s *Service) ScrapeURL(rawURL, selector string) (string, error) {
	s.logger.Printf("Scraping: %s", rawURL)

	resp, err := s.FetchWithRetries(rawURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	htmlBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return s.ExtractContent(string(htmlBytes), selector)
}

// GetOutputPath determines the output file path for a URL
func (s *Service) GetOutputPath(rawURL, baseDir string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL %s: %w", rawURL, err)
	}

	urlPath := strings.TrimPrefix(parsedURL.Path, "/")
	if urlPath == "" {
		urlPath = "index"
	}

	if !strings.HasSuffix(urlPath, ".md") {
		urlPath += ".md"
	}

	outputPath := filepath.Join(baseDir, urlPath)
	
	dir := filepath.Dir(outputPath)
	if err := s.fs.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	return outputPath, nil
}

// SaveMarkdown saves markdown content to a file
func (s *Service) SaveMarkdown(content, filePath string) error {
	file, err := s.fs.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer file.Close()

	_, err = io.WriteString(file, content)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filePath, err)
	}

	return nil
}

// ScrapeURLs scrapes multiple URLs either sequentially or concurrently
func (s *Service) ScrapeURLs(urls []string, selector, output string) error {
	if s.config.Workers <= 1 {
		return s.scrapeSequential(urls, selector, output)
	}
	return s.scrapeConcurrent(urls, selector, output)
}

func (s *Service) scrapeSequential(urls []string, selector, output string) error {
	successCount := 0
	errorCount := 0

	for _, rawURL := range urls {
		markdown, err := s.ScrapeURL(rawURL, selector)
		if err != nil {
			s.logger.Printf("Error scraping %s: %v", rawURL, err)
			errorCount++
			continue
		}

		outputPath, err := s.GetOutputPath(rawURL, output)
		if err != nil {
			s.logger.Printf("Error determining output path for %s: %v", rawURL, err)
			errorCount++
			continue
		}

		if err := s.SaveMarkdown(markdown, outputPath); err != nil {
			s.logger.Printf("Error saving %s: %v", outputPath, err)
			errorCount++
			continue
		}

		s.logger.Printf("✓ Saved: %s", outputPath)
		successCount++
	}

	s.logger.Printf("Completed: %d successful, %d errors", successCount, errorCount)
	return nil
}

func (s *Service) scrapeConcurrent(urls []string, selector, output string) error {
	numWorkers := s.config.Workers
	if numWorkers > len(urls) {
		numWorkers = len(urls)
	}

	s.logger.Printf("Starting %d workers to process %d URLs", numWorkers, len(urls))

	// Create channels
	jobs := make(chan Job, len(urls))
	results := make(chan Result, len(urls))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go s.worker(i, jobs, results, &wg)
	}

	// Send jobs
	for _, rawURL := range urls {
		jobs <- Job{
			URL:      rawURL,
			Selector: selector,
			Output:   output,
		}
	}
	close(jobs)

	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	successCount := 0
	errorCount := 0
	for result := range results {
		if result.Success {
			s.logger.Printf("✓ Saved: %s", result.OutputPath)
			successCount++
		} else {
			s.logger.Printf("Error scraping %s: %v", result.URL, result.Error)
			errorCount++
		}
	}

	s.logger.Printf("Completed: %d successful, %d errors", successCount, errorCount)
	return nil
}

func (s *Service) worker(id int, jobs <-chan Job, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		result := Result{
			URL: job.URL,
		}

		markdown, err := s.ScrapeURL(job.URL, job.Selector)
		if err != nil {
			result.Success = false
			result.Error = err
			results <- result
			continue
		}

		outputPath, err := s.GetOutputPath(job.URL, job.Output)
		if err != nil {
			result.Success = false
			result.Error = fmt.Errorf("error determining output path: %w", err)
			results <- result
			continue
		}

		if err := s.SaveMarkdown(markdown, outputPath); err != nil {
			result.Success = false
			result.Error = fmt.Errorf("error saving file: %w", err)
			results <- result
			continue
		}

		result.Success = true
		result.OutputPath = outputPath
		results <- result
	}
}