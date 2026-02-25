package server

import (
	"bytes"
	"fmt"
	"io"

	"mdify/internal/filesystem"
)

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

func (m *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	if m.readError != nil {
		return nil, m.readError
	}
	
	if content, exists := m.files[filename]; exists {
		return []byte(content), nil
	}
	
	return nil, fmt.Errorf("file not found: %s", filename)
}

func (m *MockFileSystem) Stat(name string) (filesystem.FileInfo, error) {
	if m.statError != nil {
		return nil, m.statError
	}
	
	if _, exists := m.files[name]; exists {
		return &MockFileInfo{exists: true}, nil
	}
	
	if _, exists := m.directories[name]; exists {
		return &MockFileInfo{exists: true}, nil
	}
	
	return &MockFileInfo{exists: false}, nil
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

func (m *MockFileSystem) SetReadError(err error) {
	m.readError = err
}

func (m *MockFileSystem) SetStatError(err error) {
	m.statError = err
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

type MockFileInfo struct {
	exists bool
}

func (fi *MockFileInfo) IsExist() bool {
	return fi.exists
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