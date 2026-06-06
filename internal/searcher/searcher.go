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
	matchLineRegex := regexp.MustCompile(`^(.+):(\d+):(.*)$`)
	// ripgrep 上下文行格式：行号-内容（注意是 - 不是 :）
	contextLineRegex := regexp.MustCompile(`^(\d+)-(.*)$`)

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
	var currentFile string // 跟踪当前文件名（多文件搜索时）

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

		// 检查是否是文件名行（多文件搜索时，ripgrep 会先输出文件名）
		if !strings.Contains(line, ":") && !strings.Contains(line, "-") && line != "--" {
			currentFile = line
			continue
		}

		isMatch, isContext, file, lineNum, content := parseLine(line, matchLineRegex, contextLineRegex, currentFile)

		if isMatch {
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
				File:          file,
				LineNum:       lineNum,
				Raw:           content,
				BeforeContext: append([]string{}, beforeLines...),
				AfterContext:  []string{},
			}
			// 关键！在设置新的 currentMatch 后，我们要把 beforeLines 重置
			beforeLines = []string{}
			continue
		}

		if hasContext && isContext {
			if currentMatch != nil {
				// 这是 after context
				currentMatch.AfterContext = append(currentMatch.AfterContext, content)
			} else {
				// 这是 before context，保留最后 N 行
				beforeLines = append(beforeLines, content)
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

func parseLine(line string, matchLineRegex, contextLineRegex *regexp.Regexp, currentFile string) (bool, bool, string, int, string) {
	// 尝试匹配行：filename:linenum:content
	// 注意：filename 可能包含 :，所以从右边找 :linenum: 模式
	if matchGroups := matchLineRegex.FindStringSubmatch(line); len(matchGroups) == 4 {
		if lineNum, err := strconv.Atoi(matchGroups[2]); err == nil && lineNum > 0 {
			return true, false, matchGroups[1], lineNum, matchGroups[3]
		}
	}

	// 尝试上下文行：linenum-content（ripgrep 上下文格式）
	if ctxGroups := contextLineRegex.FindStringSubmatch(line); len(ctxGroups) == 3 {
		if lineNum, err := strconv.Atoi(ctxGroups[1]); err == nil && lineNum > 0 {
			return false, true, currentFile, lineNum, ctxGroups[2]
		}
	}

	return false, false, "", 0, ""
}

func ParseOutput(output string, hasContext bool) []types.LogMatch {
	var matches []types.LogMatch
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// 正则表达式：匹配行格式 file:line:content
	matchLineRegex := regexp.MustCompile(`^(.+):(\d+):(.*)$`)
	// 正则表达式：上下文行格式 line-content（ripgrep 格式）
	contextLineRegex := regexp.MustCompile(`^(\d+)-(.*)$`)

	type MatchInternal struct {
		File          string
		LineNum       int
		Raw           string
		BeforeContext []string
		AfterContext  []string
	}
	var currentMatch *MatchInternal
	var beforeLines []string
	var currentFile string

	for _, line := range lines {
		if line == "" {
			continue
		}

		// 检查是否是文件名行
		if !strings.Contains(line, ":") && !strings.Contains(line, "-") && line != "--" {
			currentFile = line
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

		isMatch, isContext, file, lineNum, content := parseLine(line, matchLineRegex, contextLineRegex, currentFile)

		// 决策逻辑：优先匹配行
		if isMatch {
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
			currentMatch = &MatchInternal{
				File:          file,
				LineNum:       lineNum,
				Raw:           content,
				BeforeContext: append([]string{}, beforeLines...),
				AfterContext:  []string{},
			}
			beforeLines = []string{}
			continue
		}

		// 检查是不是上下文行
		if hasContext && isContext {
			if currentMatch != nil {
				currentMatch.AfterContext = append(currentMatch.AfterContext, content)
			} else {
				beforeLines = append(beforeLines, content)
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
