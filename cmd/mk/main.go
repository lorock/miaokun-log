package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"gitee.com/lorock/miaokun-log/internal/auth"
	"gitee.com/lorock/miaokun-log/internal/config"
	"gitee.com/lorock/miaokun-log/internal/discover"
	"gitee.com/lorock/miaokun-log/internal/output"
	"gitee.com/lorock/miaokun-log/internal/searcher"
	"gitee.com/lorock/miaokun-log/internal/server"
	"gitee.com/lorock/miaokun-log/internal/timefilter"
	"gitee.com/lorock/miaokun-log/internal/trace"
	"gitee.com/lorock/miaokun-log/pkg/types"
	"gitee.com/lorock/miaokun-log/pkg/version"
	"github.com/spf13/cobra"
)

const (
	appName   = "喵坤"
	appSlug   = "mk"
	appSlogan = "日志搜索利器"
)

var (
	cfgFile         string
	sinceDays       float64
	fromStr         string
	toStr           string
	traceMode       bool
	statsMode       bool
	jsonOut         bool
	jqQuery         string
	noBanner        bool
	noColor         bool
	countOnly       bool
	before          int
	after           int
	caseInsensitive bool
	level           string
	servePort       string
	webDir          string
	serveVerbose    int
	authEnabled     bool
	authJWTSecret   string
	adminPassword   string
)

func printBanner() {
	fmt.Printf("\n")
	fmt.Printf("  🐾 %s · %s  v%s\n", appName, appSlogan, version.Version)
	fmt.Printf("  ─────────────────────────────────────\n")
	fmt.Printf("\n")
}

func main() {
	if strings.HasSuffix(os.Args[0], "miaokun") || strings.HasSuffix(os.Args[0], "mklog") {
		os.Args[0] = "mk"
	}

	rootCmd := &cobra.Command{
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if os.Getenv("MK_NO_BANNER") == "" && !noBanner {
				printBanner()
			}
		},
		Use:   appSlug,
		Short: "🐾 喵坤 (MiaoKun) - 多目录日志搜索工具，支持 Java 和 Go 日志格式",
		Long: `  🐾 喵坤 (MiaoKun) 是一个高效的多目录日志搜索工具。
  它结合了 ripgrep 的速度、智能缓存和 traceId 关联能力，支持 Java 和 Go 主流日志格式，
  让多目录日志排查变得简单高效。

  常用命令:
    mk search <pattern> [paths...]  - 搜索日志
    mk trace <traceId> [paths...]   - 按 traceId 追踪
    mk list [paths...]              - 列出日志文件
    mk stats [paths...]             - 显示日志统计

  特性:
    • 🚀 基于 ripgrep，极速搜索
    • 💨 流式处理，实时输出结果
    • 📦 自动解压并缓存 .gz 文件
    • 🔗 traceId 跨文件关联追踪
    • ⏰ 时间窗口过滤
    • 📊 日志统计和分析
    • 🎨 彩色输出和友好界面
    • 📄 JSON 格式 + 内置 jq 查询
    • 💾 内存优化，超大文件支持

  快速开始:
    1️⃣  mk list                  - 查看可搜索的日志文件
    2️⃣  mk stats                 - 查看日志统计信息
    3️⃣  mk search "ERROR"        - 搜索 ERROR 级别日志
    4️⃣  mk trace <traceId>       - 追踪指定的 traceId

  进阶用法:
    mk search "ERROR" -B 2 -A 5      - 显示匹配行前后各 2 行
    mk search "ERROR" --from "2026-05-31 02:00:00
    mk search "ERROR" --json --jq ".[].file"`,
		SilenceErrors: true,
		SilenceUsage:  true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: false,
		},
	}

	rootCmd.Version = version.Version
	rootCmd.SetVersionTemplate(fmt.Sprintf("🐾 喵坤 v%s  %s/%s\n", version.Version, runtime.GOOS, runtime.GOARCH))

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "配置文件 (默认: $HOME/.miaokun.yaml)")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "JSON 格式输出")
	rootCmd.PersistentFlags().StringVar(&jqQuery, "jq", "", "jq 查询表达式（配合 --json 使用）")
	rootCmd.PersistentFlags().BoolVar(&noBanner, "no-banner", false, "不显示 banner")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "禁用彩色输出")
	rootCmd.PersistentFlags().Lookup("no-color").NoOptDefVal = "true"

	searchCmd := &cobra.Command{
		Use:   "search <pattern> [paths...]",
		Short: "🔍 搜索日志（支持 .gz 缓存 + trace 关联）",
		Long: `  在日志文件中搜索匹配的内容。
  支持 Java 和 Go 主流日志格式。

  默认搜索路径: /var/log, /opt/logs

  基本用法:
    mk search "ERROR"                                - 搜索 ERROR
    mk search "NullPointerException" /var/log/app      - 指定路径搜索

  过滤选项:
    -i, --ignore-case    - 大小写不敏感搜索
        --level string     - 按日志级别过滤 (ERROR, WARN, INFO, DEBUG, TRACE)
    -s, --since int      - 只扫描最近 N 天的文件 (默认 3)
        --from string    - 开始时间 (YYYY-MM-DD HH:MM)
        --to string      - 结束时间 (YYYY-MM-DD HH:MM)

  上下文显示:
    -B, --before int     - 显示匹配行前 N 行
    -A, --after int      - 显示匹配行后 N 行

  输出选项:
        --trace          - 启用 traceId 关联显示
        --stats          - 显示统计信息
        --count          - 只显示数量

  进阶用法:
    mk search "ERROR" -i --level ERROR               - 忽略大小写
    mk search "WARN" --from "2026-05-31 02:00
    mk search "ERROR" -B 3 -A 5                     - 显示前后 3/5 行
    mk search "ERROR" --json --jq ".[].file"              - JSON + jq 查询

  示例:
    mk search "ERROR"
    mk search "NullPointerException" /var/log/app
    mk search "WARN" --since 1 --trace
    mk search "error" -i --level ERROR`,
		Args: cobra.MinimumNArgs(1),
		RunE: runSearch,
	}
	searchCmd.Flags().Float64VarP(&sinceDays, "since", "s", 3, "只扫描最近 N 天的文件（支持小数，如 0.02 = 30分钟, 0.25 = 6小时）")
	searchCmd.Flags().StringVar(&fromStr, "from", "", "开始时间 (YYYY-MM-DD HH:MM)")
	searchCmd.Flags().StringVar(&toStr, "to", "", "结束时间 (YYYY-MM-DD HH:MM)")
	searchCmd.Flags().BoolVar(&traceMode, "trace", false, "启用 traceId 关联显示")
	searchCmd.Flags().BoolVar(&statsMode, "stats", false, "显示统计信息")
	searchCmd.Flags().BoolVar(&countOnly, "count", false, "只显示匹配数量")
	searchCmd.Flags().IntVarP(&before, "before", "B", 0, "显示匹配行前 N 行")
	searchCmd.Flags().IntVarP(&after, "after", "A", 0, "显示匹配行后 N 行")
	searchCmd.Flags().BoolVarP(&caseInsensitive, "ignore-case", "i", false, "大小写不敏感搜索")
	searchCmd.Flags().StringVar(&level, "level", "", "按日志级别过滤 (ERROR, WARN, INFO, DEBUG, TRACE)")

	traceCmd := &cobra.Command{
		Use:   "trace <traceId> [paths...]",
		Short: "🔗 按 traceId 跨文件追踪",
		Long: `  跨多个日志文件追踪指定的 traceId。
  自动识别包含 traceId 的日志行并按时间排序显示。

  默认搜索路径: /var/log, /opt/logs

  基本用法:
    mk trace <traceId>                                - 追踪 traceId
    mk trace <traceId> /var/log/app                   - 指定路径追踪

  选项:
    -s, --since int     - 只扫描最近 N 天的文件 (默认 3)
    -B, --before int    - 显示匹配行前 N 行
    -A, --after int     - 显示匹配行后 N 行
    -i, --ignore-case   - 大小写不敏感搜索

  示例:
    mk trace abc123def456
    mk trace 7a8b9c0d1e2f3a4b5c6d /var/log/app
    mk trace abc123 -i -B 2 -A 3`,
		Args: cobra.MinimumNArgs(1),
		RunE: runTrace,
	}
	traceCmd.Flags().Float64VarP(&sinceDays, "since", "s", 3, "只扫描最近 N 天的文件（支持小数，如 0.02 = 30分钟, 0.25 = 6小时）")
	traceCmd.Flags().IntVarP(&before, "before", "B", 0, "显示匹配行前 N 行")
	traceCmd.Flags().IntVarP(&after, "after", "A", 0, "显示匹配行后 N 行")
	traceCmd.Flags().BoolVarP(&caseInsensitive, "ignore-case", "i", false, "大小写不敏感搜索")

	listCmd := &cobra.Command{
		Use:   "list [paths...]",
		Short: "📁 列出找到的日志文件",
		Long: `  列出指定目录下的所有日志文件。
  支持识别常见的日志文件扩展名。

  默认搜索路径: /var/log, /opt/logs

  选项:
    -s, --since int     - 只显示最近 N 天的文件 (默认 30)

  示例:
    mk list
    mk list /var/log/app
    mk list --since 7

  说明:
    • 显示文件名、大小、修改时间
    • 支持 .log, .log.gz, .log.*.gz 等格式
    • 默认显示最近 30 天的文件`,
		Args: cobra.MinimumNArgs(0),
		RunE: runList,
	}
	listCmd.Flags().Float64VarP(&sinceDays, "since", "s", 30, "只显示最近 N 天的文件（支持小数）")

	statsCmd := &cobra.Command{
		Use:   "stats [paths...]",
		Short: "📊 显示日志统计信息",
		Long: `  统计指定目录下的日志信息。
  提供日志级别分布和文件匹配统计。

  默认搜索路径: /var/log, /opt/logs

  选项:
    -s, --since int     - 只扫描最近 N 天的文件 (默认 30)

  统计内容:
    • 日志级别分布 (ERROR, WARN, INFO, DEBUG, TRACE)
    • 文件匹配统计
    • 可视化柱状图展示

  示例:
    mk stats
    mk stats /var/log/app
    mk stats --since 7`,
		Args: cobra.MinimumNArgs(0),
		RunE: runStats,
	}
	statsCmd.Flags().Float64VarP(&sinceDays, "since", "s", 30, "只扫描最近 N 天的文件（支持小数）")

	grepCmd := &cobra.Command{
		Use:   "grep <pattern> [paths...]",
		Short: "🔍 搜索日志 (search 的别名)",
		Long: `  grep 是 search 的别名，功能完全一致。
  方便习惯使用 grep 的用户。

  基本用法:
    mk grep "ERROR"
    mk grep "NullPointerException" /var/log/app

  请使用 mk search --help 查看完整选项列表。`,
		Args: cobra.MinimumNArgs(1),
		RunE: runSearch,
	}
	grepCmd.Flags().Float64VarP(&sinceDays, "since", "s", 3, "只扫描最近 N 天的文件（支持小数，如 0.02 = 30分钟, 0.25 = 6小时）")
	grepCmd.Flags().StringVar(&fromStr, "from", "", "开始时间 (YYYY-MM-DD HH:MM)")
	grepCmd.Flags().StringVar(&toStr, "to", "", "结束时间 (YYYY-MM-DD HH:MM)")
	grepCmd.Flags().BoolVar(&traceMode, "trace", false, "启用 traceId 关联显示")
	grepCmd.Flags().BoolVar(&statsMode, "stats", false, "显示统计信息")
	grepCmd.Flags().BoolVar(&countOnly, "count", false, "只显示匹配数量")
	grepCmd.Flags().IntVarP(&before, "before", "B", 0, "显示匹配行前 N 行")
	grepCmd.Flags().IntVarP(&after, "after", "A", 0, "显示匹配行后 N 行")
	grepCmd.Flags().BoolVarP(&caseInsensitive, "ignore-case", "i", false, "大小写不敏感搜索")
	grepCmd.Flags().StringVar(&level, "level", "", "按日志级别过滤 (ERROR, WARN, INFO, DEBUG, TRACE)")

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "🐾 启动 Web 版日志搜索服务",
		Long: `  启动 HTTP 服务器，提供 Web 界面和 REST API。
  
  默认端口: 9528
  默认前端目录: ./web/dist
  
  用法:
    mk serve
    mk serve --port 9528 --web-dir ./web/dist
  
  API 端点:
    GET  /api/v1/health          - 健康检查
    GET  /api/v1/files           - 获取日志文件列表
    POST /api/v1/search          - 普通搜索
    POST /api/v1/search/stream   - SSE 流式搜索（推荐）
    POST /api/v1/stats           - 日志统计
    POST /api/v1/trace           - traceId 追踪`,
		RunE: runServe,
	}
	serveCmd.Flags().StringVar(&servePort, "port", "9528", "服务端口")
	serveCmd.Flags().StringVar(&webDir, "web-dir", "./web/dist", "前端静态资源目录")
	serveCmd.Flags().CountVarP(&serveVerbose, "verbose", "v", "增加日志详细程度 (-v 显示请求/响应, -vv 显示调试详情)")
	serveCmd.Flags().BoolVar(&authEnabled, "auth", true, "启用认证系统")
	serveCmd.Flags().StringVar(&authJWTSecret, "jwt-secret", "", "JWT密钥 (至少32字符，启用认证时推荐显式配置)")
	serveCmd.Flags().StringVar(&adminPassword, "admin-password", "", "管理员密码 (未提供时自动生成随机密码)")

	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(traceCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(grepCmd)
	rootCmd.AddCommand(serveCmd)

	if err := rootCmd.Execute(); err != nil {
		if noColor || os.Getenv("NO_COLOR") != "" {
			fmt.Printf("错误: %s\n", err)
		} else {
			fmt.Printf("\033[1;31m错误: %s\033[0m\n", err)
		}
		fmt.Printf("\n使用 \"mk --help\" 查看帮助信息\n")
		os.Exit(1)
	}
}

func runSearch(cmd *cobra.Command, args []string) error {
	s := searcher.New()
	if err := s.CheckRipgrep(); err != nil {
		return err
	}

	pattern := args[0]
	paths := []string{"/var/log", "/opt/logs"}
	if len(args) > 1 {
		paths = args[1:]
	}

	if err := config.Load(cfgFile); err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	tf := timefilter.New()
	if fromStr != "" {
		if err := tf.SetFrom(fromStr); err != nil {
			return err
		}
	}
	if toStr != "" {
		if err := tf.SetTo(toStr); err != nil {
			return err
		}
	}

	files, err := discover.FindLogs(paths, sinceDays)
	if err != nil {
		return fmt.Errorf("发现日志文件失败: %w", err)
	}
	if len(files) == 0 {
		fmt.Println("  (ﾉﾟ0ﾟ)ﾉ~ 喵坤挠了个空，没找到日志文件")
		fmt.Println("  提示: 尝试 \"mk list\" 查看可用的日志文件")
		return nil
	}

	logPaths := make([]string, len(files))
	for i, f := range files {
		logPaths[i] = f.Path
	}

	opts := types.SearchOptions{
		Pattern:         pattern,
		Paths:           logPaths,
		Glob:            []string{"*.log", "*.log.gz", "*.log.*.gz"},
		MaxCount:        10000,
		CaseInsensitive: caseInsensitive,
		Level:           level,
		Before:          before,
		After:           after,
	}

	out := output.New(jsonOut, jqQuery, noColor)

	if countOnly {
		matches, err := s.Search(cmd.Context(), opts)
		if err != nil {
			return fmt.Errorf("搜索失败: %w", err)
		}
		filtered := tf.Filter(matches)
		fmt.Printf("  🎯 共找到 %d 条匹配\n", len(filtered))
		return nil
	}

	if traceMode || statsMode || before > 0 || after > 0 || jsonOut {
		matches, err := s.Search(cmd.Context(), opts)
		if err != nil {
			return fmt.Errorf("搜索失败: %w", err)
		}
		filtered := tf.Filter(matches)
		if len(filtered) == 0 {
			fmt.Println("  (⊙ˍ⊙) 喵坤没找到匹配的日志，试试其他关键词？")
			return nil
		}
		if traceMode {
			correlated := trace.Correlate(filtered)
			out.PrintTraceCorrelation(correlated)
		} else {
			out.PrintMatches(filtered)
		}
		if statsMode {
			out.PrintStats(filtered)
		}
		return nil
	}

	err = s.SearchStream(cmd.Context(), opts, func(m types.LogMatch) bool {
		out.PrintMatch(m)
		return true
	})

	if err != nil {
		return fmt.Errorf("搜索失败: %w", err)
	}

	return nil
}

func runTrace(cmd *cobra.Command, args []string) error {
	s := searcher.New()
	if err := s.CheckRipgrep(); err != nil {
		return err
	}

	traceId := args[0]
	paths := []string{"/var/log", "/opt/logs"}
	if len(args) > 1 {
		paths = args[1:]
	}

	if err := config.Load(cfgFile); err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	files, err := discover.FindLogs(paths, sinceDays)
	if err != nil {
		return fmt.Errorf("发现日志文件失败: %w", err)
	}
	if len(files) == 0 {
		fmt.Println("  (ﾉﾟ0ﾟ)ﾉ~ 喵坤挠了个空，没找到日志文件")
		fmt.Println("  提示: 尝试 \"mk list\" 查看可用的日志文件")
		return nil
	}

	logPaths := make([]string, len(files))
	for i, f := range files {
		logPaths[i] = f.Path
	}

	opts := types.SearchOptions{
		Pattern:         traceId,
		Paths:           logPaths,
		Glob:            []string{"*.log", "*.log.gz", "*.log.*.gz"},
		MaxCount:        10000,
		CaseInsensitive: caseInsensitive,
		Before:          before,
		After:           after,
	}

	matches, err := s.Search(cmd.Context(), opts)
	if err != nil {
		return fmt.Errorf("搜索失败: %w", err)
	}

	out := output.New(jsonOut, jqQuery, noColor)
	correlated := trace.Correlate(matches)
	out.PrintTraceCorrelation(correlated)

	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	paths := []string{"/var/log", "/opt/logs"}
	if len(args) > 0 {
		paths = args
	}

	if err := config.Load(cfgFile); err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	files, err := discover.FindLogs(paths, sinceDays)
	if err != nil {
		return fmt.Errorf("发现日志文件失败: %w", err)
	}

	out := output.New(jsonOut, jqQuery, noColor)
	if jsonOut {
		out.PrintJSON(files)
		return nil
	}

	if len(files) == 0 {
		fmt.Println("  (ﾉﾟ0ﾟ)ﾉ~ 喵坤挠了个空，没找到日志文件")
		return nil
	}

	for _, f := range files {
		sizeStr := ""
		if f.Size < 1024 {
			sizeStr = fmt.Sprintf("%d B", f.Size)
		} else if f.Size < 1024*1024 {
			sizeStr = fmt.Sprintf("%.1f KB", float64(f.Size)/1024)
		} else if f.Size < 1024*1024*1024 {
			sizeStr = fmt.Sprintf("%.1f MB", float64(f.Size)/(1024*1024))
		} else {
			sizeStr = fmt.Sprintf("%.1f GB", float64(f.Size)/(1024*1024*1024))
		}
		modTimeStr := f.ModTime.Format("2006-01-02 15:04:05")
		fmt.Printf("  %s  %s  %s\n", sizeStr, modTimeStr, f.Path)
	}
	fmt.Printf("\n  共找到 %d 个日志文件\n", len(files))

	return nil
}

func runStats(cmd *cobra.Command, args []string) error {
	s := searcher.New()
	if err := s.CheckRipgrep(); err != nil {
		return err
	}

	paths := []string{"/var/log", "/opt/logs"}
	if len(args) > 0 {
		paths = args
	}

	if err := config.Load(cfgFile); err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	files, err := discover.FindLogs(paths, sinceDays)
	if err != nil {
		return fmt.Errorf("发现日志文件失败: %w", err)
	}
	if len(files) == 0 {
		fmt.Println("  (ﾉﾟ0ﾟ)ﾉ~ 喵坤挠了个空，没找到日志文件")
		return nil
	}

	logPaths := make([]string, len(files))
	for i, f := range files {
		logPaths[i] = f.Path
	}

	opts := types.SearchOptions{
		Pattern:         "ERROR|WARN|INFO|DEBUG|TRACE",
		Paths:           logPaths,
		Glob:            []string{"*.log", "*.log.gz", "*.log.*.gz"},
		MaxCount:        1000000,
		CaseInsensitive: true,
	}

	matches, err := s.Search(cmd.Context(), opts)
	if err != nil {
		return fmt.Errorf("搜索失败: %w", err)
	}

	out := output.New(jsonOut, jqQuery, noColor)
	out.PrintStats(matches)

	return nil
}

func runServe(cmd *cobra.Command, args []string) error {
	// Issue3: Validate configuration when auth is enabled
	if authEnabled && authJWTSecret != "" && len(authJWTSecret) < 32 {
		return fmt.Errorf("--jwt-secret 长度不足 32 字符，HMAC-SHA256 需要足够强度的密钥")
	}

	authCfg := &auth.AuthConfig{
		Enabled:         authEnabled,
		JWTSecret:       authJWTSecret,
		DefaultPassword: adminPassword,
	}
	s := server.New(servePort, webDir, cfgFile, serveVerbose, authCfg)
	return s.Start()
}
