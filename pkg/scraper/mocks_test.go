package scraper

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type MockHTTPClient struct {
	responses map[string]*http.Response
	errors    map[string]error
	callCount map[string]int
}

func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		responses: make(map[string]*http.Response),
		errors:    make(map[string]error),
		callCount: make(map[string]int),
	}
}

func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	m.callCount[url]++
	
	if err, exists := m.errors[url]; exists {
		return nil, err
	}
	
	if resp, exists := m.responses[url]; exists {
		return resp, nil
	}
	
	return nil, fmt.Errorf("no mock response configured for %s", url)
}

func (m *MockHTTPClient) SetResponse(url string, statusCode int, body string) {
	resp := &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
	m.responses[url] = resp
}

func (m *MockHTTPClient) SetError(url string, err error) {
	m.errors[url] = err
}

func (m *MockHTTPClient) GetCallCount(url string) int {
	return m.callCount[url]
}

type MockFileSystem struct {
	files         map[string]string
	directories   map[string]bool
	createError   error
	mkdirError    error
	readError     error
	statError     error
	createdFiles  []string
	createdDirs   []string
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files:       make(map[string]string),
		directories: make(map[string]bool),
	}
}

func (m *MockFileSystem) Create(name string) (io.WriteCloser, error) {
	if m.createError != nil {
		return nil, m.createError
	}
	
	m.createdFiles = append(m.createdFiles, name)
	return &MockFileWriter{fs: m, filename: name}, nil
}

func (m *MockFileSystem) MkdirAll(path string, perm int) error {
	if m.mkdirError != nil {
		return m.mkdirError
	}
	
	m.createdDirs = append(m.createdDirs, path)
	m.directories[path] = true
	return nil
}

func (m *MockFileSystem) SetFile(filename, content string) {
	m.files[filename] = content
}

func (m *MockFileSystem) SetCreateError(err error) {
	m.createError = err
}

func (m *MockFileSystem) SetMkdirError(err error) {
	m.mkdirError = err
}

func (m *MockFileSystem) GetCreatedFiles() []string {
	return m.createdFiles
}

func (m *MockFileSystem) GetCreatedDirs() []string {
	return m.createdDirs
}

type MockFileWriter struct {
	fs       *MockFileSystem
	filename string
	buffer   bytes.Buffer
	closed   bool
}

func (w *MockFileWriter) Write(p []byte) (n int, err error) {
	if w.closed {
		return 0, fmt.Errorf("write to closed file")
	}
	return w.buffer.Write(p)
}

func (w *MockFileWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	w.fs.files[w.filename] = w.buffer.String()
	return nil
}

type MockSleeper struct {
	sleepDurations []time.Duration
}

func NewMockSleeper() *MockSleeper {
	return &MockSleeper{}
}

func (m *MockSleeper) Sleep(duration time.Duration) {
	m.sleepDurations = append(m.sleepDurations, duration)
}

func (m *MockSleeper) GetSleepDurations() []time.Duration {
	return m.sleepDurations
}

type MockLogger struct {
	messages []string
}

func NewMockLogger() *MockLogger {
	return &MockLogger{}
}

func (m *MockLogger) Printf(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	m.messages = append(m.messages, message)
}

func (m *MockLogger) GetMessages() []string {
	return m.messages
}

func (m *MockLogger) GetLastMessage() string {
	if len(m.messages) == 0 {
		return ""
	}
	return m.messages[len(m.messages)-1]
}

func (m *MockLogger) Clear() {
	m.messages = nil
}