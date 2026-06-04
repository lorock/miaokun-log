package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gitee.com/lorock/miaokun-log/pkg/types"
	"github.com/itchyny/gojq"
)

type Output struct {
	json    bool
	jq      string
	noColor bool
}

func New(json bool, jq string, noColor bool) *Output {
	// 检测是否禁用彩色
	disabled := noColor || os.Getenv("NO_COLOR") != ""
	return &Output{json: json, jq: jq, noColor: disabled}
}

// 颜色常量
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[1;32m"
	colorYellow = "\033[1;33m"
	colorCyan   = "\033[1;36m"
	colorGray   = "\033[90m"
)

// 颜色辅助函数
func (o *Output) c(color string) string {
	if o.noColor {
		return ""
	}
	return color
}

func (o *Output) PrintMatches(matches []types.LogMatch) {
	if o.json {
		o.PrintJSON(matches)
		return
	}

	if len(matches) == 0 {
		fmt.Println("  (ﾉﾟ0ﾟ)ﾉ~ 喵坤挠了个空，没捞到匹配")
		return
	}

	for i, m := range matches {
		if i >= 50 {
			fmt.Printf("  … 还有 %d 条，加 --json 看全量\n", len(matches)-50)
			break
		}
		o.PrintMatch(m)
	}
	
	fmt.Printf("\n  🐾 喵坤 · 命中 %s%d%s 条\n",
		o.c(colorGreen), len(matches), o.c(colorReset))
}

func (o *Output) PrintMatch(m types.LogMatch) {
	hasContext := len(m.BeforeContext) > 0 || len(m.AfterContext) > 0 || len(m.Context) > 0
	shortPath := trimPath(m.File, 38)
	
	if hasContext {
		fmt.Printf("\n  %s%-40s%s %s:%s%s%-5d%s\n",
			o.c(colorCyan), shortPath, o.c(colorReset),
			o.c(colorGray), o.c(colorReset),
			o.c(colorYellow), m.LineNum, o.c(colorReset))
		
		// 显示 BeforeContext 或者使用 Context（向后兼容）
		if len(m.BeforeContext) > 0 {
			for _, ctxLine := range m.BeforeContext {
				fmt.Printf("  %s  |%s %s\n", o.c(colorGray), o.c(colorReset), truncate(ctxLine, 180))
			}
		} else if len(m.Context) > 0 {
			for _, ctxLine := range m.Context {
				fmt.Printf("  %s  |%s %s\n", o.c(colorGray), o.c(colorReset), truncate(ctxLine, 180))
			}
		}
		
		fmt.Printf("  %s> |%s %s\n", o.c(colorGreen), o.c(colorReset), truncate(m.Raw, 180))
		
		// 显示 AfterContext
		if len(m.AfterContext) > 0 {
			for _, ctxLine := range m.AfterContext {
				fmt.Printf("  %s  |%s %s\n", o.c(colorGray), o.c(colorReset), truncate(ctxLine, 180))
			}
		}
	} else {
		fmt.Printf("  %s%-40s%s %s:%s%s%-5d%s %s\n",
			o.c(colorCyan), shortPath, o.c(colorReset),
			o.c(colorGray), o.c(colorReset),
			o.c(colorYellow), m.LineNum, o.c(colorReset),
			truncate(m.Raw, 180))
	}
}

func (o *Output) PrintTraceCorrelation(correlated map[string][]types.LogMatch) {
	if o.json {
		o.PrintJSON(correlated)
		return
	}

	for tid, matches := range correlated {
		fmt.Printf("\n%s🔍 TraceId: %s%s (%s%d%s 个事件)\n",
			o.c(colorYellow), tid, o.c(colorReset),
			o.c(colorCyan), len(matches), o.c(colorReset))
		for i, m := range matches {
			if i >= 5 {
				fmt.Printf("  … 还有 %d 个\n", len(matches)-5)
				break
			}
			
			hasContext := len(m.Context) > 0
			shortPath := trimPath(m.File, 32)
			
			if hasContext {
				fmt.Printf("  %s%-34s%s %s:%s%s%-5d%s\n",
					o.c(colorCyan), shortPath, o.c(colorReset),
					o.c(colorGray), o.c(colorReset),
					o.c(colorYellow), m.LineNum, o.c(colorReset))
				for _, ctxLine := range m.Context {
					fmt.Printf("  %s  |%s %s\n", o.c(colorGray), o.c(colorReset), truncate(ctxLine, 140))
				}
				fmt.Printf("  %s> |%s %s\n", o.c(colorGreen), o.c(colorReset), truncate(m.Raw, 140))
			} else {
				fmt.Printf("  %s%-34s%s %s:%s%s%-5d%s  %s\n",
					o.c(colorCyan), shortPath, o.c(colorReset),
					o.c(colorGray), o.c(colorReset),
					o.c(colorYellow), m.LineNum, o.c(colorReset),
					truncate(m.Raw, 140))
			}
		}
	}
}

func (o *Output) PrintStats(matches []types.LogMatch) {
	stats := make(map[string]int)
	for _, m := range matches {
		level := extractLevel(m.Raw)
		stats[level]++
	}

	if o.json {
		o.PrintJSON(stats)
		return
	}

	fmt.Println("\n📊 统计信息:")
	for level, count := range stats {
		bar := strings.Repeat("█", min(count/10, 50))
		fmt.Printf("  %-8s %-50s (%s%d%s)\n", level, bar,
			o.c(colorGreen), count, o.c(colorReset))
	}
}

func trimPath(p string, max int) string {
	parts := strings.Split(p, "/")
	if len(parts) <= 2 {
		return p
	}
	short := strings.Join(parts[len(parts)-2:], "/")
	if len(short) > max {
		short = "…" + short[len(short)-max+1:]
	}
	return short
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func extractLevel(line string) string {
	upper := strings.ToUpper(line)
	levels := []string{"ERROR", "WARN", "INFO", "DEBUG", "TRACE"}
	for _, l := range levels {
		if strings.Contains(upper, l) {
			return l
		}
	}
	return "UNKNOWN"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (o *Output) PrintJSON(data interface{}) {
	var result interface{}

	// 如果设置了 jq 查询表达式
	if o.jq != "" {
		query, err := gojq.Parse(o.jq)
		if err != nil {
			fmt.Fprintf(os.Stderr, "jq 查询语法错误: %v\n", err)
			os.Exit(1)
		}

		// 将数据转换为 gojq 可以处理的格式
		var input interface{}
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "JSON 序列化错误: %v\n", err)
			os.Exit(1)
		}
		if err := json.Unmarshal(jsonBytes, &input); err != nil {
			fmt.Fprintf(os.Stderr, "JSON 反序列化错误: %v\n", err)
			os.Exit(1)
		}

		// 执行 jq 查询
		iter := query.Run(input)
		var results []interface{}
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok := v.(error); ok {
				fmt.Fprintf(os.Stderr, "jq 查询执行错误: %v\n", err)
				os.Exit(1)
			}
			results = append(results, v)
		}

		// 如果只有一个结果，直接输出该结果；否则输出数组
		if len(results) == 1 {
			result = results[0]
		} else {
			result = results
		}
	} else {
		result = data
	}

	// 输出 JSON
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "JSON 输出错误: %v\n", err)
		os.Exit(1)
	}
}
