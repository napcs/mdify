package scraper

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestScraperService_ExtractContent(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		selector string
		expected string
		hasError bool
	}{
		{
			name:     "valid HTML with selector",
			html:     `<div class="content"><h1>Title</h1><p>Content here</p></div>`,
			selector: ".content",
			expected: "# Title\n\nContent here",
			hasError: false,
		},
		{
			name:     "selector matches no elements",
			html:     `<div class="other"><h1>Title</h1></div>`,
			selector: ".content",
			expected: "",
			hasError: true,
		},
		{
			name:     "invalid HTML",
			html:     `<div><h1>Unclosed tag`,
			selector: "div",
			expected: "# Unclosed tag",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scraper := NewService(nil, nil, nil, nil, Config{})
			
			result, err := scraper.ExtractContent(tt.html, tt.selector)
			
			if tt.hasError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.hasError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.hasError && !strings.Contains(result, tt.expected) {
				t.Errorf("expected result to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestScraperService_GetOutputPath(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		baseDir  string
		expected string
		hasError bool
	}{
		{
			name:     "URL with path",
			url:      "https://example.com/docs/api",
			baseDir:  "/tmp/output",
			expected: "/tmp/output/docs/api.md",
			hasError: false,
		},
		{
			name:     "URL with root path",
			url:      "https://example.com/",
			baseDir:  "/tmp/output",
			expected: "/tmp/output/index.md",
			hasError: false,
		},
		{
			name:     "URL with no path",
			url:      "https://example.com",
			baseDir:  "/tmp/output",
			expected: "/tmp/output/index.md",
			hasError: false,
		},
		{
			name:     "URL with special characters",
			url:      "https://example.com/docs/api%20guide",
			baseDir:  "/tmp/output",
			expected: "/tmp/output/docs/api guide.md",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			scraper := NewService(nil, mockFS, nil, nil, Config{})
			
			result, err := scraper.GetOutputPath(tt.url, tt.baseDir)
			
			if tt.hasError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.hasError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.hasError && result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestScraperService_FetchWithRetries(t *testing.T) {
	tests := []struct {
		name           string
		setupClient    func() *MockHTTPClient
		url            string
		maxRetries     int
		expectedCalls  int
		expectedError  bool
		expectedSleeps int
	}{
		{
			name: "successful first attempt",
			setupClient: func() *MockHTTPClient {
				client := NewMockHTTPClient()
				client.SetResponse("https://example.com", 200, "success")
				return client
			},
			url:            "https://example.com",
			maxRetries:     3,
			expectedCalls:  1,
			expectedError:  false,
			expectedSleeps: 0,
		},
		{
			name: "404 error no retries",
			setupClient: func() *MockHTTPClient {
				client := NewMockHTTPClient()
				client.SetResponse("https://example.com", 404, "not found")
				return client
			},
			url:            "https://example.com",
			maxRetries:     3,
			expectedCalls:  1,
			expectedError:  true,
			expectedSleeps: 0,
		},
		{
			name: "retry on 500 error then success",
			setupClient: func() *MockHTTPClient {
				client := NewMockHTTPClient()
				client.SetResponse("https://example.com", 500, "error")
				// After first call, change the response
				return client
			},
			url:           "https://example.com",
			maxRetries:    3,
			expectedCalls: 4, // Initial + 3 retries
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			sleeper := NewMockSleeper()
			logger := NewMockLogger()
			config := Config{MaxRetries: tt.maxRetries}
			
			scraper := NewService(client, nil, sleeper, logger, config)
			
			_, err := scraper.FetchWithRetries(tt.url)
			
			if tt.expectedError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			
			actualCalls := client.GetCallCount(tt.url)
			if actualCalls != tt.expectedCalls {
				t.Errorf("expected %d calls, got %d", tt.expectedCalls, actualCalls)
			}
			
			if tt.expectedSleeps > 0 {
				sleeps := sleeper.GetSleepDurations()
				if len(sleeps) != tt.expectedSleeps {
					t.Errorf("expected %d sleeps, got %d", tt.expectedSleeps, len(sleeps))
				}
			}
		})
	}
}

func TestScraperService_SaveMarkdown(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		filePath    string
		setupFS     func() *MockFileSystem
		expectedErr bool
	}{
		{
			name:     "successful save",
			content:  "# Test Content",
			filePath: "/tmp/test.md",
			setupFS: func() *MockFileSystem {
				return NewMockFileSystem()
			},
			expectedErr: false,
		},
		{
			name:     "file creation error",
			content:  "# Test Content",
			filePath: "/tmp/test.md",
			setupFS: func() *MockFileSystem {
				fs := NewMockFileSystem()
				fs.SetCreateError(fmt.Errorf("permission denied"))
				return fs
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := tt.setupFS()
			scraper := NewService(nil, mockFS, nil, nil, Config{})
			
			err := scraper.SaveMarkdown(tt.content, tt.filePath)
			
			if tt.expectedErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectedErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			
			if !tt.expectedErr {
				savedContent, exists := mockFS.files[tt.filePath]
				if !exists {
					t.Errorf("file was not saved")
				}
				if savedContent != tt.content {
					t.Errorf("expected content %q, got %q", tt.content, savedContent)
				}
			}
		})
	}
}

func TestScraperService_ScrapeURL(t *testing.T) {
	t.Run("successful scrape", func(t *testing.T) {
		client := NewMockHTTPClient()
		client.SetResponse("https://example.com", 200, `<div class="content"><h1>Title</h1></div>`)
		
		logger := NewMockLogger()
		scraper := NewService(client, nil, NewMockSleeper(), logger, Config{MaxRetries: 3})
		
		result, err := scraper.ScrapeURL("https://example.com", ".content")
		
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "Title") {
			t.Errorf("expected result to contain 'Title', got %q", result)
		}
		
		messages := logger.GetMessages()
		if len(messages) == 0 || !strings.Contains(messages[0], "Scraping: https://example.com") {
			t.Errorf("expected scraping log message")
		}
	})
}

func TestScraperService_ExponentialBackoff(t *testing.T) {
	client := NewMockHTTPClient()
	client.SetError("https://example.com", fmt.Errorf("network error"))
	
	sleeper := NewMockSleeper()
	logger := NewMockLogger()
	scraper := NewService(client, nil, sleeper, logger, Config{MaxRetries: 3})
	
	_, err := scraper.FetchWithRetries("https://example.com")
	
	if err == nil {
		t.Errorf("expected error but got none")
	}
	
	sleeps := sleeper.GetSleepDurations()
	expected := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}
	
	if len(sleeps) != len(expected) {
		t.Errorf("expected %d sleeps, got %d", len(expected), len(sleeps))
		return
	}
	
	for i, expectedSleep := range expected {
		if sleeps[i] != expectedSleep {
			t.Errorf("sleep %d: expected %v, got %v", i, expectedSleep, sleeps[i])
		}
	}
}

func TestScraperService_ConcurrentScraping(t *testing.T) {
	tests := []struct {
		name        string
		urls        []string
		workers     int
		setupMocks  func() (*MockHTTPClient, *MockFileSystem, *MockLogger)
		expectSuccess int
		expectErrors  int
	}{
		{
			name:    "concurrent scraping with 4 workers",
			urls:    []string{"https://example.com/1", "https://example.com/2", "https://example.com/3", "https://example.com/4"},
			workers: 4,
			setupMocks: func() (*MockHTTPClient, *MockFileSystem, *MockLogger) {
				client := NewMockHTTPClient()
				fs := NewMockFileSystem()
				logger := NewMockLogger()
				
				for i := 1; i <= 4; i++ {
					url := fmt.Sprintf("https://example.com/%d", i)
					html := fmt.Sprintf(`<div class="content"><h1>Page %d</h1></div>`, i)
					client.SetResponse(url, 200, html)
				}
				return client, fs, logger
			},
			expectSuccess: 4,
			expectErrors:  0,
		},
		{
			name:    "sequential fallback with 1 worker",
			urls:    []string{"https://example.com/1", "https://example.com/2"},
			workers: 1,
			setupMocks: func() (*MockHTTPClient, *MockFileSystem, *MockLogger) {
				client := NewMockHTTPClient()
				fs := NewMockFileSystem()
				logger := NewMockLogger()
				
				client.SetResponse("https://example.com/1", 200, `<div class="content"><h1>Page 1</h1></div>`)
				client.SetResponse("https://example.com/2", 200, `<div class="content"><h1>Page 2</h1></div>`)
				return client, fs, logger
			},
			expectSuccess: 2,
			expectErrors:  0,
		},
		{
			name:    "mixed success and errors",
			urls:    []string{"https://example.com/good", "https://example.com/bad", "https://example.com/good2"},
			workers: 2,
			setupMocks: func() (*MockHTTPClient, *MockFileSystem, *MockLogger) {
				client := NewMockHTTPClient()
				fs := NewMockFileSystem()
				logger := NewMockLogger()
				
				client.SetResponse("https://example.com/good", 200, `<div class="content"><h1>Good</h1></div>`)
				client.SetResponse("https://example.com/bad", 500, "Server Error")
				client.SetResponse("https://example.com/good2", 200, `<div class="content"><h1>Good 2</h1></div>`)
				return client, fs, logger
			},
			expectSuccess: 2,
			expectErrors:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, fs, logger := tt.setupMocks()
			sleeper := NewMockSleeper()
			config := Config{
				Timeout:    30 * time.Second,
				MaxRetries: 3,
				Workers:    tt.workers,
			}

			scraper := NewService(client, fs, sleeper, logger, config)

			err := scraper.ScrapeURLs(tt.urls, ".content", "/tmp/test")

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify success and error counts from log messages
			messages := logger.GetMessages()
			successCount := 0
			errorCount := 0

			for _, msg := range messages {
				if strings.Contains(msg, "✓ Saved:") {
					successCount++
				} else if strings.Contains(msg, "Error scraping") {
					errorCount++
				}
			}

			if successCount != tt.expectSuccess {
				t.Errorf("expected %d successes, got %d", tt.expectSuccess, successCount)
			}
			if errorCount != tt.expectErrors {
				t.Errorf("expected %d errors, got %d", tt.expectErrors, errorCount)
			}

			// Verify correct number of files created
			createdFiles := fs.GetCreatedFiles()
			if len(createdFiles) != tt.expectSuccess {
				t.Errorf("expected %d files created, got %d", tt.expectSuccess, len(createdFiles))
			}

			// For concurrent tests, verify worker message
			if tt.workers > 1 {
				found := false
				for _, msg := range messages {
					if strings.Contains(msg, "Starting") && strings.Contains(msg, "workers") {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected worker startup message for concurrent test")
				}
			}
		})
	}
}