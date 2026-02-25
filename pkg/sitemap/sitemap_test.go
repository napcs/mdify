package sitemap

import (
	"fmt"
	"strings"
	"testing"
)

func TestSitemapService_FetchSitemap(t *testing.T) {
	tests := []struct {
		name          string
		setupClient   func() *MockHTTPClient
		sitemapURL    string
		expectedCount int
		expectedError bool
	}{
		{
			name: "valid sitemap",
			setupClient: func() *MockHTTPClient {
				client := NewMockHTTPClient()
				sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/</loc></url>
	<url><loc>https://example.com/docs/</loc></url>
	<url><loc>https://example.com/docs/api</loc></url>
	<url><loc>https://example.com/blog/</loc></url>
</urlset>`
				client.SetResponse("https://example.com/sitemap.xml", 200, sitemapXML)
				return client
			},
			sitemapURL:    "https://example.com/sitemap.xml",
			expectedCount: 4,
			expectedError: false,
		},
		{
			name: "empty sitemap",
			setupClient: func() *MockHTTPClient {
				client := NewMockHTTPClient()
				sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
</urlset>`
				client.SetResponse("https://example.com/sitemap.xml", 200, sitemapXML)
				return client
			},
			sitemapURL:    "https://example.com/sitemap.xml",
			expectedCount: 0,
			expectedError: false,
		},
		{
			name: "404 error",
			setupClient: func() *MockHTTPClient {
				client := NewMockHTTPClient()
				client.SetResponse("https://example.com/sitemap.xml", 404, "Not Found")
				return client
			},
			sitemapURL:    "https://example.com/sitemap.xml",
			expectedCount: 0,
			expectedError: true,
		},
		{
			name: "invalid XML",
			setupClient: func() *MockHTTPClient {
				client := NewMockHTTPClient()
				client.SetResponse("https://example.com/sitemap.xml", 200, "not valid xml")
				return client
			},
			sitemapURL:    "https://example.com/sitemap.xml",
			expectedCount: 0,
			expectedError: true,
		},
		{
			name: "network error",
			setupClient: func() *MockHTTPClient {
				client := NewMockHTTPClient()
				client.SetError("https://example.com/sitemap.xml", fmt.Errorf("network error"))
				return client
			},
			sitemapURL:    "https://example.com/sitemap.xml",
			expectedCount: 0,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			logger := NewMockLogger()
			service := NewService(client, logger)

			sitemap, err := service.FetchSitemap(tt.sitemapURL)

			if tt.expectedError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectedError {
				if len(sitemap.URLs) != tt.expectedCount {
					t.Errorf("expected %d URLs, got %d", tt.expectedCount, len(sitemap.URLs))
				}

				messages := logger.GetMessages()
				if len(messages) == 0 {
					t.Errorf("expected log messages")
				}
			}
		})
	}
}

func TestSitemapService_FilterURLs(t *testing.T) {
	sitemap := &Sitemap{
		URLs: []URL{
			{Loc: "https://example.com/"},
			{Loc: "https://example.com/docs/"},
			{Loc: "https://example.com/docs/api"},
			{Loc: "https://example.com/docs/guide"},
			{Loc: "https://example.com/blog/"},
			{Loc: "https://example.com/blog/post1"},
		},
	}

	tests := []struct {
		name          string
		pathFilter    string
		expectedCount int
		expectedURLs  []string
	}{
		{
			name:          "no filter",
			pathFilter:    "",
			expectedCount: 6,
			expectedURLs:  nil, // Don't check specific URLs for this case
		},
		{
			name:          "docs filter",
			pathFilter:    "/docs/",
			expectedCount: 3,
			expectedURLs: []string{
				"https://example.com/docs/",
				"https://example.com/docs/api",
				"https://example.com/docs/guide",
			},
		},
		{
			name:          "docs filter without trailing slash",
			pathFilter:    "/docs",
			expectedCount: 3,
			expectedURLs: []string{
				"https://example.com/docs/",
				"https://example.com/docs/api",
				"https://example.com/docs/guide",
			},
		},
		{
			name:          "blog filter",
			pathFilter:    "/blog",
			expectedCount: 2,
			expectedURLs: []string{
				"https://example.com/blog/",
				"https://example.com/blog/post1",
			},
		},
		{
			name:          "no matches",
			pathFilter:    "/nonexistent",
			expectedCount: 0,
			expectedURLs:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewMockLogger()
			service := NewService(nil, logger)

			result := service.FilterURLs(sitemap, tt.pathFilter)

			if len(result) != tt.expectedCount {
				t.Errorf("expected %d URLs, got %d", tt.expectedCount, len(result))
			}

			if tt.expectedURLs != nil {
				for _, expectedURL := range tt.expectedURLs {
					found := false
					for _, actualURL := range result {
						if actualURL == expectedURL {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected URL %s not found in results", expectedURL)
					}
				}
			}

			if tt.pathFilter != "" {
				messages := logger.GetMessages()
				if len(messages) == 0 {
					t.Errorf("expected filtering log message")
				}
				lastMessage := logger.GetLastMessage()
				if !strings.Contains(lastMessage, "Filtered to") {
					t.Errorf("expected filtering message, got: %s", lastMessage)
				}
			}
		})
	}
}

func TestSitemapService_GetURLsFromSitemap(t *testing.T) {
	t.Run("successful end-to-end", func(t *testing.T) {
		client := NewMockHTTPClient()
		sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/</loc></url>
	<url><loc>https://example.com/docs/</loc></url>
	<url><loc>https://example.com/docs/features</loc></url>
	<url><loc>https://example.com/blog/</loc></url>
</urlset>`
		client.SetResponse("https://example.com/sitemap.xml", 200, sitemapXML)

		logger := NewMockLogger()
		service := NewService(client, logger)

		urls, err := service.GetURLsFromSitemap("https://example.com/sitemap.xml", "/docs")

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		expectedCount := 2
		if len(urls) != expectedCount {
			t.Errorf("expected %d URLs, got %d", expectedCount, len(urls))
		}

		expectedURLs := []string{
			"https://example.com/docs/",
			"https://example.com/docs/features",
		}

		for _, expectedURL := range expectedURLs {
			found := false
			for _, actualURL := range urls {
				if actualURL == expectedURL {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected URL %s not found in results", expectedURL)
			}
		}
	})

	t.Run("fetch error propagates", func(t *testing.T) {
		client := NewMockHTTPClient()
		client.SetError("https://example.com/sitemap.xml", fmt.Errorf("network error"))

		logger := NewMockLogger()
		service := NewService(client, logger)

		urls, err := service.GetURLsFromSitemap("https://example.com/sitemap.xml", "")

		if err == nil {
			t.Errorf("expected error but got none")
		}
		if urls != nil {
			t.Errorf("expected nil URLs on error")
		}
	})
}

func TestSitemapXMLParsing(t *testing.T) {
	tests := []struct {
		name        string
		xml         string
		expectedErr bool
		expectedLen int
		description string
	}{
		{
			name: "standard UTF-8 sitemap",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/page1</loc></url>
	<url><loc>https://example.com/page2</loc></url>
</urlset>`,
			expectedErr: false,
			expectedLen: 2,
			description: "Standard UTF-8 encoding",
		},
		{
			name: "US-ASCII encoding sitemap",
			xml: `<?xml version="1.0" encoding="us-ascii"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/page1</loc></url>
	<url><loc>https://example.com/page2</loc></url>
</urlset>`,
			expectedErr: false,
			expectedLen: 2,
			description: "US-ASCII encoding that caused original failure",
		},
		{
			name: "ISO-8859-1 encoding sitemap",
			xml: `<?xml version="1.0" encoding="ISO-8859-1"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/page1</loc></url>
</urlset>`,
			expectedErr: false,
			expectedLen: 1,
			description: "ISO-8859-1 encoding",
		},
		{
			name: "sitemap with additional fields",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
		<loc>https://example.com/page1</loc>
		<lastmod>2023-01-01</lastmod>
		<priority>0.8</priority>
		<changefreq>daily</changefreq>
	</url>
</urlset>`,
			expectedErr: false,
			expectedLen: 1,
			description: "Sitemap with lastmod, priority, changefreq",
		},
		{
			name: "sitemap without namespace",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<urlset>
	<url><loc>https://example.com/page1</loc></url>
</urlset>`,
			expectedErr: false,
			expectedLen: 1,
			description: "Sitemap without xmlns namespace",
		},
		{
			name: "sitemap with different namespace",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xsi:schemaLocation="http://www.sitemaps.org/schemas/sitemap/0.9
        http://www.sitemaps.org/schemas/sitemap/0.9/sitemap.xsd">
	<url><loc>https://example.com/page1</loc></url>
</urlset>`,
			expectedErr: false,
			expectedLen: 1,
			description: "Sitemap with schema location",
		},
		{
			name:        "completely invalid XML",
			xml:         `<invalid xml`,
			expectedErr: true,
			expectedLen: 0,
			description: "Malformed XML",
		},
		{
			name:        "not XML at all",
			xml:         `this is not xml`,
			expectedErr: true,
			expectedLen: 0,
			description: "Plain text instead of XML",
		},
		{
			name: "XML but not sitemap format",
			xml: `<?xml version="1.0"?>
<rss version="2.0">
	<channel>
		<title>Not a sitemap</title>
	</channel>
</rss>`,
			expectedErr: false,
			expectedLen: 0,
			description: "Valid XML but RSS format, not sitemap",
		},
		{
			name: "empty urlset",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
</urlset>`,
			expectedErr: false,
			expectedLen: 0,
			description: "Empty but valid sitemap",
		},
		{
			name: "sitemap with CDATA sections",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc><![CDATA[https://example.com/page with spaces]]></loc></url>
</urlset>`,
			expectedErr: false,
			expectedLen: 1,
			description: "URLs with CDATA sections",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewMockHTTPClient()
			client.SetResponse("https://example.com/sitemap.xml", 200, tt.xml)

			logger := NewMockLogger()
			service := NewService(client, logger)

			sitemap, err := service.FetchSitemap("https://example.com/sitemap.xml")

			if tt.expectedErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectedErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectedErr {
				if sitemap == nil {
					t.Errorf("expected sitemap but got nil")
					return
				}
				if len(sitemap.URLs) != tt.expectedLen {
					t.Errorf("expected %d URLs, got %d", tt.expectedLen, len(sitemap.URLs))
				}
			}
		})
	}
}

func TestSitemapRealWorldScenarios(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectedErr  bool
		expectedLen  int
		description  string
	}{
		{
			name:       "large WordPress sitemap",
			statusCode: 200,
			responseBody: `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://blog.example.com/</loc><lastmod>2023-12-01T10:00:00+00:00</lastmod></url>
	<url><loc>https://blog.example.com/post-1/</loc><lastmod>2023-11-30T15:30:00+00:00</lastmod></url>
	<url><loc>https://blog.example.com/post-2/</loc><lastmod>2023-11-29T09:15:00+00:00</lastmod></url>
	<url><loc>https://blog.example.com/category/tech/</loc><lastmod>2023-12-01T12:00:00+00:00</lastmod></url>
	<url><loc>https://blog.example.com/about/</loc><lastmod>2023-10-15T14:20:00+00:00</lastmod></url>
</urlset>`,
			expectedErr: false,
			expectedLen: 5,
			description: "WordPress-style sitemap with timestamps",
		},
		{
			name:       "documentation site sitemap",
			statusCode: 200,
			responseBody: `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://docs.example.com/</loc></url>
	<url><loc>https://docs.example.com/getting-started/</loc></url>
	<url><loc>https://docs.example.com/api/</loc></url>
	<url><loc>https://docs.example.com/api/authentication/</loc></url>
	<url><loc>https://docs.example.com/api/endpoints/</loc></url>
	<url><loc>https://docs.example.com/guides/</loc></url>
	<url><loc>https://docs.example.com/guides/best-practices/</loc></url>
	<url><loc>https://docs.example.com/faq/</loc></url>
</urlset>`,
			expectedErr: false,
			expectedLen: 8,
			description: "Documentation site sitemap",
		},
		{
			name:       "sitemap index file",
			statusCode: 200,
			responseBody: `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap>
		<loc>https://example.com/sitemap-posts.xml</loc>
		<lastmod>2023-12-01T10:00:00+00:00</lastmod>
	</sitemap>
	<sitemap>
		<loc>https://example.com/sitemap-pages.xml</loc>
		<lastmod>2023-11-30T15:30:00+00:00</lastmod>
	</sitemap>
</sitemapindex>`,
			expectedErr: false,
			expectedLen: 0,
			description: "Sitemap index (should return 0 URLs as it's not a urlset)",
		},
		{
			name:         "gzipped sitemap content type",
			statusCode:   200,
			responseBody: `not actually gzipped but simulates the scenario`,
			expectedErr:  true,
			expectedLen:  0,
			description:  "Invalid content that might come from gzipped response",
		},
		{
			name:       "sitemap with encoded URLs",
			statusCode: 200,
			responseBody: `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/search?q=hello%20world&amp;type=docs</loc></url>
	<url><loc>https://example.com/user/john%40example.com</loc></url>
	<url><loc>https://example.com/path/with%2Fslashes</loc></url>
</urlset>`,
			expectedErr: false,
			expectedLen: 3,
			description: "URLs with encoded characters and XML entities",
		},
		{
			name:         "server returns 403",
			statusCode:   403,
			responseBody: `Forbidden`,
			expectedErr:  true,
			expectedLen:  0,
			description:  "Access denied to sitemap",
		},
		{
			name:         "server returns 500",
			statusCode:   500,
			responseBody: `Internal Server Error`,
			expectedErr:  true,
			expectedLen:  0,
			description:  "Server error when fetching sitemap",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewMockHTTPClient()
			client.SetResponse("https://example.com/sitemap.xml", tt.statusCode, tt.responseBody)

			logger := NewMockLogger()
			service := NewService(client, logger)

			sitemap, err := service.FetchSitemap("https://example.com/sitemap.xml")

			if tt.expectedErr && err == nil {
				t.Errorf("expected error but got none for %s", tt.description)
			}
			if !tt.expectedErr && err != nil {
				t.Errorf("unexpected error for %s: %v", tt.description, err)
			}

			if !tt.expectedErr {
				if sitemap == nil {
					t.Errorf("%s: expected sitemap but got nil", tt.description)
					return
				}
				if len(sitemap.URLs) != tt.expectedLen {
					t.Errorf("%s: expected %d URLs, got %d", tt.description, tt.expectedLen, len(sitemap.URLs))
				}
			}
		})
	}
}
