package timefilter

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"gitee.com/lorock/miaokun-log/pkg/types"
)

// cst 东八区时区
var cst = time.FixedZone("CST", 8*3600)

// ---------------------------------------------------------------------------
// TimeFilter 结构体
// ---------------------------------------------------------------------------

type TimeFilter struct {
	from time.Time
	to   time.Time
}

func New() *TimeFilter {
	return &TimeFilter{}
}

func (tf *TimeFilter) SetFrom(s string) error {
	t, err := parseTime(s)
	if err != nil {
		return err
	}
	tf.from = t
	return nil
}

func (tf *TimeFilter) SetTo(s string) error {
	t, err := parseTime(s)
	if err != nil {
		return err
	}
	tf.to = t
	return nil
}

func (tf *TimeFilter) From() time.Time {
	return tf.from
}

func (tf *TimeFilter) To() time.Time {
	return tf.to
}

// Filter 按 from/to 过滤日志匹配结果。
// 每条日志通过 extractTimestamp 提取时间戳，自动转换为 CST（东八区）后比较。
func (tf *TimeFilter) Filter(matches []types.LogMatch) []types.LogMatch {
	if tf.from.IsZero() && tf.to.IsZero() {
		return matches
	}

	var filtered []types.LogMatch
	for _, m := range matches {
		ts := extractTimestamp(m.Raw)
		if ts.IsZero() {
			continue
		}
		// 统一转到 CST 再比较，避免时区不一致导致误判
		tsCST := ts.In(cst)
		if !tf.from.IsZero() && tsCST.Before(tf.from) {
			continue
		}
		if !tf.to.IsZero() && tsCST.After(tf.to) {
			continue
		}
		filtered = append(filtered, m)
	}
	return filtered
}

// ---------------------------------------------------------------------------
// 时间戳提取 — 支持 Go / Java 常见日志格式
// ---------------------------------------------------------------------------

// extractTimestamp 从一行日志中提取时间戳，自动转换为 CST（东八区）。
// 支持格式见下方正则表达式注释。
func extractTimestamp(line string) time.Time {
	patterns := []*regexp.Regexp{
		// ① [2026-06-07 00:35:31.984] 或 [2026-06-07T00:35:31Z] 或 [2026-06-07 00:35:31+08:00]
		//    方括号包裹，最通用；支持 T 分隔符、毫秒、时区后缀
		regexp.MustCompile(`\[(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)\]`),

		// ② 2026-06-07 00:35:31.984 或 2026-06-07 00:35:31
		//    bare 格式，无方括号，横杠日期
		regexp.MustCompile(`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)`),

		// ③ 2026-06-07T00:35:31.984 或 2026-06-07T00:35:31Z
		//    ISO 8601，T 分隔，无方括号
		regexp.MustCompile(`(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)`),

		// ④ 2026/06/07 00:35:31 — Go 常见 / 亚洲格式
		regexp.MustCompile(`(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})`),

		// ⑤ [I] 2026/06/07 00:35:31 file.go:123: — Go 标准 log 格式
		//    [I] / [W] / [ERRO] 等日志级别后跟时间
		regexp.MustCompile(`\]\s+(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})\s+`),

		// ⑥ 07-06-2026 00:35:31.984 或 07-06-2026 00:35:31 — Java Log4j/Logback 欧式日期（dd-MM-yyyy）
		regexp.MustCompile(`(\d{2}-\d{2}-\d{4} \d{2}:\d{2}:\d{2}(?:\.\d+)?)`),

		// ⑦ JSON 日志 — Go slog / zap / logrus JSON 格式
		//    {"time":"2026-06-07T00:35:31.984+08:00",...} 或 {"timestamp":"2026-06-07 00:35:31",...}
		//    支持 "time" / "timestamp" / "ts" / "@timestamp" 字段
		regexp.MustCompile(`"(?:time|timestamp|ts|@timestamp)"\s*:\s*"(\d{4}[^"]{15,})"`),

		// ⑧ Jun 07, 2026 1:15:40 AM 或 June 07, 2026 13:15:40 — Java 英文日期格式
		regexp.MustCompile(`(?i)((?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]* \d{1,2}, \d{4} \d{1,2}:\d{2}:\d{2} (?:AM|PM))`),
		regexp.MustCompile(`(?i)((?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]* \d{1,2}, \d{4} \d{1,2}:\d{2}:\d{2})`),
	}

	for _, re := range patterns {
		if m := re.FindStringSubmatch(line); len(m) >= 2 {
			t, err := parseTime(m[1])
			if err == nil {
				return t.In(cst)
			}
		}
	}
	return time.Time{}
}

// ---------------------------------------------------------------------------
// 时间字符串解析 — 支持几乎所有常见 layout，自动转 CST
// ---------------------------------------------------------------------------

// parseTime 解析时间字符串，优先处理带时区的 ISO 8601 / RFC3339 格式（自动转 CST），
// 其余无时区信息的一律按 CST（东八区）解析。
func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)

	// 1. 先尝试标准库 RFC3339 / RFC3339Nano（自带时区，自动正确转换）
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.In(cst), nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.In(cst), nil
	}

	// 2. 自定义带时区偏移的格式（如 "2026-06-07 00:35:31+08:00"）
	//    注意 layout 中的 Z07:00 表示解析 ±HH:MM 格式时区
	if t, err := time.Parse("2006-01-02 15:04:05.999Z07:00", s); err == nil {
		return t.In(cst), nil
	}
	if t, err := time.Parse("2006-01-02T15:04:05.999Z07:00", s); err == nil {
		return t.In(cst), nil
	}
	// 无冒号的时区偏移，如 +0800
	if t, err := time.Parse("2006-01-02 15:04:05.999Z0700", s); err == nil {
		return t.In(cst), nil
	}

	// 3. 无时区信息 → 按 CST 解析
	//    注意：用 cst（FixedZone）确保 ParseInLocation 将时间解释为东八区
	layouts := []string{
		// yyyy-MM-dd 横杠格式（最常见）
		"2006-01-02 15:04:05.999",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		// yyyy-MM-ddTHH:mm:ss ISO 8601 无时区
		"2006-01-02T15:04:05.999",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		// yyyy/MM/dd 斜杠格式（Go 常见）
		"2006/01/02 15:04:05.999",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		// dd-MM-yyyy 欧式（Java Log4j 常见）
		"02-01-2006 15:04:05.999",
		"02-01-2006 15:04:05",
		"02-01-2006 15:04",
		// 逗号毫秒（欧洲格式：2026-06-07 00:35:31,984）
		"2006-01-02 15:04:05,999",
		// 英文日期：Jun 07, 2026 1:15:40 AM
		"Jan 02, 2006 3:04:05 PM",
		"Jan 02, 2006 15:04:05",
		"Jan 02, 2006 3:04 PM",
		"Jan 02, 2006 15:04",
		// 英文日期（无逗号）：Jun 07 2026 1:15:40 AM
		"Jan 02 2006 3:04:05 PM",
		"Jan 02 2006 15:04:05",
	}

	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, cst); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("无效的时间格式: %s", s)
}
