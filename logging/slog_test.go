package logging

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fulcrumproject/commons/config"
	"github.com/go-chi/chi/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *config.Log
		expectNil bool
	}{
		{
			name: "json format with info level",
			cfg: &config.Log{
				Format: "json",
				Level:  "info",
			},
			expectNil: false,
		},
		{
			name: "text format with error level",
			cfg: &config.Log{
				Format: "text",
				Level:  "error",
			},
			expectNil: false,
		},
		{
			name: "text format with warn level",
			cfg: &config.Log{
				Format: "text",
				Level:  "warn",
			},
			expectNil: false,
		},
		{
			name: "text format with silent level",
			cfg: &config.Log{
				Format: "text",
				Level:  "silent",
			},
			expectNil: false,
		},
		{
			name: "default format (text) with empty level",
			cfg: &config.Log{
				Format: "",
				Level:  "",
			},
			expectNil: false,
		},
		{
			name: "json format with empty level",
			cfg: &config.Log{
				Format: "json",
				Level:  "",
			},
			expectNil: false,
		},
		{
			name: "unknown format defaults to text",
			cfg: &config.Log{
				Format: "unknown",
				Level:  "info",
			},
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.cfg)

			if tt.expectNil {
				assert.Nil(t, logger)
			} else {
				require.NotNil(t, logger)
				assert.IsType(t, &slog.Logger{}, logger)
			}
		})
	}
}

func TestSlogFormatter_NewLogEntry(t *testing.T) {
	tests := []struct {
		name   string
		logger *slog.Logger
		req    *http.Request
	}{
		{
			name:   "GET request",
			logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			req:    httptest.NewRequest("GET", "/test", nil),
		},
		{
			name:   "POST request",
			logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			req:    httptest.NewRequest("POST", "/api/test", nil),
		},
		{
			name:   "PUT request with query params",
			logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			req:    httptest.NewRequest("PUT", "/api/test?id=123", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &SlogFormatter{Logger: tt.logger}
			entry := formatter.NewLogEntry(tt.req)

			require.NotNil(t, entry)
			assert.Implements(t, (*middleware.LogEntry)(nil), entry)

			// Verify it's the correct type
			slogEntry, ok := entry.(*SlogLogEntry)
			require.True(t, ok)
			assert.Equal(t, tt.logger, slogEntry.Logger)
			assert.Equal(t, tt.req, slogEntry.req)
		})
	}
}

func TestSlogLogEntry_Write(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		uri      string
		status   int
		bytes    int
		elapsed  time.Duration
		remoteIP string
	}{
		{
			name:     "successful GET request",
			method:   "GET",
			uri:      "/api/test",
			status:   200,
			bytes:    1024,
			elapsed:  100 * time.Millisecond,
			remoteIP: "127.0.0.1:8080",
		},
		{
			name:     "POST request with error",
			method:   "POST",
			uri:      "/api/users",
			status:   400,
			bytes:    256,
			elapsed:  50 * time.Millisecond,
			remoteIP: "192.168.1.1:9090",
		},
		{
			name:     "PUT request",
			method:   "PUT",
			uri:      "/api/users/123",
			status:   204,
			bytes:    0,
			elapsed:  200 * time.Millisecond,
			remoteIP: "10.0.0.1:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))

			req := httptest.NewRequest(tt.method, tt.uri, nil)
			req.RemoteAddr = tt.remoteIP

			entry := &SlogLogEntry{
				Logger: logger,
				req:    req,
			}

			// This should not panic
			assert.NotPanics(t, func() {
				entry.Write(tt.status, tt.bytes, nil, tt.elapsed, nil)
			})

			// Verify log was written
			logOutput := buf.String()
			assert.Contains(t, logOutput, "HTTP Request")
			assert.Contains(t, logOutput, tt.method)
			assert.Contains(t, logOutput, tt.uri)
		})
	}
}

func TestSlogLogEntry_Panic(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		uri      string
		panicVal interface{}
		stack    []byte
	}{
		{
			name:     "string panic",
			method:   "GET",
			uri:      "/api/test",
			panicVal: "something went wrong",
			stack:    []byte("stack trace here"),
		},
		{
			name:     "error panic",
			method:   "POST",
			uri:      "/api/users",
			panicVal: assert.AnError,
			stack:    []byte("goroutine 1 [running]:\nstack trace"),
		},
		{
			name:     "nil panic",
			method:   "PUT",
			uri:      "/api/users/123",
			panicVal: nil,
			stack:    []byte(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))

			req := httptest.NewRequest(tt.method, tt.uri, nil)

			entry := &SlogLogEntry{
				Logger: logger,
				req:    req,
			}

			// This should not panic
			assert.NotPanics(t, func() {
				entry.Panic(tt.panicVal, tt.stack)
			})

			// Verify panic log was written
			logOutput := buf.String()
			assert.Contains(t, logOutput, "HTTP Request Panic")
			assert.Contains(t, logOutput, tt.method)
			assert.Contains(t, logOutput, tt.uri)
		})
	}
}
