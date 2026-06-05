package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "normal path",
			input:    "/var/log",
			expected: "/var/log",
			wantErr:  false,
		},
		{
			name:     "path with traversal",
			input:    "../../../etc/passwd",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
			wantErr:  false,
		},
		{
			name:     "path with backslashes",
			input:    "\\var\\log",
			expected: "/var/log",
			wantErr:  false,
		},
		{
			name:     "relative path",
			input:    "var/log",
			expected: "/var/log",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sanitizePath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("sanitizePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("sanitizePath() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsPathAllowed(t *testing.T) {
	allowedPaths := []string{"/var/log", "/opt/logs"}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "allowed path",
			path:     "/var/log/app",
			expected: true,
		},
		{
			name:     "another allowed path",
			path:     "/opt/logs/test",
			expected: true,
		},
		{
			name:     "not allowed path",
			path:     "/etc/passwd",
			expected: false,
		},
		{
			name:     "empty allowed list",
			path:     "/any/path",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var paths []string
			if tt.name != "empty allowed list" {
				paths = allowedPaths
			}
			got := isPathAllowed(tt.path, paths)
			if got != tt.expected {
				t.Errorf("isPathAllowed() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{1024 * 1024 * 1024 * 1024, "1.00 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.expected {
				t.Errorf("formatBytes() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPaginate(t *testing.T) {
	files := make([]FileInfo, 100)
	for i := 0; i < 100; i++ {
		files[i] = FileInfo{Name: fmt.Sprintf("file%d.log", i)}
	}

	tests := []struct {
		name           string
		page           int
		pageSize       int
		expectedLen    int
		expectedTotal  int
		expectedHasNext bool
		expectedHasPrev bool
	}{
		{
			name:           "first page",
			page:           1,
			pageSize:       10,
			expectedLen:    10,
			expectedTotal:  100,
			expectedHasNext: true,
			expectedHasPrev: false,
		},
		{
			name:           "last page",
			page:           10,
			pageSize:       10,
			expectedLen:    10,
			expectedTotal:  100,
			expectedHasNext: false,
			expectedHasPrev: true,
		},
		{
			name:           "page beyond total",
			page:           11,
			pageSize:       10,
			expectedLen:    10,
			expectedTotal:  100,
			expectedHasNext: false,
			expectedHasPrev: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, pagination := paginate(files, tt.page, tt.pageSize)
			if len(result) != tt.expectedLen {
				t.Errorf("paginate() len = %v, want %v", len(result), tt.expectedLen)
			}
			if pagination.Total != tt.expectedTotal {
				t.Errorf("paginate() total = %v, want %v", pagination.Total, tt.expectedTotal)
			}
			if pagination.HasNext != tt.expectedHasNext {
				t.Errorf("paginate() HasNext = %v, want %v", pagination.HasNext, tt.expectedHasNext)
			}
			if pagination.HasPrev != tt.expectedHasPrev {
				t.Errorf("paginate() HasPrev = %v, want %v", pagination.HasPrev, tt.expectedHasPrev)
			}
		})
	}
}

func TestHandleFileList(t *testing.T) {
	// Test path traversal attempt
	t.Run("path traversal", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/files/list?path=../../../etc", nil)
		rr := httptest.NewRecorder()

		handleFileList(rr, req)

		// Path with traversal should be rejected with bad request
		if rr.Code != http.StatusBadRequest {
			t.Errorf("handleFileList() status = %v, want %v", rr.Code, http.StatusBadRequest)
		}
	})

	// Test non-existent path (but valid format)
	t.Run("non-existent path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/files/list?path=/nonexistent/path12345", nil)
		rr := httptest.NewRecorder()

		handleFileList(rr, req)

		if rr.Code != http.StatusNotFound && rr.Code != http.StatusForbidden {
			t.Errorf("handleFileList() status = %v, want %v or %v", rr.Code, http.StatusNotFound, http.StatusForbidden)
		}
	})

	// Test method not allowed
	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/files/list", nil)
		rr := httptest.NewRecorder()

		handleFileList(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("handleFileList() status = %v, want %v", rr.Code, http.StatusMethodNotAllowed)
		}
	})

	// Test invalid page parameter
	t.Run("invalid page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/files/list?page=abc", nil)
		rr := httptest.NewRecorder()

		handleFileList(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("handleFileList() status = %v, want %v", rr.Code, http.StatusBadRequest)
		}
	})

	// Test invalid page_size parameter
	t.Run("invalid page_size", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/files/list?page_size=1000", nil)
		rr := httptest.NewRecorder()

		handleFileList(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("handleFileList() status = %v, want %v", rr.Code, http.StatusBadRequest)
		}
	})
}

func TestAPIKeyAuthMiddleware(t *testing.T) {
	validKeys := []string{"valid-key-123"}

	// Test with valid key
	t.Run("valid key", func(t *testing.T) {
		handler := APIKeyAuthMiddleware(validKeys, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-API-Key", "valid-key-123")
		rr := httptest.NewRecorder()

		handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("APIKeyAuthMiddleware() status = %v, want %v", rr.Code, http.StatusOK)
		}
	})

	// Test with invalid key
	t.Run("invalid key", func(t *testing.T) {
		handler := APIKeyAuthMiddleware(validKeys, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-API-Key", "invalid-key")
		rr := httptest.NewRecorder()

		handler(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("APIKeyAuthMiddleware() status = %v, want %v", rr.Code, http.StatusUnauthorized)
		}
	})

	// Test without key
	t.Run("no key", func(t *testing.T) {
		handler := APIKeyAuthMiddleware(validKeys, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		handler(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("APIKeyAuthMiddleware() status = %v, want %v", rr.Code, http.StatusUnauthorized)
		}
	})

	// Test with empty keys (no auth required)
	t.Run("no auth required", func(t *testing.T) {
		handler := APIKeyAuthMiddleware(nil, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("APIKeyAuthMiddleware() status = %v, want %v", rr.Code, http.StatusOK)
		}
	})
}
