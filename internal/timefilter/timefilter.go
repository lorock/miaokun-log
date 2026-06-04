package timefilter

import (
	"fmt"
	"regexp"
	"time"

	"gitee.com/lorock/miaokun-log/pkg/types"
)

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
		if !tf.from.IsZero() && ts.Before(tf.from) {
			continue
		}
		if !tf.to.IsZero() && ts.After(tf.to) {
			continue
		}
		filtered = append(filtered, m)
	}
	return filtered
}

func parseTime(s string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("无效的时间格式: %s", s)
}

func extractTimestamp(line string) time.Time {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\[(\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)\]`),
		regexp.MustCompile(`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})`),
		regexp.MustCompile(`(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})`),
	}

	for _, re := range patterns {
		if m := re.FindStringSubmatch(line); len(m) >= 2 {
			t, err := parseTime(m[1])
			if err == nil {
				return t
			}
		}
	}
	return time.Time{}
}
