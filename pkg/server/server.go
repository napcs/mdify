package server

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"mdify/internal/filesystem"
)

type FileSystem interface {
	ReadFile(filename string) ([]byte, error)
	Stat(name string) (filesystem.FileInfo, error)
}

type Logger interface {
	Printf(format string, v ...interface{})
}

type Service struct {
	fs     FileSystem
	logger Logger
}

type MarkdownHandler struct {
	baseDir string
	fs      FileSystem
	logger  Logger
}

// NewService creates a new server service
func NewService(fs FileSystem, logger Logger) *Service {
	return &Service{
		fs:     fs,
		logger: logger,
	}
}

// ServeMarkdownFiles starts an HTTP server to serve markdown files
func (s *Service) ServeMarkdownFiles(dir string, port int) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", dir, err)
	}

	if _, err := s.fs.Stat(absDir); err != nil {
		return fmt.Errorf("directory does not exist: %s", absDir)
	}

	handler := &MarkdownHandler{baseDir: absDir, fs: s.fs, logger: s.logger}

	s.logger.Printf("Starting server on port %d, serving files from %s", port, absDir)
	s.logger.Printf("Server running at http://localhost:%d", port)

	return http.ListenAndServe(fmt.Sprintf(":%d", port), handler)
}

// ServeHTTP handles individual HTTP requests
func (h *MarkdownHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requestPath := strings.TrimPrefix(r.URL.Path, "/")

	if requestPath == "" {
		requestPath = "index.md"
	}

	if !strings.HasSuffix(requestPath, ".md") {
		requestPath += ".md"
	}

	filePath := filepath.Join(h.baseDir, requestPath)

	if !strings.HasPrefix(filePath, h.baseDir) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	fileInfo, err := h.fs.Stat(filePath)
	if err != nil || !fileInfo.IsExist() {
		h.logger.Printf("File not found: %s", filePath)
		http.NotFound(w, r)
		return
	}

	content, err := h.fs.ReadFile(filePath)
	if err != nil {
		h.logger.Printf("Error reading file %s: %v", filePath, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	h.logger.Printf("Serving: %s", filePath)
	w.Write(content)
}
