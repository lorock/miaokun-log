package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gitee.com/lorock/miaokun-log/internal/auth"
	"gitee.com/lorock/miaokun-log/internal/config"
	"gitee.com/lorock/miaokun-log/internal/discover"
	"gitee.com/lorock/miaokun-log/internal/searcher"
	"gitee.com/lorock/miaokun-log/internal/timefilter"
	"gitee.com/lorock/miaokun-log/pkg/types"
	"gitee.com/lorock/miaokun-log/pkg/version"
)

var logLevel int

// Server holds the HTTP server
type Server struct {
	port      string
	webDir    string
	cfgFile   string
	verbose   int
	authCfg   *auth.AuthConfig
	jwtMgr    *auth.JWTManager
	apiKeyMgr *auth.APIKeyManager
	userStore *auth.UserStore
	authMware *auth.JWTAuthMiddleware
}

// New creates a new server
func New(port, webDir, cfgFile string, verbose int, authCfg *auth.AuthConfig) *Server {
	return &Server{
		port:    port,
		webDir:  webDir,
		cfgFile: cfgFile,
		verbose: verbose,
		authCfg: authCfg,
	}
}

// InitializeAuth initializes authentication components
func (s *Server) InitializeAuth() error {
	if s.authCfg == nil || !s.authCfg.Enabled {
		return nil
	}

	// Issue3: Validate JWT secret — if empty, generate a random secret at startup
	// instead of using a hardcoded fallback value.
	if s.authCfg.JWTSecret == "" {
		randomSecret, err := auth.GenerateRandomSecret(32)
		if err != nil {
			return fmt.Errorf("failed to generate JWT secret: %w", err)
		}
		s.authCfg.JWTSecret = randomSecret
		fmt.Printf("  ⚠️  警告: 未提供 --jwt-secret，已自动生成随机 JWT 密钥\n")
		fmt.Printf("     本次会话密钥: %s\n", randomSecret)
		fmt.Printf("     生产部署请显式配置: --jwt-secret <至少32字符随机字符串>\n")
	}
	if len(s.authCfg.JWTSecret) < 16 {
		return fmt.Errorf("JWT secret 长度不足 16 字符，安全性不足")
	}

	if s.authCfg.AccessTokenTTL == 0 {
		s.authCfg.AccessTokenTTL = 24 * time.Hour
	}
	if s.authCfg.RefreshTokenTTL == 0 {
		s.authCfg.RefreshTokenTTL = 7 * 24 * time.Hour
	}

	jwtMgr, err := auth.NewJWTManager(s.authCfg.JWTSecret, s.authCfg.AccessTokenTTL, s.authCfg.RefreshTokenTTL)
	if err != nil {
		return fmt.Errorf("failed to create JWT manager: %w", err)
	}
	s.jwtMgr = jwtMgr

	// Initialize API Key Manager
	s.apiKeyMgr = auth.NewAPIKeyManager()

	// Initialize User Store with random admin password by default
	// Issue1: No more hardcoded "admin123". Generate random password if not provided.
	var adminPassword string
	s.userStore, adminPassword = auth.NewUserStore(s.authCfg.DefaultPassword)

	if len(s.authCfg.APIKeys) > 0 {
		for _, key := range s.authCfg.APIKeys {
			if key != "" {
				_, _ = s.apiKeyMgr.GenerateAPIKey("system", key, 0)
			}
		}
	}

	// Print admin credential info for operator
	fmt.Printf("  🔑 默认管理员账号: admin / %s\n", adminPassword)
	if s.authCfg.DefaultPassword == "" {
		fmt.Printf("     提示: 建议通过 --admin-password <自定义密码> 显式设置管理员密码\n")
	}

	// Initialize auth middleware
	s.authMware = auth.NewJWTAuthMiddleware(s.jwtMgr, s.apiKeyMgr)

	return nil
}

func (s *Server) Start() error {
	// Initialize auth if not already done
	if s.authMware == nil {
		if err := s.InitializeAuth(); err != nil {
			return err
		}
	}

	logLevel = s.verbose
	mux := http.NewServeMux()

	// Public endpoints (no authentication required)
	mux.HandleFunc("/api/v1/health", loggingMiddleware("health", handleHealth))
	mux.HandleFunc("/api/v1/version", loggingMiddleware("version", handleVersion))

	// Auth endpoints (no authentication required)
	if s.authMware != nil {
		authHandler := auth.NewAuthHandler(s.jwtMgr, s.userStore)
		mux.HandleFunc("/api/v1/auth/login", loggingMiddleware("auth-login", authHandler.Login))
		mux.HandleFunc("/api/v1/auth/refresh", loggingMiddleware("auth-refresh", authHandler.Refresh))
		mux.HandleFunc("/api/v1/auth/logout", loggingMiddleware("auth-logout", authHandler.Logout))
	}

	// Protected endpoints (authentication required)
	if s.authMware != nil {
		authHandler := auth.NewAuthHandler(s.jwtMgr, s.userStore)

		// User endpoints
		userHandler := authMiddleware(authHandler.GetCurrentUser, s.authMware.Authenticate())
		mux.HandleFunc("/api/v1/auth/me", loggingMiddleware("auth-me", userHandler))

		adminHandler := authMiddleware(authHandler.ListUsers,
			s.authMware.Authenticate(),
			s.authMware.RequireRole(auth.RoleAdmin))
		mux.HandleFunc("/api/v1/auth/users", loggingMiddleware("auth-users", adminHandler))

		// Protected API endpoints with permission checks
		protected := s.authMware.Authenticate()

		// File browsing requires file_browse permission
		filesHandler := authMiddleware(handleFiles, protected, s.authMware.RequirePermission(auth.PermFileBrowse))
		mux.HandleFunc("/api/v1/files", loggingMiddleware("files", filesHandler))

		// File list requires file_read permission
		fileListHandler := authMiddleware(handleFileList, protected, s.authMware.RequirePermission(auth.PermFileRead))
		mux.HandleFunc("/api/v1/files/list", loggingMiddleware("files-list", fileListHandler))

		// Paths discovery requires file_browse permission
		pathsHandler := authMiddleware(handlePaths(s.cfgFile), protected, s.authMware.RequirePermission(auth.PermFileBrowse))
		mux.HandleFunc("/api/v1/paths", loggingMiddleware("paths", pathsHandler))

		// Search requires search permission
		searchHandler := authMiddleware(handleSearch, protected, s.authMware.RequirePermission(auth.PermSearch))
		mux.HandleFunc("/api/v1/search", loggingMiddleware("search", searchHandler))

		// Stream search requires search permission
		streamHandler := authMiddleware(handleSearchStream, protected, s.authMware.RequirePermission(auth.PermSearch))
		mux.HandleFunc("/api/v1/search/stream", loggingMiddleware("search-stream", streamHandler))

		// Stats requires search permission
		statsHandler := authMiddleware(handleStats, protected, s.authMware.RequirePermission(auth.PermSearch))
		mux.HandleFunc("/api/v1/stats", loggingMiddleware("stats", statsHandler))

		// Trace requires search permission
		traceHandler := authMiddleware(handleTrace, protected, s.authMware.RequirePermission(auth.PermSearch))
		mux.HandleFunc("/api/v1/trace", loggingMiddleware("trace", traceHandler))
	} else {
		// No authentication - allow all endpoints
		mux.HandleFunc("/api/v1/files", loggingMiddleware("files", handleFiles))
		mux.HandleFunc("/api/v1/files/list", loggingMiddleware("files-list", handleFileList))
		mux.HandleFunc("/api/v1/paths", loggingMiddleware("paths", handlePaths(s.cfgFile)))
		mux.HandleFunc("/api/v1/search", loggingMiddleware("search", handleSearch))
		mux.HandleFunc("/api/v1/search/stream", loggingMiddleware("search-stream", handleSearchStream))
		mux.HandleFunc("/api/v1/stats", loggingMiddleware("stats", handleStats))
		mux.HandleFunc("/api/v1/trace", loggingMiddleware("trace", handleTrace))
	}

	// Static files and SPA
	mux.HandleFunc("/", loggingMiddleware("static", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		if r.URL.Path == "/" || r.URL.Path == "" {
			data, err := webAssets.ReadFile("web/dist/index.html")
			if err != nil {
				logAPIError("static", fmt.Sprintf("Failed to load index.html: %v", err))
				http.Error(w, fmt.Sprintf("Failed to load index.html: %v", err), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
			return
		}

		filePath := "web/dist" + r.URL.Path
		data, err := webAssets.ReadFile(filePath)
		if err != nil {
			data, err = webAssets.ReadFile("web/dist/index.html")
			if err != nil {
				logAPIError("static", fmt.Sprintf("File not found: %s (%v)", filePath, err))
				http.Error(w, fmt.Sprintf("File not found: %v", err), http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		} else {
			switch {
			case strings.HasSuffix(filePath, ".css"):
				w.Header().Set("Content-Type", "text/css")
			case strings.HasSuffix(filePath, ".js"):
				w.Header().Set("Content-Type", "application/javascript")
			case strings.HasSuffix(filePath, ".html"):
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
			}
		}
		w.Write(data)
	}))

	server := &http.Server{
		Addr:         ":" + s.port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 300 * time.Second,
	}

	authStatus := "已禁用"
	if s.authMware != nil {
		authStatus = "已启用"
	}

	fmt.Printf("🐾 喵坤 Web 服务启动\n")
	fmt.Printf("  📡 服务端口: %s\n", s.port)
	fmt.Printf("  🌐 访问地址: http://localhost:%s\n", s.port)
	fmt.Printf("  🔐 认证系统: %s\n", authStatus)
	if s.verbose >= 1 {
		fmt.Printf("  📝 API 日志: 已启用 (level=%d)\n", s.verbose)
	}
	fmt.Printf("\n")

	return server.ListenAndServe()
}

// authMiddleware wraps handlers with middleware
func authMiddleware(handler http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) http.HandlerFunc {
	var wrapped http.Handler = handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}
	return wrapped.ServeHTTP
}

type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func loggingMiddleware(apiName string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w}

		clientIP := getClientIP(r)
		queryParams := r.URL.RawQuery
		userAgent := r.UserAgent()

		if queryParams != "" {
			logAPIRequest(apiName, fmt.Sprintf("[%s] %s?%s from %s", r.Method, r.URL.Path, queryParams, clientIP))
		} else {
			logAPIRequest(apiName, fmt.Sprintf("[%s] %s from %s", r.Method, r.URL.Path, clientIP))
		}
		if userAgent != "" {
			logAPIDetail(apiName, fmt.Sprintf("User-Agent: %s", userAgent))
		}

		handler(rw, r)

		duration := time.Since(start)
		status := rw.status
		if status == 0 {
			status = http.StatusOK
		}

		logAPIResponse(apiName, fmt.Sprintf("[%s] %s %d %s %dB", r.Method, r.URL.Path, status, formatDuration(duration), rw.size))
	}
}

func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		parts := strings.Split(ip, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dμs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Milliseconds()))
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func logAPIRequest(apiName, msg string) {
	if logLevel >= 1 {
		fmt.Fprintf(os.Stderr, "[%s] [REQUEST]  %s %s\n", time.Now().Format("2006-01-02 15:04:05.000"), apiName, msg)
	}
}

func logAPIDetail(apiName, msg string) {
	if logLevel >= 2 {
		fmt.Fprintf(os.Stderr, "[%s] [DEBUG]    %s %s\n", time.Now().Format("2006-01-02 15:04:05.000"), apiName, msg)
	}
}

func logAPIInfo(apiName, msg string) {
	if logLevel >= 1 {
		fmt.Fprintf(os.Stderr, "[%s] [INFO]     %s %s\n", time.Now().Format("2006-01-02 15:04:05.000"), apiName, msg)
	}
}

func logAPIResponse(apiName, msg string) {
	if logLevel >= 1 {
		fmt.Fprintf(os.Stderr, "[%s] [RESPONSE] %s %s\n", time.Now().Format("2006-01-02 15:04:05.000"), apiName, msg)
	}
}

func logAPIError(apiName, msg string) {
	fmt.Fprintf(os.Stderr, "[%s] [ERROR]    %s %s\n", time.Now().Format("2006-01-02 15:04:05.000"), apiName, msg)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func handleFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var sinceDays float64 = 30
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if f, err := strconv.ParseFloat(sinceStr, 64); err == nil {
			sinceDays = f
		}
	}

	paths := []string{"/var/log", "/opt/logs"}
	if p := r.URL.Query().Get("path"); p != "" {
		paths = []string{p}
	}

	logAPIDetail("files", fmt.Sprintf("Params: since=%.0f, paths=%v", sinceDays, paths))

	if err := config.Load(""); err != nil {
		logAPIError("files", fmt.Sprintf("Config load failed: %v", err))
		http.Error(w, fmt.Sprintf("Config load failed: %v", err), http.StatusInternalServerError)
		return
	}

	start := time.Now()
	files, err := discover.FindLogs(paths, sinceDays)
	discoverDuration := time.Since(start)
	if err != nil {
		logAPIError("files", fmt.Sprintf("Find logs failed: %v", err))
		http.Error(w, fmt.Sprintf("Find logs failed: %v", err), http.StatusInternalServerError)
		return
	}

	logAPIInfo("files", fmt.Sprintf("Found %d log files (duration: %s)", len(files), formatDuration(discoverDuration)))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(files)
}

func handlePaths(cfgFile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var sinceDays float64 = 30
		if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
			if f, err := strconv.ParseFloat(sinceStr, 64); err == nil {
				sinceDays = f
			}
		}

		logAPIDetail("paths", fmt.Sprintf("Params: since=%.0f, cfgFile=%s", sinceDays, cfgFile))

		if err := config.Load(cfgFile); err != nil {
			logAPIError("paths", fmt.Sprintf("Config load failed: %v", err))
			http.Error(w, fmt.Sprintf("Config load failed: %v", err), http.StatusInternalServerError)
			return
		}

		cfg := config.Get()
		start := time.Now()
		paths := discover.DiscoverPaths(sinceDays, cfg.DefaultPaths...)
		discoverDuration := time.Since(start)

		logAPIInfo("paths", fmt.Sprintf("Found %d paths (duration: %s)", len(paths), formatDuration(discoverDuration)))

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(paths)
	}
}

type SearchRequest struct {
	Pattern         string   `json:"pattern"`
	Paths           []string `json:"paths"`
	Level           string   `json:"level"`
	Before          int      `json:"before"`
	After           int      `json:"after"`
	MaxCount        int      `json:"max_count"`
	CaseInsensitive bool     `json:"case_insensitive"`
	SinceDays       float64  `json:"since_days"`
	From            string   `json:"from"`
	To              string   `json:"to"`
}

func (r SearchRequest) summary() string {
	levelStr := "all"
	if r.Level != "" {
		levelStr = r.Level
	}
	fromStr := r.From
	if fromStr == "" {
		fromStr = "none"
	}
	toStr := r.To
	if toStr == "" {
		toStr = "none"
	}
	return fmt.Sprintf("pattern=%q, paths=%d, level=%s, case_insensitive=%v, max_count=%d, before=%d, after=%d, since=%.0fdays, from=%s, to=%s",
		r.Pattern, len(r.Paths), levelStr, r.CaseInsensitive, r.MaxCount, r.Before, r.After, r.SinceDays, fromStr, toStr)
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(version.GetInfo())
}

func buildOptsFromRequest(req SearchRequest) (types.SearchOptions, []string, error) {
	if req.MaxCount <= 0 {
		req.MaxCount = 50000
	}
	if req.SinceDays <= 0 {
		req.SinceDays = 30
	}

	if err := config.Load(""); err != nil {
		return types.SearchOptions{}, nil, err
	}

	paths := req.Paths
	if len(paths) == 0 {
		paths = []string{"/var/log", "/opt/logs"}
	}

	discoverStart := time.Now()
	files, err := discover.FindLogs(paths, req.SinceDays)
	if err != nil {
		return types.SearchOptions{}, nil, err
	}
	discoverDuration := time.Since(discoverStart)

	logPaths := make([]string, len(files))
	for i, f := range files {
		logPaths[i] = f.Path
	}

	logAPIDetail("search", fmt.Sprintf("Found %d files (duration: %s)", len(files), formatDuration(discoverDuration)))

	opts := types.SearchOptions{
		Pattern:         req.Pattern,
		Paths:           logPaths,
		Glob:            []string{"*.log", "*.log.gz", "*.log.*.gz"},
		MaxCount:        req.MaxCount,
		Before:          req.Before,
		After:           req.After,
		CaseInsensitive: req.CaseInsensitive,
		Level:           req.Level,
	}

	if req.From != "" {
		t, err := time.ParseInLocation("2006-01-02 15:04", req.From, time.Local)
		if err == nil {
			opts.From = t
		} else {
			// Try with seconds
			t, err = time.ParseInLocation("2006-01-02 15:04:05", req.From, time.Local)
			if err == nil {
				opts.From = t
			}
		}
	}
	if req.To != "" {
		t, err := time.ParseInLocation("2006-01-02 15:04", req.To, time.Local)
		if err == nil {
			opts.To = t
		} else {
			// Try with seconds
			t, err = time.ParseInLocation("2006-01-02 15:04:05", req.To, time.Local)
			if err == nil {
				opts.To = t
			}
		}
	}

	return opts, logPaths, nil
}

func applyTimeFilter(matches []types.LogMatch, from, to time.Time) []types.LogMatch {
	if from.IsZero() && to.IsZero() {
		return matches
	}

	tf := timefilter.New()
	if !from.IsZero() {
		_ = tf.SetFrom(from.Format("2006-01-02 15:04:05.000"))
	}
	if !to.IsZero() {
		_ = tf.SetTo(to.Format("2006-01-02 15:04:05.000"))
	}

	return tf.Filter(matches)
}

func containsI(s, substr string) bool {
	n := len(s)
	m := len(substr)
	if m == 0 {
		return true
	}
	if m > n {
		return false
	}

	upper := make([]byte, m)
	for i := 0; i < m; i++ {
		c := substr[i]
		if c >= 'a' && c <= 'z' {
			upper[i] = c - 32
		} else {
			upper[i] = c
		}
	}
	upperStr := string(upper)

	sBytes := make([]byte, n)
	for i := 0; i < n; i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			sBytes[i] = c - 32
		} else {
			sBytes[i] = c
		}
	}

	for i := 0; i <= n-m; i++ {
		if string(sBytes[i:i+m]) == upperStr {
			return true
		}
	}
	return false
}

func applyLevelFilter(matches []types.LogMatch, level string) []types.LogMatch {
	if level == "" {
		return matches
	}

	var filtered []types.LogMatch
	for _, m := range matches {
		if containsI(m.Raw, level) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logAPIError("search", fmt.Sprintf("Request parse failed: %v", err))
		http.Error(w, fmt.Sprintf("Request parse failed: %v", err), http.StatusBadRequest)
		return
	}

	logAPIDetail("search", fmt.Sprintf("Request params: %s", req.summary()))

	s := searcher.New()
	if err := s.CheckRipgrep(); err != nil {
		logAPIError("search", fmt.Sprintf("ripgrep check failed: %v", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	opts, logPaths, err := buildOptsFromRequest(req)
	if err != nil {
		logAPIError("search", fmt.Sprintf("Build options failed: %v", err))
		http.Error(w, fmt.Sprintf("Prepare search params failed: %v", err), http.StatusInternalServerError)
		return
	}

	logAPIDetail("search", fmt.Sprintf("Searching in %d files: %v", len(logPaths), pathsSummary(logPaths)))

	searchStart := time.Now()
	matches, err := s.Search(r.Context(), opts)
	searchDuration := time.Since(searchStart)
	if err != nil {
		logAPIError("search", fmt.Sprintf("Search failed: %v", err))
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	logAPIDetail("search", fmt.Sprintf("Raw matches: %d (duration: %s)", len(matches), formatDuration(searchDuration)))

	matches = applyTimeFilter(matches, opts.From, opts.To)
	matches = applyLevelFilter(matches, opts.Level)

	logAPIInfo("search", fmt.Sprintf("Final results: %d (level filter: %q, time: %s→%s)",
		len(matches), opts.Level, formatTime(opts.From), formatTime(opts.To)))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(matches)
}

func handleSearchStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logAPIError("search-stream", fmt.Sprintf("Request parse failed: %v", err))
		http.Error(w, fmt.Sprintf("Request parse failed: %v", err), http.StatusBadRequest)
		return
	}

	logAPIDetail("search-stream", fmt.Sprintf("Request params: %s", req.summary()))

	s := searcher.New()
	if err := s.CheckRipgrep(); err != nil {
		logAPIError("search-stream", fmt.Sprintf("ripgrep check failed: %v", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	opts, _, err := buildOptsFromRequest(req)
	if err != nil {
		logAPIError("search-stream", fmt.Sprintf("Build options failed: %v", err))
		http.Error(w, fmt.Sprintf("Prepare search params failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		logAPIError("search-stream", "Streaming not supported")
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	writeEvent := func(evType string, data interface{}) {
		dataBytes, _ := json.Marshal(map[string]interface{}{
			"type": evType,
			"data": data,
		})
		fmt.Fprintf(w, "data: %s\n\n", string(dataBytes))
		flusher.Flush()
	}

	ctx := r.Context()
	count := 0
	filteredCount := 0
	start := time.Now()

	searchStart := time.Now()
	matches, err := s.Search(ctx, opts)
	searchDuration := time.Since(searchStart)
	if err != nil {
		logAPIError("search-stream", fmt.Sprintf("Search failed: %v", err))
		writeEvent("error", map[string]string{"message": err.Error()})
		return
	}

	logAPIDetail("search-stream", fmt.Sprintf("Raw matches: %d (duration: %s)", len(matches), formatDuration(searchDuration)))

	matches = applyTimeFilter(matches, opts.From, opts.To)

	for _, m := range matches {
		select {
		case <-ctx.Done():
			logAPIDetail("search-stream", "Client disconnected")
			return
		default:
			count++
			if opts.Level == "" || containsI(m.Raw, opts.Level) {
				filteredCount++
				writeEvent("match", m)
			}
		}
	}

	totalDuration := time.Since(start)
	logAPIInfo("search-stream", fmt.Sprintf("Stream complete: raw=%d, filtered=%d (duration: %s)",
		count, filteredCount, formatDuration(totalDuration)))

	writeEvent("done", map[string]interface{}{
		"total_matches":    count,
		"filtered_matches": filteredCount,
		"duration_ms":      totalDuration.Milliseconds(),
	})
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logAPIError("stats", fmt.Sprintf("Request parse failed: %v", err))
		http.Error(w, fmt.Sprintf("Request parse failed: %v", err), http.StatusBadRequest)
		return
	}

	logAPIDetail("stats", fmt.Sprintf("Request params: %s", req.summary()))

	s := searcher.New()
	if err := s.CheckRipgrep(); err != nil {
		logAPIError("stats", fmt.Sprintf("ripgrep check failed: %v", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	opts, files, err := buildOptsFromRequest(req)
	if err != nil {
		logAPIError("stats", fmt.Sprintf("Build options failed: %v", err))
		http.Error(w, fmt.Sprintf("Prepare search params failed: %v", err), http.StatusInternalServerError)
		return
	}

	opts.Pattern = "ERROR|WARN|INFO|DEBUG|TRACE"
	opts.CaseInsensitive = true
	opts.Level = ""

	searchStart := time.Now()
	matches, err := s.Search(context.Background(), opts)
	searchDuration := time.Since(searchStart)
	if err != nil {
		logAPIError("stats", fmt.Sprintf("Search failed: %v", err))
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	logAPIDetail("stats", fmt.Sprintf("Stats matches: %d (duration: %s)", len(matches), formatDuration(searchDuration)))

	matches = applyTimeFilter(matches, opts.From, opts.To)

	stats := make(map[string]int)
	for _, m := range matches {
		level := extractLevel(m.Raw)
		stats[level]++
	}

	logAPIInfo("stats", fmt.Sprintf("Stats complete: total=%d, files=%d, levels=%v", len(matches), len(files), stats))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total":       len(matches),
		"by_level":    stats,
		"total_files": len(files),
	})
}

func handleTrace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logAPIError("trace", fmt.Sprintf("Request parse failed: %v", err))
		http.Error(w, fmt.Sprintf("Request parse failed: %v", err), http.StatusBadRequest)
		return
	}

	logAPIDetail("trace", fmt.Sprintf("Request params: %s", req.summary()))

	s := searcher.New()
	if err := s.CheckRipgrep(); err != nil {
		logAPIError("trace", fmt.Sprintf("ripgrep check failed: %v", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	opts, _, err := buildOptsFromRequest(req)
	if err != nil {
		logAPIError("trace", fmt.Sprintf("Build options failed: %v", err))
		http.Error(w, fmt.Sprintf("Prepare search params failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		logAPIError("trace", "Streaming not supported")
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	writeEvent := func(evType string, data interface{}) {
		dataBytes, _ := json.Marshal(map[string]interface{}{
			"type": evType,
			"data": data,
		})
		fmt.Fprintf(w, "data: %s\n\n", string(dataBytes))
		flusher.Flush()
	}

	ctx := r.Context()
	count := 0

	searchStart := time.Now()
	matches, err := s.Search(ctx, opts)
	searchDuration := time.Since(searchStart)
	if err != nil {
		logAPIError("trace", fmt.Sprintf("Search failed: %v", err))
		writeEvent("error", map[string]string{"message": err.Error()})
		return
	}

	logAPIDetail("trace", fmt.Sprintf("Raw matches: %d (duration: %s)", len(matches), formatDuration(searchDuration)))

	matches = applyTimeFilter(matches, opts.From, opts.To)

	for _, m := range matches {
		select {
		case <-ctx.Done():
			logAPIDetail("trace", "Client disconnected")
			return
		default:
			count++
			writeEvent("match", m)
		}
	}

	totalDuration := time.Since(searchStart)
	logAPIInfo("trace", fmt.Sprintf("Trace complete: total=%d (duration: %s)", count, formatDuration(totalDuration)))

	writeEvent("done", map[string]interface{}{
		"total_matches": count,
	})
}

func extractLevel(line string) string {
	upper := make([]byte, 0, len(line))
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c >= 'a' && c <= 'z' {
			upper = append(upper, c-32)
		} else {
			upper = append(upper, c)
		}
	}
	upperStr := string(upper)

	levels := []string{"ERROR", "WARN", "INFO", "DEBUG", "TRACE"}
	for _, l := range levels {
		if contains(upperStr, l) {
			return l
		}
	}
	return "UNKNOWN"
}

func contains(s, substr string) bool {
	n := len(s)
	m := len(substr)
	if m == 0 {
		return true
	}
	for i := 0; i <= n-m; i++ {
		if s[i:i+m] == substr {
			return true
		}
	}
	return false
}

func pathsSummary(paths []string) string {
	if len(paths) <= 5 {
		return fmt.Sprintf("%v", paths)
	}
	first := paths[:3]
	last := paths[len(paths)-1:]
	return fmt.Sprintf("%v ... %v (total %d)", first, last, len(paths))
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "not set"
	}
	return t.Format("2006-01-02 15:04")
}
