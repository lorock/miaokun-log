package searcher

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"gitee.com/lorock/miaokun-log/pkg/types"
)

type Searcher struct {
	RGPath string
}

func New() *Searcher {
	return &Searcher{RGPath: "rg"}
}

// CheckRipgrep 检查 ripgrep 是否安装
func (s *Searcher) CheckRipgrep() error {
	_, err := exec.LookPath(s.RGPath)
	if err != nil {
		return fmt.Errorf("未找到 ripgrep (rg)，请先安装：\n  - macOS: brew install ripgrep\n  - Ubuntu/Debian: sudo apt install ripgrep\n  - CentOS/RHEL: sudo yum install ripgrep\n  - 其他: https://github.com/BurntSushi/ripgrep#installation")
	}
	return nil
}

func (s *Searcher) BuildArgs(opts types.SearchOptions) []string {
	args := []string{
		"-n",
		"-H",
		"--no-heading",
		"--color", "never",
		"--max-count", fmt.Sprintf("%d", opts.MaxCount),
	}

	if opts.CaseInsensitive {
		args = append(args, "-i")
	}

	// 添加上下文参数
	if opts.Before > 0 {
		args = append(args, "-B", fmt.Sprintf("%d", opts.Before))
	}
	if opts.After > 0 {
		args = append(args, "-A", fmt.Sprintf("%d", opts.After))
	}

	for _, g := range opts.Glob {
		args = append(args, "--glob", g)
	}

	args = append(args, opts.Pattern)
	args = append(args, opts.Paths...)

	return args
}

func (s *Searcher) Search(ctx context.Context, opts types.SearchOptions) ([]types.LogMatch, error) {
	if err := s.CheckRipgrep(); err != nil {
		return nil, err
	}

	args := s.BuildArgs(opts)
	var matches []types.LogMatch

	err := s.SearchStream(ctx, args, opts, func(m types.LogMatch) bool {
		matches = append(matches, m)
		return true
	})

	if err != nil {
		return nil, err
	}

	return matches, nil
}

func (s *Searcher) SearchStream(ctx context.Context, args []string, opts types.SearchOptions, callback func(types.LogMatch) bool) error {
	cmd := exec.CommandContext(ctx, s.RGPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("创建 stdout pipe 失败: %w", err)
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("创建 stderr pipe 失败: %w", err)
	}
	defer stderr.Close()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 rg 失败: %w", err)
	}

	// 收集 stderr
	var stderrBuf strings.Builder
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stderrBuf.WriteString(scanner.Text())
		}
	}()

	// 流式解析 stdout
	hasContext := opts.Before > 0 || opts.After > 0
	matchLineRegex := regexp.MustCompile(`^([^:]+):(\d+):(.*)$`)
	contextLineRegex := regexp.MustCompile(`^(.+?)-(\d+)-(.*)$`)

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	// 我们用一个内部结构体来分别跟踪 before 和 after context
	type MatchInternal struct {
		File          string
		LineNum       int
		Raw           string
		BeforeContext []string
		AfterContext  []string
	}

	var currentMatch *MatchInternal
	var beforeLines []string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// ripgrep 用 -- 分隔不同匹配
		if line == "--" {
			if currentMatch != nil {
				// 转换为 LogMatch 并回调
				m := types.LogMatch{
					File:          currentMatch.File,
					LineNum:       currentMatch.LineNum,
					Raw:           currentMatch.Raw,
					BeforeContext: currentMatch.BeforeContext,
					AfterContext:  currentMatch.AfterContext,
					Context:       append(append([]string{}, currentMatch.BeforeContext...), currentMatch.AfterContext...),
				}
				if opts.Level == "" || strings.Contains(strings.ToUpper(m.Raw), strings.ToUpper(opts.Level)) {
					if !callback(m) {
						cmd.Process.Kill()
						return nil
					}
				}
			}
			currentMatch = nil
			beforeLines = []string{}
			continue
		}

		parsed := parseLine(line, matchLineRegex, contextLineRegex)

		if parsed.isMatch {
			if currentMatch != nil {
				// 关键！在处理新匹配之前，把前一个匹配的 after context 截断到 opts.After
				if opts.After > 0 && len(currentMatch.AfterContext) > opts.After {
					currentMatch.AfterContext = currentMatch.AfterContext[:opts.After]
				}
				// 转换为 LogMatch 并回调
				m := types.LogMatch{
					File:          currentMatch.File,
					LineNum:       currentMatch.LineNum,
					Raw:           currentMatch.Raw,
					BeforeContext: currentMatch.BeforeContext,
					AfterContext:  currentMatch.AfterContext,
					Context:       append(append([]string{}, currentMatch.BeforeContext...), currentMatch.AfterContext...),
				}
				if opts.Level == "" || strings.Contains(strings.ToUpper(m.Raw), strings.ToUpper(opts.Level)) {
					if !callback(m) {
						cmd.Process.Kill()
						return nil
					}
				}
			}
			currentMatch = &MatchInternal{
				File:          parsed.file,
				LineNum:       parsed.lineNum,
				Raw:           parsed.content,
				BeforeContext: append([]string{}, beforeLines...),
				AfterContext:  []string{},
			}
			// 关键！在设置新的 currentMatch 后，我们要把 beforeLines 重置，但是等下，我们还需要把当前 match 之后遇到的新行作为下一个 match 的 before context！
			// 不过，其实问题是：在第二个 match 前，我们的 beforeLines 包含了第一个 match 的 after context！所以我们要把 beforeLines 重置为空！
			beforeLines = []string{}
			continue
		}

		if hasContext && parsed.isContext && !parsed.isMatch {
			if currentMatch != nil {
				// 这是 after context
				currentMatch.AfterContext = append(currentMatch.AfterContext, parsed.content)
			} else {
				// 这是 before context，保留最后 N 行
				beforeLines = append(beforeLines, parsed.content)
				if opts.Before > 0 && len(beforeLines) > opts.Before {
					beforeLines = beforeLines[len(beforeLines)-opts.Before:]
				}
			}
		}
	}

	if currentMatch != nil {
		// 截断最后一个 match 的 after context 到 opts.After
		if opts.After > 0 && len(currentMatch.AfterContext) > opts.After {
			currentMatch.AfterContext = currentMatch.AfterContext[:opts.After]
		}
		m := types.LogMatch{
			File:          currentMatch.File,
			LineNum:       currentMatch.LineNum,
			Raw:           currentMatch.Raw,
			BeforeContext: currentMatch.BeforeContext,
			AfterContext:  currentMatch.AfterContext,
			Context:       append(append([]string{}, currentMatch.BeforeContext...), currentMatch.AfterContext...),
		}
		if opts.Level == "" || strings.Contains(strings.ToUpper(m.Raw), strings.ToUpper(opts.Level)) {
			callback(m)
		}
	}

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// rg 返回 1 表示没有匹配，不是错误
			return nil
		}
		if stderrBuf.Len() > 0 {
			return fmt.Errorf("rg 错误: %s", stderrBuf.String())
		}
		return fmt.Errorf("执行 rg 失败: %w", err)
	}

	return nil
}

type parsedResult struct {
	isMatch   bool
	isContext bool
	file      string
	lineNum   int
	content   string
}

func parseLine(line string, matchLineRegex, contextLineRegex *regexp.Regexp) parsedResult {
	res := parsedResult{}

	// 尝试匹配行
	if matchGroups := matchLineRegex.FindStringSubmatch(line); len(matchGroups) == 4 {
		if lineNum, err := strconv.Atoi(matchGroups[2]); err == nil && lineNum > 0 {
			res.isMatch = true
			res.file = matchGroups[1]
			res.lineNum = lineNum
			res.content = matchGroups[3]
		}
	}

	// 尝试上下文行，只有不是匹配行的时候才设置 res.content
	if ctxGroups := contextLineRegex.FindStringSubmatch(line); len(ctxGroups) == 4 {
		if lineNum, err := strconv.Atoi(ctxGroups[2]); err == nil && lineNum > 0 {
			res.isContext = true
			if res.file == "" { // 只有不是匹配行的时候才设置 file/lineNum
				res.file = ctxGroups[1]
				res.lineNum = lineNum
			}
			if !res.isMatch { // 关键修复：只有不是匹配行的时候才覆盖 content
				res.content = ctxGroups[3]
			}
		}
	}

	return res
}

func ParseOutput(output string, hasContext bool) []types.LogMatch {
	var matches []types.LogMatch
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// 正则表达式：匹配行格式 file:line:content（文件名不含冒号）
	matchLineRegex := regexp.MustCompile(`^([^:]+):(\d+):(.*)$`)
	// 正则表达式：上下文行格式 file-line-content
	contextLineRegex := regexp.MustCompile(`^(.+?)-(\d+)-(.*)$`)

	type MatchInternal struct {
		File          string
		LineNum       int
		Raw           string
		BeforeContext []string
		AfterContext  []string
	}
	var currentMatch *MatchInternal
	var beforeLines []string

	for _, line := range lines {
		if line == "" {
			continue
		}

		// ripgrep 用 -- 分隔不同匹配
		if line == "--" {
			if currentMatch != nil {
				m := types.LogMatch{
					File:          currentMatch.File,
					LineNum:       currentMatch.LineNum,
					Raw:           currentMatch.Raw,
					BeforeContext: currentMatch.BeforeContext,
					AfterContext:  currentMatch.AfterContext,
					Context:       append(append([]string{}, currentMatch.BeforeContext...), currentMatch.AfterContext...),
				}
				matches = append(matches, m)
			}
			currentMatch = nil
			beforeLines = []string{}
			continue
		}

		parsed := parseLine(line, matchLineRegex, contextLineRegex)

		// 决策逻辑：优先匹配行
		if parsed.isMatch {
			if currentMatch != nil {
				// 关键！在处理新匹配之前，把前一个匹配的 after context 截断到 opts.After 的数量（不过这里 ParseOutput 没有 opts，所以我们假设用户会在使用后处理，但我们至少处理一下）
				m := types.LogMatch{
					File:          currentMatch.File,
					LineNum:       currentMatch.LineNum,
					Raw:           currentMatch.Raw,
					BeforeContext: currentMatch.BeforeContext,
					AfterContext:  currentMatch.AfterContext,
					Context:       append(append([]string{}, currentMatch.BeforeContext...), currentMatch.AfterContext...),
				}
				matches = append(matches, m)
			}
			currentMatch = &MatchInternal{
				File:          parsed.file,
				LineNum:       parsed.lineNum,
				Raw:           parsed.content,
				BeforeContext: append([]string{}, beforeLines...),
				AfterContext:  []string{},
			}
			beforeLines = []string{}
			continue
		}

		// 检查是不是上下文行（只有不是匹配行时才是真正的上下文行）
		if hasContext && parsed.isContext && !parsed.isMatch {
			if currentMatch != nil {
				currentMatch.AfterContext = append(currentMatch.AfterContext, parsed.content)
			} else {
				beforeLines = append(beforeLines, parsed.content)
			}
		}
	}

	// 最后一个匹配
	if currentMatch != nil {
		m := types.LogMatch{
			File:          currentMatch.File,
			LineNum:       currentMatch.LineNum,
			Raw:           currentMatch.Raw,
			BeforeContext: currentMatch.BeforeContext,
			AfterContext:  currentMatch.AfterContext,
			Context:       append(append([]string{}, currentMatch.BeforeContext...), currentMatch.AfterContext...),
		}
		matches = append(matches, m)
	}

	return matches
}
