package types

import "time"

type LogMatch struct {
	File          string    `json:"file"`
	LineNum       int       `json:"line_num"`
	Timestamp     time.Time `json:"timestamp,omitempty"`
	Level         string    `json:"level,omitempty"`
	TraceID       string    `json:"trace_id,omitempty"`
	Raw           string    `json:"raw"`
	Context       []string  `json:"context,omitempty"` // Deprecated: use Before and After`
	BeforeContext []string `json:"before_context,omitempty"`
	AfterContext  []string  `json:"after_context,omitempty"`
}

type SearchOptions struct {
	Pattern        string
	Paths          []string
	Glob           []string
	From           time.Time
	To             time.Time
	Before         int
	After          int
	MaxCount       int
	Color          bool
	Format         string // "terminal", "json"
	CaseInsensitive bool   // 大小写不敏感搜索
	Level          string // 按日志级别搜索 (ERROR, WARN, INFO, DEBUG, TRACE)
}

type FileInfo struct {
	Path    string
	Size    int64
	ModTime time.Time
	IsGzip  bool
}

type LogStats struct {
	TotalMatches int            `json:"total_matches"`
	ByFile       map[string]int `json:"by_file"`
	ByLevel      map[string]int `json:"by_level"`
	TotalFiles   int            `json:"total_files"`
}
