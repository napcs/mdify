package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMarkdownHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name               string
		requestPath        string
		setupFS            func() *MockFileSystem
		expectedStatus     int
		expectedContent    string
		expectedContentType string
	}{
		{
			name:        "serve existing file",
			requestPath: "/docs/test",
			setupFS: func() *MockFileSystem {
				fs := NewMockFileSystem()
				fs.SetFile("/base/docs/test.md", "# Test Content\n\nThis is a test.")
				return fs
			},
			expectedStatus:      200,
			expectedContent:     "# Test Content\n\nThis is a test.",
			expectedContentType: "text/markdown; charset=utf-8",
		},
		{
			name:        "serve index file",
			requestPath: "/",
			setupFS: func() *MockFileSystem {
				fs := NewMockFileSystem()
				fs.SetFile("/base/index.md", "# Welcome\n\nHome page.")
				return fs
			},
			expectedStatus:      200,
			expectedContent:     "# Welcome\n\nHome page.",
			expectedContentType: "text/markdown; charset=utf-8",
		},
		{
			name:        "file not found",
			requestPath: "/nonexistent",
			setupFS: func() *MockFileSystem {
				return NewMockFileSystem()
			},
			expectedStatus:  404,
			expectedContent: "404 page not found",
		},
		{
			name:        "read error",
			requestPath: "/error",
			setupFS: func() *MockFileSystem {
				fs := NewMockFileSystem()
				fs.SetFile("/base/error.md", "content")
				fs.SetReadError(fmt.Errorf("read permission denied"))
				return fs
			},
			expectedStatus:  500,
			expectedContent: "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := tt.setupFS()
			logger := NewMockLogger()
			
			handler := &MarkdownHandler{
				baseDir: "/base",
				fs:      mockFS,
				logger:  logger,
			}
			
			req := httptest.NewRequest("GET", tt.requestPath, nil)
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			
			if tt.expectedContentType != "" {
				contentType := w.Header().Get("Content-Type")
				if contentType != tt.expectedContentType {
					t.Errorf("expected content type %q, got %q", tt.expectedContentType, contentType)
				}
			}
			
			body := w.Body.String()
			if !strings.Contains(body, tt.expectedContent) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedContent, body)
			}
		})
	}
}

func TestMarkdownHandler_PathSecurity(t *testing.T) {
	tests := []struct {
		name           string
		requestPath    string
		expectedStatus int
	}{
		{
			name:           "path traversal attempt",
			requestPath:    "/../../../etc/passwd",
			expectedStatus: 400,
		},
		{
			name:           "valid nested path",
			requestPath:    "/docs/api/overview",
			expectedStatus: 404, // File doesn't exist in mock, but path is valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			logger := NewMockLogger()
			
			handler := &MarkdownHandler{
				baseDir: "/base",
				fs:      mockFS,
				logger:  logger,
			}
			
			req := httptest.NewRequest("GET", tt.requestPath, nil)
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestMarkdownHandler_HTTPMethods(t *testing.T) {
	mockFS := NewMockFileSystem()
	logger := NewMockLogger()
	
	handler := &MarkdownHandler{
		baseDir: "/base",
		fs:      mockFS,
		logger:  logger,
	}

	methods := []string{"POST", "PUT", "DELETE", "PATCH"}
	
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d for %s method, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

func TestServerService_ServeMarkdownFiles(t *testing.T) {
	t.Run("directory does not exist", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.SetStatError(fmt.Errorf("directory not found"))
		logger := NewMockLogger()
		
		server := NewService(mockFS, logger)
		
		err := server.ServeMarkdownFiles("/nonexistent", 8080)
		
		if err == nil {
			t.Errorf("expected error for non-existent directory")
		}
		if !strings.Contains(err.Error(), "directory does not exist") {
			t.Errorf("expected directory error, got: %v", err)
		}
	})
}

func TestMarkdownHandler_CacheHeaders(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.SetFile("/base/test.md", "# Test")
	logger := NewMockLogger()
	
	handler := &MarkdownHandler{
		baseDir: "/base",
		fs:      mockFS,
		logger:  logger,
	}
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "public, max-age=3600" {
		t.Errorf("expected Cache-Control header 'public, max-age=3600', got %q", cacheControl)
	}
}

func TestMarkdownHandler_PathNormalization(t *testing.T) {
	tests := []struct {
		name         string
		requestPath  string
		expectedFile string
	}{
		{
			name:         "path without .md extension",
			requestPath:  "/docs/api",
			expectedFile: "/base/docs/api.md",
		},
		{
			name:         "path with .md extension",
			requestPath:  "/docs/api.md",
			expectedFile: "/base/docs/api.md",
		},
		{
			name:         "root path",
			requestPath:  "/",
			expectedFile: "/base/index.md",
		},
		{
			name:         "root with slash",
			requestPath:  "/index",
			expectedFile: "/base/index.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			mockFS.SetFile(tt.expectedFile, "content")
			logger := NewMockLogger()
			
			handler := &MarkdownHandler{
				baseDir: "/base",
				fs:      mockFS,
				logger:  logger,
			}
			
			req := httptest.NewRequest("GET", tt.requestPath, nil)
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			if w.Code != 200 {
				t.Errorf("expected status 200, got %d", w.Code)
			}
			
			body := w.Body.String()
			if body != "content" {
				t.Errorf("expected body 'content', got %q", body)
			}
		})
	}
}