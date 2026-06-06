package searcher

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

// expandPaths 将 paths（可能包含目录）展开为具体文件列表，应用 glob 过滤
func expandPaths(paths []string, globs []string) ([]string, error) {
	var files []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue // 跳过不存在的路径
		}
		if !info.IsDir() {
			files = append(files, p)
			continue
		}
		// 递归遍历目录
		filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if len(globs) > 0 {
				matched := false
				for _, g := range globs {
					if m, err := filepath.Match(g, filepath.Base(path)); err == nil && m {
						matched = true
						break
					}
				}
				if !matched {
					return nil
				}
			}
			files = append(files, path)
			return nil
		})
	}
	return files, nil
}

// BuildArgs 构建 rg 命令行参数（供外部调用，保持向后兼容）
// 注意：新的 Search / SearchStream 内部已自行处理参数构建，
// 此方法主要用于测试或需要直接获取 rg 参数的场景。
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

// rgArgsForFile 构建单个文件的 rg 参数（不含文件路径）
func rgArgsForFile(opts types.SearchOptions) []string {
	args := []string{
		"-n",
		"--no-heading",
		"--color", "never",
		"--max-count", fmt.Sprintf("%d", opts.MaxCount),
	}
	if opts.CaseInsensitive {
		args = append(args, "-i")
	}
	if opts.Before > 0 {
		args = append(args, "-B", fmt.Sprintf("%d", opts.Before))
	}
	if opts.After > 0 {
		args = append(args, "-A", fmt.Sprintf("%d", opts.After))
	}
	args = append(args, opts.Pattern)
	return args
}

// Search 非流式搜索，返回全部结果
func (s *Searcher) Search(ctx context.Context, opts types.SearchOptions) ([]types.LogMatch, error) {
	if err := s.CheckRipgrep(); err != nil {
		return nil, err
	}
	var matches []types.LogMatch
	err := s.searchAllFiles(ctx, opts, func(m types.LogMatch) bool {
		matches = append(matches, m)
		return true
	})
	return matches, err
}

// SearchStream 流式搜索，逐个文件搜索，搜到即回调
func (s *Searcher) SearchStream(_ context.Context, opts types.SearchOptions, callback func(types.LogMatch) bool) error {
	if err := s.CheckRipgrep(); err != nil {
		return err
	}
	return s.searchAllFiles(context.Background(), opts, callback)
}

// searchAllFiles 展开文件列表，逐个文件调用 rg，流式回调
func (s *Searcher) searchAllFiles(ctx context.Context, opts types.SearchOptions, callback func(types.LogMatch) bool) error {
	files, err := expandPaths(opts.Paths, opts.Glob)
	if err != nil {
		return err
	}

	for _, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		matches, err := s.searchSingleFile(ctx, file, opts)
		if err != nil {
			continue // 单个文件失败跳过
		}
		for _, m := range matches {
			if opts.Level != "" && !strings.Contains(strings.ToUpper(m.Raw), strings.ToUpper(opts.Level)) {
				continue
			}
			if !callback(m) {
				return nil // 调用方要求停止
			}
		}
	}
	return nil
}

// searchSingleFile 对单个文件执行 rg，解析输出并返回匹配列表
func (s *Searcher) searchSingleFile(ctx context.Context, file string, opts types.SearchOptions) ([]types.LogMatch, error) {
	args := rgArgsForFile(opts)
	args = append(args, file)

	cmd := exec.CommandContext(ctx, s.RGPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("创建 stdout pipe 失败: %w", err)
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动 rg 失败: %w", err)
	}

	matches := parseSingleFileStream(stdout, file, opts)

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return matches, nil // 无匹配不是错误
		}
		return matches, nil // 已有部分结果，不中断
	}

	return matches, nil
}

// parseSingleFileStream 从 stdout 流式解析单个文件的 rg 输出
// 单文件 rg 输出格式（无 -H）：
//   匹配行：linenum:content
//   上下文行（before/after）：linenum-content  （注意是 - 不是 :）
//   不同匹配之间用 -- 分隔
func parseSingleFileStream(r io.Reader, file string, opts types.SearchOptions) []types.LogMatch {
	var matches []types.LogMatch

	matchRegex := regexp.MustCompile(`^(\d+):(.*)$`)
	ctxRegex := regexp.MustCompile(`^(\d+)-(.*)$`)

	type matchInternal struct {
		lineNum       int
		raw           string
		beforeCtx     []string
		afterCtx      []string
	}

	var current *matchInternal
	var beforeLines []string

	flush := func() {
		if current == nil {
			return
		}
		// 截断 after context 到 opts.After
		after := current.afterCtx
		if opts.After > 0 && len(after) > opts.After {
			after = after[:opts.After]
		}
		// 截断 before context 到 opts.Before
		before := current.beforeCtx
		if opts.Before > 0 && len(before) > opts.Before {
			before = before[len(before)-opts.Before:]
		}
		m := types.LogMatch{
			File:          file,
			LineNum:       current.lineNum,
			Raw:           current.raw,
			BeforeContext: before,
			AfterContext:  after,
			Context:       append(append([]string{}, before...), after...),
		}
		matches = append(matches, m)
		current = nil
		beforeLines = []string{}
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if line == "--" {
			flush()
			continue
		}

		// 尝试匹配行
		if groups := matchRegex.FindStringSubmatch(line); len(groups) == 3 {
			flush()
			lineNum, _ := strconv.Atoi(groups[1])
			current = &matchInternal{
				lineNum:   lineNum,
				raw:        groups[2],
				beforeCtx:  append([]string{}, beforeLines...),
				afterCtx:   []string{},
			}
			beforeLines = []string{}
			continue
		}

		// 尝试上下文行
		if groups := ctxRegex.FindStringSubmatch(line); len(groups) == 3 {
			if current != nil {
				current.afterCtx = append(current.afterCtx, groups[2])
			} else {
				beforeLines = append(beforeLines, groups[2])
				// 只保留最后 N 行作为 before context
				if opts.Before > 0 && len(beforeLines) > opts.Before {
					beforeLines = beforeLines[len(beforeLines)-opts.Before:]
				}
			}
		}
	}

	flush()
	return matches
}

// ParseOutput 解析 rg 输出字符串（用于测试或非流式场景）
// output 是单文件 rg 输出（不含文件名）
func ParseOutput(output string, hasContext bool) []types.LogMatch {
	// 伪造一个 opts，Only Before/After 从 hasContext 无法得知，默认 0
	// 实际使用时 SearchOptions 会传进来，这个函数主要用于测试
	opts := types.SearchOptions{}
	return parseOutputFromString("", output, opts)
}

// parseOutputFromString 从字符串解析（非流式，用于测试）
func parseOutputFromString(file, output string, opts types.SearchOptions) []types.LogMatch {
	return parseLines(strings.Split(output, "\n"), file, opts)
}

// parseLines 解析行列表（供测试用）
func parseLines(lines []string, file string, opts types.SearchOptions) []types.LogMatch {
	matchRegex := regexp.MustCompile(`^(\d+):(.*)$`)
	ctxRegex := regexp.MustCompile(`^(\d+)-(.*)$`)

	type matchInternal struct {
		lineNum   int
		raw       string
		beforeCtx []string
		afterCtx  []string
	}

	var matches []types.LogMatch
	var current *matchInternal
	var beforeLines []string

	flush := func() {
		if current == nil {
			return
		}
		after := current.afterCtx
		if opts.After > 0 && len(after) > opts.After {
			after = after[:opts.After]
		}
		before := current.beforeCtx
		if opts.Before > 0 && len(before) > opts.Before {
			before = before[len(before)-opts.Before:]
		}
		m := types.LogMatch{
			File:          file,
			LineNum:       current.lineNum,
			Raw:           current.raw,
			BeforeContext: before,
			AfterContext:  after,
			Context:       append(append([]string{}, before...), after...),
		}
		matches = append(matches, m)
		current = nil
		beforeLines = []string{}
	}

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		if line == "--" {
			flush()
			continue
		}
		if groups := matchRegex.FindStringSubmatch(line); len(groups) == 3 {
			flush()
			lineNum, _ := strconv.Atoi(groups[1])
			current = &matchInternal{
				lineNum:  lineNum,
				raw:       groups[2],
				beforeCtx: append([]string{}, beforeLines...),
				afterCtx:  []string{},
			}
			beforeLines = []string{}
			continue
		}
		if groups := ctxRegex.FindStringSubmatch(line); len(groups) == 3 {
			if current != nil {
				current.afterCtx = append(current.afterCtx, groups[2])
			} else {
				beforeLines = append(beforeLines, groups[2])
				if opts.Before > 0 && len(beforeLines) > opts.Before {
					beforeLines = beforeLines[len(beforeLines)-opts.Before:]
				}
			}
		}
	}

	flush()
	return matches
}
