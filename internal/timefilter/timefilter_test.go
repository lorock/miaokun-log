package timefilter

import (
	"testing"
	"time"
)

// 使用包级 cst（与 timefilter.go 中的 var cst 保持一致）
// 注意：测试文件中不能直接访问包级私有变量，所以重新定义
var testCST = time.FixedZone("CST", 8*3600)

func TestExtractTimestamp_GoFormats(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string // CST time, empty = zero
	}{
		{"go_std_log", `[I] 2026/06/07 01:15:40 hybrid_storage.go:330: msg`, "2026-06-07 01:15:40"},
		{"go_slash_bare", `2026/06/07 01:15:40 INFO msg`, "2026-06-07 01:15:40"},
		{"go_json_time_field", `{"time":"2026-06-07T01:15:40.984+08:00","level":"INFO"}`, "2026-06-07 01:15:40"},
		{"go_json_timestamp_field", `{"timestamp":"2026-06-07 01:15:40.984","level":"INFO"}`, "2026-06-07 01:15:40"},
		{"bracket_millis", `[2026-06-07 00:35:31.984] [INFO] msg`, "2026-06-07 00:35:31"},
		{"bare_iso_T", `2026-06-07T00:35:31.984 INFO msg`, "2026-06-07 00:35:31"},
		// UTC Z 后缀，应转 CST +8
		{"utc_Z_suffix", `2026-06-07T01:15:31Z [INFO] msg`, "2026-06-07 09:15:31"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := extractTimestamp(tt.line)
			if tt.expected == "" {
				if !ts.IsZero() {
					t.Errorf("expected zero, got %v", ts)
				}
				return
			}
			want, err := time.ParseInLocation("2006-01-02 15:04:05", tt.expected, testCST)
			if err != nil {
				t.Fatalf("bad expected value %q: %v", tt.expected, err)
			}
			// 统一转到 CST 后比较 Unix 时间戳，避免时区对象不一致问题
			if ts.Unix() != want.Unix() {
				t.Errorf("got %v (unix:%d), want %v (unix:%d)",
					ts.Format("2006-01-02 15:04:05"), ts.Unix(),
					tt.expected, want.Unix())
			}
		})
	}
}

func TestExtractTimestamp_JavaFormats(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{"java_logback", `2026-06-07 01:15:40.984 [INFO] com.foo.Bar - message`, "2026-06-07 01:15:40"},
		{"java_european_date", `07-06-2026 01:15:40.984 INFO  [main] com.foo.Bar`, "2026-06-07 01:15:40"},
		{"java_log4j2_iso_tz", `2026-06-07T01:15:40.984+08:00 INFO  [main]`, "2026-06-07 01:15:40"},
		{"java_english_date_am", `Jun 07, 2026 1:15:40 AM INFO message`, "2026-06-07 01:15:40"},
		{"java_english_date_24h", `Jun 07, 2026 13:15:40 INFO message`, "2026-06-07 13:15:40"},
		{"comma_millis", `2026-06-07 01:15:40,984 INFO  [main]`, "2026-06-07 01:15:40"},
		{"java_json_at_timestamp", `<log @timestamp="2026-06-07T01:15:40.984+08:00"/>`, "2026-06-07 01:15:40"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := extractTimestamp(tt.line)
			if tt.expected == "" {
				if !ts.IsZero() {
					t.Errorf("expected zero, got %v", ts)
				}
				return
			}
			want, err := time.ParseInLocation("2006-01-02 15:04:05", tt.expected, testCST)
			if err != nil {
				t.Fatalf("bad expected value %q: %v", tt.expected, err)
			}
			if ts.Unix() != want.Unix() {
				t.Errorf("got %v (unix:%d), want %v (unix:%d)",
					ts.Format("2006-01-02 15:04:05"), ts.Unix(),
					tt.expected, want.Unix())
			}
		})
	}
}

func TestExtractTimestamp_NoTimestamp(t *testing.T) {
	lines := []string{
		"this is a log line without any timestamp",
		"just some random text",
	}
	for _, line := range lines {
		ts := extractTimestamp(line)
		if !ts.IsZero() {
			t.Errorf("expected zero for %q, got %v", line, ts)
		}
	}
}

func TestParseTime_CSTConversion(t *testing.T) {
	tests := []struct {
		input    string
		expected string // CST
	}{
		{"2026-06-07 01:15:40", "2026-06-07 01:15:40"},
		{"2026-06-07 01:15:40.984", "2026-06-07 01:15:40"},
		{"2026/06/07 01:15:40", "2026-06-07 01:15:40"},
		{"07-06-2026 01:15:40", "2026-06-07 01:15:40"},
		{"2026-06-07T01:15:40.984+08:00", "2026-06-07 01:15:40"},
		{"2026-06-07T01:15:31Z", "2026-06-07 09:15:31"}, // UTC+8
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseTime(tt.input)
			if err != nil {
				t.Fatalf("parseTime(%q) error: %v", tt.input, err)
			}
			want, _ := time.ParseInLocation("2006-01-02 15:04:05", tt.expected, testCST)
			if got.Unix() != want.Unix() {
				t.Errorf("got %v (unix:%d), want %v (unix:%d)",
					got.Format("2006-01-02 15:04:05 MST"), got.Unix(),
					tt.expected, want.Unix())
			}
		})
	}
}
