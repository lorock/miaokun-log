package trace

import (
	"regexp"

	"gitee.com/lorock/miaokun-log/pkg/types"
)

var patterns = []*regexp.Regexp{
	regexp.MustCompile(`traceId[=\s:]+([a-zA-Z0-9_\-]{6,64})`),
	regexp.MustCompile(`trace_id[=\s:]+([a-zA-Z0-9_\-]{6,64})`),
	regexp.MustCompile(`"traceId"\s*:\s*"([a-zA-Z0-9_\-]{6,64})"`),
	regexp.MustCompile(`X-B3-TraceId[=\s:]+([a-zA-Z0-9_\-]{6,64})`),
}

func Extract(line string) string {
	for _, re := range patterns {
		if m := re.FindStringSubmatch(line); len(m) >= 2 {
			return m[1]
		}
	}
	return ""
}

func Correlate(matches []types.LogMatch) map[string][]types.LogMatch {
	correlated := make(map[string][]types.LogMatch)
	for _, m := range matches {
		if tid := Extract(m.Raw); tid != "" {
			correlated[tid] = append(correlated[tid], m)
		}
	}
	return correlated
}
