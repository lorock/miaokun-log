package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// FileListResponse represents the paginated file list response
type FileListResponse struct {
	Success    bool       `json:"success"`
	Data       []FileInfo `json:"data"`
	Pagination Pagination `json:"pagination"`
	Error      *APIError  `json:"error,omitempty"`
}

// FileInfo represents detailed file information
type FileInfo struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	FullPath     string    `json:"full_path"`
	Size         int64     `json:"size"`
	SizeReadable string    `json:"size_readable"`
	ModTime      time.Time `json:"mod_time"`
	ModTimeStr   string    `json:"mod_time_str"`
	FileType     string    `json:"file_type"`
	IsDir        bool      `json:"is_dir"`
	IsReadable   bool      `json:"is_readable"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// APIError represents structured error information
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}


// Sensitive directories that should be excluded from file browsing for security
// Note: /home is intentionally excluded to allow users to browse their own home directory
var sensitiveDirs = []string{
	"/etc",           // 系统配置目录
	"/root",          // root 用户目录（普通用户不可见）
	"/proc",          // 进程信息
	"/sys",           // 系统信息
	"/dev",           // 设备文件
	"/run",           // 运行时的 PID 文件
	"/var/run",       // 运行时 PID 文件
	"/var/lock",      // 锁文件
	"/var/tmp",       // 临时文件
	"/.git",          // Git 仓库
	"/.svn",          // SVN 仓库
	"/.hg",           // Mercurial 仓库
	"/.dockerenv",    // Docker 环境文件
}

// FileListRequest represents the request parameters for file listing
type FileListRequest struct {
	Path     string  `json:"path"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
	Since    float64 `json:"since"`
}

// handleFileList returns a paginated list of files with full path information
// GET /api/v1/files/list
//
// Query Parameters:
//   - path: Directory path to list (default: /var/log)
//   - page: Page number (default: 1)
//   - page_size: Items per page (default: 50, max: 500)
//   - since: Only show files modified within last N days (default: 30)
//
// Authentication: Required (Basic Auth or API Key)
//
// Response: JSON with file list and pagination info
func handleFileList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET method is allowed", "")
		return
	}

	// Parse query parameters
	req, err := parseFileListRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Failed to parse request parameters", err.Error())
		return
	}

	// Sanitize and validate path
	sanitizedPath, err := sanitizePath(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PATH", "Invalid path format", err.Error())
		return
	}

	if sanitizedPath == "" {
		writeError(w, http.StatusBadRequest, "INVALID_PATH", "Path cannot be empty", "")
		return
	}

	// Allow browsing all accessible directories (except root which is handled separately)
	// The allowedPaths check is only for search functionality, not file browsing
	// For file browsing, we allow any path the user has permission to access
	// Sensitive directories are filtered when listing directory contents

	// Check if path exists and is accessible
	info, err := os.Stat(sanitizedPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "PATH_NOT_FOUND", 
				"The requested path does not exist",
				sanitizedPath)
			return
		}
		if os.IsPermission(err) {
			writeError(w, http.StatusForbidden, "ACCESS_DENIED",
				"Permission denied accessing the requested path",
				sanitizedPath)
			return
		}
		writeError(w, http.StatusInternalServerError, "FILESYSTEM_ERROR",
			"Failed to access the filesystem",
			err.Error())
		return
	}

	// If path is a file, return single file info
	if !info.IsDir() {
		fileInfo := createFileInfo(sanitizedPath, info)
		writeSuccess(w, []FileInfo{fileInfo}, Pagination{
			Page:       req.Page,
			PageSize:   req.PageSize,
			Total:      1,
			TotalPages: 1,
			HasNext:    false,
			HasPrev:    false,
		})
		return
	}

	// Read directory contents
	entries, err := os.ReadDir(sanitizedPath)
	if err != nil {
		if os.IsPermission(err) {
			writeError(w, http.StatusForbidden, "ACCESS_DENIED",
				"Permission denied reading directory contents",
				sanitizedPath)
			return
		}
		writeError(w, http.StatusInternalServerError, "DIRECTORY_READ_ERROR",
			"Failed to read directory contents",
			err.Error())
		return
	}

	// Filter and collect file information
	// 注意：使用 make 初始化空切片，避免 JSON 序列化为 null
	allFiles := make([]FileInfo, 0)
	cutoff := time.Now().Add(-time.Duration(req.Since * 24 * float64(time.Hour)))

	for _, entry := range entries {
		entryPath := filepath.Join(sanitizedPath, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue // Skip entries we can't read
		}

		// Security: Skip sensitive directories
		if entry.IsDir() && isSensitiveDir(entry.Name()) {
			continue
		}

		// Apply time filter (only for files, not directories)
		if !entry.IsDir() && info.ModTime().Before(cutoff) {
			continue
		}

		fileInfo := createFileInfo(entryPath, info)
		allFiles = append(allFiles, fileInfo)
	}

	// Apply pagination
	paginatedFiles, pagination := paginate(allFiles, req.Page, req.PageSize)

	writeSuccess(w, paginatedFiles, pagination)
}

// parseFileListRequest parses and validates query parameters
func parseFileListRequest(r *http.Request) (*FileListRequest, error) {
	path := r.URL.Query().Get("path")
	if path == "" {
		// 默认使用根目录
		path = "/"
	}

	req := &FileListRequest{
		Path:     path,
		Page:     1,
		PageSize: 50,
		Since:    30,
	}

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return nil, fmt.Errorf("invalid page number: %s", pageStr)
		}
		req.Page = page
	}

	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 || pageSize > 500 {
			return nil, fmt.Errorf("invalid page_size: %s (must be 1-500)", pageSizeStr)
		}
		req.PageSize = pageSize
	}

	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		since, err := strconv.ParseFloat(sinceStr, 64)
		if err != nil || since < 0 {
			return nil, fmt.Errorf("invalid since value: %s", sinceStr)
		}
		req.Since = since
	}

	return req, nil
}

// createFileInfo creates a FileInfo from os.FileInfo
func createFileInfo(path string, info os.FileInfo) FileInfo {
	fileType := getFileType(path, info)
	
	// Check if the path is readable
	isReadable := true
	if info.IsDir() {
		// Try to open the directory to check read access
		dir, err := os.Open(path)
		if err != nil {
			isReadable = false
		} else {
			dir.Close()
		}
	} else {
		// Check if file is readable
		_, err := os.Stat(path)
		isReadable = err == nil
	}
	
	return FileInfo{
		Name:         info.Name(),
		Path:         filepath.Dir(path),
		FullPath:     path,
		Size:         info.Size(),
		SizeReadable: formatBytes(info.Size()),
		ModTime:      info.ModTime(),
		ModTimeStr:   info.ModTime().Format(time.RFC3339),
		FileType:     fileType,
		IsDir:        info.IsDir(),
		IsReadable:   isReadable,
	}
}

// getFileType determines the file type based on extension
func getFileType(path string, info os.FileInfo) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".log":
		return "log"
	case ".gz":
		return "gzip"
	case ".zip":
		return "zip"
	case ".tar":
		return "tar"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	case ".txt":
		return "text"
	case "":
		// Use the provided info parameter instead of calling os.Stat again
		if info != nil && info.IsDir() {
			return "directory"
		}
		return "unknown"
	default:
		return "unknown"
	}
}

// formatBytes converts bytes to human readable format
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// paginate slices the files array based on page and pageSize
func paginate(files []FileInfo, page, pageSize int) ([]FileInfo, Pagination) {
	total := len(files)
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	// Adjust page if out of bounds
	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	// 确保返回非 nil 切片，JSON 序列化为 [] 而不是 null
	result := make([]FileInfo, 0)
	if start < end {
		result = append(result, files[start:end]...)
	}

	return result, Pagination{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    end < total,
		HasPrev:    start > 0,
	}
}

// writeSuccess writes a successful JSON response
func writeSuccess(w http.ResponseWriter, files []FileInfo, pagination Pagination) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(FileListResponse{
		Success:    true,
		Data:       files,
		Pagination: pagination,
	})
}

// writeError writes an error JSON response
func writeError(w http.ResponseWriter, status int, code, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(FileListResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// isSensitiveDir checks if a directory name is in the sensitive directories list
// If the current user is root, /root directory will be allowed
func isSensitiveDir(dirName string) bool {
	// Special case: if running as root, allow /root directory
	if dirName == "root" && isRunningAsRoot() {
		return false
	}

	for _, sensitive := range sensitiveDirs {
		// Check for exact match or subdirectory of sensitive path
		// e.g., "/etc" matches "etc", "/var/run" matches "run" if parent is "/var"
		parts := strings.Split(sensitive, "/")
		if len(parts) > 1 {
			// Multi-level path like /var/run, check if the last part matches
			if parts[len(parts)-1] == dirName {
				return true
			}
		} else {
			// Single-level path like "etc", "root"
			if dirName == sensitive {
				return true
			}
		}
	}
	return false
}

// isRunningAsRoot checks if the current process is running as root user
func isRunningAsRoot() bool {
	return os.Getuid() == 0
}
