package sitemap

import (
	"fmt"
	"io"
	"net/http"
	"strings"
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