# 变更日志

所有重要的版本更新都会记录在此文件中。

## [v0.6.4] - 2026-06-08

### 🎨 前端 UX 优化（日志结果显示区域）

**核心功能（LogList.vue 组件全面重构）

- **搜索摘要工具栏**
  - 顶部固定显示关键指标：匹配数、文件数、ERROR/WARN 数量、搜索耗时（ms）
  - 流式搜索时显示 `扫描中 X/Y` 绿色脉冲进度徽章

- **结果内搜索（Find in Results）
  - 工具栏输入框实时二次筛选，高亮匹配关键词
  - 键盘快捷键：`Ctrl+F` 聚焦输入框，`Ctrl+G` 下一个，`Shift+Ctrl+G` 上一个
  - 实时统计匹配数，`Esc` 清空
  - 点击匹配项自动滚动到对应日志行并高亮

- **上下文行切换**
  - 一键显示/隐藏日志上下文行（before/after）
  - 设置面板开关控制，默认开启
  - 关闭时仅显示匹配行，屏幕利用率大幅提升

- **JSON 格式化显示**
  - 设置面板开关控制，自动识别 JSON 内容
  - 开启后自动缩进美化结构化日志

- **文件分组可折叠**
  - 每个文件一个 header 分组（显示文件名 + 匹配数）
  - 点击 `▼/▶` 箭头展开/折叠该文件下所有日志

- **导出功能**
  - 支持 **TXT / JSON / CSV / Markdown** 四种格式
  - JSON 导出包含搜索元信息（pattern、total、duration_ms、timestamp、matches）
  - CSV 导出包含 file、line_num、raw 字段
  - Markdown 导出带标题+代码块格式，便于粘贴到工单/文档

- **复制功能**
  - `复制全部：一键复制所有搜索结果
  - `单条复制：鼠标悬停日志行时显示复制按钮，复制单条日志

- **长日志折叠**
  - 超过 **500 字符**自动折叠，显示 `...[展开]`
  - 点击展开完整内容，折叠按钮始终可见
  - 动态行高根据内容长度自动计算（36px + 每18px/行）

- **时间戳识别与跳转**
  - 自动提取常见日志时间格式（ISO 8601、Go/Java 标准）
  - 单独显示在日志行内，可点击跳转
  - 工具栏日期时间选择器（Element Plus）选择时间后滚动到最接近的日志
  - 快捷跳转按钮：`⏮ 首条` / `◀ -10分` / `+10分 ▶` / `末条 ⏭`

- **动态虚拟滚动**
  - 基于内容字符数动态计算每行高度（CHARS_PER_LINE=150，EXTRA_LINE_HEIGHT=18px）
  - ResizeObserver 监控容器尺寸，响应式滚动区域
  - 虚拟滚动 offsets 随展开/折叠状态实时更新，滚动位置无错位

- **搜索历史（SearchForm.vue）
  - localStorage 持久化，最近 10 条
  - **保存完整搜索条件**：关键词、级别、时间范围、路径、上下文行数、忽略大小写
  - 点击历史标签**一键恢复全部筛选参数
  - 级别徽章、时间范围图标、路径数量直观显示
  - 自动去重 + 时间排序 + 一键清空

- **XSS 安全防护**
  - 所有用户输入文本渲染前 HTML 转义
  - 关键词高亮在转义后应用

- **空状态三态区分**
  - 🔍 正在搜索：旋转动画 + 扫描进度
  - 📭 搜索过但无结果：提示调整筛选条件
  - 💡 尚未开始搜索：提示输入关键词

### 🔧 技术改进

- `go.mod`：Go 工具链版本升级 **1.25.9 → 1.25.11（go mod tidy / go build / go vet 全部通过）
- `web/src/composables/useLogStream.ts`：新增 `searchDurationMs` 搜索耗时统计、`progress` 进度状态
- `web/src/components/LogList.vue`：全面重构虚拟滚动逻辑，新增动态行高 `calcMatchRowHeight()`、`findMatches` 结果内搜索、`handleExport` 多格式导出、`copySingleLog`、时间跳转等
- `web/src/components/SearchForm.vue`：新增 `addToHistory()` 完整搜索条件记录
- `web/src/App.vue`：新增 `progress` prop 传递，清理未使用变量

## [v0.6.3] - 2026-06-07

### 📝 新增功能

- **全面支持 Go / Java 常见日志时间格式，自动转换为东八区（CST）**
  - 新增 `timefilter` 包级 `cst` 变量（`time.FixedZone("CST", 8*3600)`），所有无时区信息的时间字符串统一按 CST 解析
  - `extractTimestamp` 新增 9 个正则模式，覆盖 8 种常见日志格式（详见下方格式表）
  - `parseTime` 重构：优先解析带时区后缀的 ISO 8601 / RFC3339 格式（自动转 CST），其余无时区信息一律按 CST 解析
  - 新增单元测试 `timefilter_test.go`，覆盖所有支持格式的解析正确性验证（15 个测试用例，100% PASS）

- **支持的日志时间格式一览表**

  | # | 格式类型 | 示例 | 来源 |
  |---|---|---|---|
  | ① | `[YYYY-MM-DD HH:MM:SS.mmm]` | `[2026-06-07 00:35:31.984]` | 通用（Go/Java 均常见） |
  | ② | `YYYY-MM-DD HH:MM:SS` bare | `2026-06-07 00:35:31` | 通用 |
  | ③ | `YYYY-MM-DDTHH:MM:SS.mmm±ZZ:ZZ` | `2026-06-07T00:35:31.984+08:00` | ISO 8601 |
  | ④ | `YYYY/MM/DD HH:MM:SS` | `2026/06/07 01:15:40` | Go 常见 |
  | ⑤ | `[I] YYYY/MM/DD HH:MM:SS file.go:123:` | `[I] 2026/06/07 01:15:40 hybrid_storage.go:330:` | Go 标准 log |
  | ⑥ | `DD-MM-YYYY HH:MM:SS` | `07-06-2026 01:15:40.984` | Java Log4j/Logback 欧式 |
  | ⑦ | `{"time":"..."}` JSON | `{"time":"2026-06-07T01:15:40.984+08:00"}` | Go slog / zap / logrus |
  | ⑧ | `Mon DD, YYYY HH:MM:SS AM` | `Jun 07, 2026 1:15:40 AM` | Java Log4j 英文日期 |
  | ⑨ | `YYYY-MM-DD HH:MM:SS,mmm` 逗号毫秒 | `2026-06-07 01:15:40,984` | 欧洲格式 |

- **时区处理规则**
  - 带时区后缀（如 `Z`、`+08:00`、`-0500`）→ 自动转换为 CST（东八区）
  - 无时区后缀 → 一律按 CST（东八区）解析
  - 示例：`2026-06-07T01:15:31Z`（UTC）→ 过滤时按 `2026-06-07 09:15:31 CST` 处理

### 🐛 问题修复

- **修复时间范围筛选后搜索结果为空的多个 Bug**
  - `timefilter.parseTime` 增加毫秒时间戳支持（`"2006-01-02 15:04:05.999"`）
  - `timefilter.extractTimestamp` 新增 Go 标准日志格式支持（`[I] 2026/06/07 01:15:40 ...`）
  - `timefilter.parseTime` 新增 `"2006/01/02 15:04:05"` 和 `"2006/01/02 15:04"` 布局
  - `server.buildOptsFromRequest` 改用 `time.ParseInLocation` 替代 `time.Parse`，修复 UTC 与本地时区偏差（CST 下相差 8 小时）
  - `server.buildOptsFromRequest` 增加秒级格式兜底解析
  - `server.applyTimeFilter` 格式化时间时保留秒和毫秒精度（`"2006-01-02 15:04:05.000"`）
  - 前端 `SearchForm.vue`：`handleSearch` 增加 `timeRange` null 检查，修复清空时间范围后搜索静默失败
  - `server.SearchRequest.summary()` 增加 `from`/`to` 字段输出，便于调试

- **修复上下文行（before/after）未正常显示的问题**
  - **根因**：`contextLineRegex` 正则表达式错误，假设上下文行格式为 `filename-linenum-content`，实际 ripgrep 格式为 `linenum-content`（行号后面是 `-` 号）
  - **修复**：`parseLine` 函数重构，正确解析 ripgrep 上下文行格式
  - 新增 `currentFile` 变量跟踪当前文件名（多文件搜索时）
  - 正确处理文件名行（ripgrep 在多文件搜索时会先输出文件名）
  - 上下文行解析逻辑：`^(\d+)-(.*)$` 提取行号和内容简介

### 🔄 重构

- **搜索逻辑重构：改为逐个文件搜索，彻底解决文件名解析问题**
  - **问题**：之前从 ripgrep 输出中解析文件名，路径含 `:` 时（Windows 盘符、URL）解析不可靠，且单文件/多文件输出格式不一致容易出错
  - **方案**：Go 层先展开文件列表，逐个文件调用 ripgrep
  - **好处**：
    - 文件名 100% 准确（Go 层自己管理，不再从 rg 输出解析）
    - 解析逻辑极度简化：单文件 rg 输出格式固定为 `linenum:content`，不再需要复杂正则
    - 支持目录递归 + glob 过滤
    - 保留流式返回能力（搜完一个文件就回调）
  - **变更**：
    - 新增 `expandPaths`：展开路径列表（支持目录递归 + glob 过滤）
    - 新增 `searchAllFiles`：逐个文件调用 rg，流式回调
    - 新增 `searchSingleFile`：对单个文件执行 rg，解析输出
    - 新增 `rgArgsForFile`：构建单文件 rg 参数
    - 删除 `parseLine` 函数和复杂的文件名检测逻辑
    - 更新 `SearchStream` 签名（不再需要预构建的 rgArgs 参数）
    - 保留 `BuildArgs` 方法以保持向后兼容
  - **测试**：单文件/多文件搜索、before/after 上下文均正常

## [v0.6.1](https://gitee.com/lorock/miaokun-log/releases/v0.6.1) - 2026-06-06

### 🐛 问题修复

- **修复进入空目录时文件浏览模态框消失问题**
  - 后端 `files.go`：`var allFiles []FileInfo` 改为 `allFiles := make([]FileInfo, 0)`
  - 后端 `files.go`：`paginate` 函数保证返回非 nil 切片，JSON 序列化为 `[]` 而非 `null`
  - 前端 `FileBrowserModal.vue`：移除 `before-close` + `@close` 双重关闭绑定，避免空目录响应时误触发
  - 前端 `FileBrowserModal.vue`：`close-on-click-modal` 改为 `false`，避免点遮罩关闭
  - 前端 `useFileList.ts`：401/错误时不再清空 `files`，保留模态框打开状态

- **修复 `data: null` 导致前端 `files.length` 计算异常**
  - 所有返回切片的后端接口均使用 `make([]T, 0)` 初始化
  - 前端 `data.data || []` 防御式兜底，确保 `data` 为 null 时也能正常渲染

- 修复文件浏览路径导航时面包屑点击空路径跳转问题
- 修复 JWT Token 刷新流程：收到 401 先刷新 token，刷新失败后才登出

### 📝 文档更新

- 更新 README.md：完善 Web 功能说明、核心 API 列表、响应格式约定
- 更新 DEVELOPMENT.md：JWT Token 认证说明、空目录响应格式、401 刷新流程、中文错误提示说明

---

## [v0.6.0](https://gitee.com/lorock/miaokun-log/releases/v0.6.0) - 2026-06-06

### 🎉 新增功能

- **文件浏览功能**
  - 新增文件浏览模态框，支持目录导航
  - 默认显示根目录 `/`，允许浏览所有可访问目录
  - 敏感目录自动过滤（`/etc`, `/proc`, `/sys`, `/root` 等）
  - Root 用户可访问自己的 `/root` 目录
  - 真正的目录权限检查（区分可读/受限状态）
  - 支持分页、排序和文件名搜索

- **搜索结果优化**
  - 上下文行只来自关键词所在文件（修复跨文件污染问题）
  - 新增关键词高亮显示功能
  - 上下文行显示/隐藏开关（默认显示）
  - 新增"滚动到最新"悬浮按钮

- **认证系统**
  - 新增登录页面组件
  - 新增认证守卫组件
  - JWT Token 管理（自动刷新、永不过期检测）
  - 用户信息持久化（localStorage）

### 🎨 UI 优化

- 登录后顶部用户信息显示优化
- 搜索结果区域固定在顶部不随滚动
- 虚拟滚动优化（可见 300-500 行，提升用户体验）
- 自动跟随最新日志功能
- 内存防溢出设计（上限 50000 条）

### 🔧 功能改进

- 文件列表 API 默认路径改为根目录 `/`
- 日志搜索防内存溢出（批量更新 + 上限控制）
- 修复 `is_dir` 和 `file_type` 不一致问题

### 🐛 问题修复

- 修复文件浏览 API 400 错误（未传 path 参数）
- 修复刷新页面后自动跳转登录页（Token 过期时间单位问题）
- 修复上下文行跨文件显示问题（CLI 和 Web 端）
- 修复 `el-dialog` 的 `v-model` 绑定问题
- 修复目录类型判断不一致问题
- 修复 401 响应后端未正确踢出用户登录状态问题
- 修复错误提示为英文（改为中文提示）

### 📝 文档更新

- 更新 DEVELOPMENT.md 添加新增模块说明
- 添加认证系统相关文档

## [v0.5.1](https://gitee.com/lorock/miaokun-log/releases/v0.5.1) - 2026-06-05

### 🎨 UI优化
- 移除重复副标题，简化头部布局
- 优化上下文行数默认值：前3行，后5行（符合业内日志排查习惯）

### 🔧 功能改进
- 更新命令行版本号至 0.5.1

### 🐛 问题修复
- 修复 tsconfig.json 中 `ignoreDeprecations` 选项导致 vue-tsc 构建失败问题

### 📝 文档更新
- 更新 CHANGELOG.md 记录版本历史

## [v0.5.0](https://gitee.com/lorock/miaokun-log/releases/v0.5.0) - 2026-06-04

### 🎉 新增功能
- 新增 Web 界面，提供可视化日志搜索体验
  - 实时日志搜索（支持正则表达式）
  - 按日志级别过滤（ERROR / WARN / INFO / DEBUG / TRACE）
  - 时间范围过滤和多路径选择
  - TraceId 跨文件追踪
  - 日志级别选择后自动触发搜索
  - 默认开启忽略大小写搜索
- 新增 `--verbose/-v` 日志级别控制
  - 默认静默模式，仅在需要排查问题时开启
  - `-v`：显示 API 请求/响应日志（路径、状态码、耗时）
  - `-vv`：显示完整调试信息（请求参数、搜索中间结果、处理详情）

### 🎨 UI 优化
- 更新 Web 界面头部设计：品牌名+产品名组合（喵坤®日志排查工具）
- 优化配色方案，使用紫色系主色调，提升视觉一致性
- 优化 Logo 交互：点击放大、悬停显示品牌名提示
- 优化统计信息展示，整合到搜索卡片中，减少页面高度占用
- 修复页脚链接文字（GitHub → Gitee）

### 🔧 配置优化
- 修改默认端口配置，避免与常用端口冲突
  - 服务默认端口：`8080` → `9528`（后端 API + 前端静态资源统一端口）
- API 日志默认关闭，仅错误信息始终输出，保证生产环境运行效率

### 🐛 问题修复
- 修复 SSE 流式接口（`/api/v1/search/stream`、`/api/v1/trace`）无法推送的问题
  - 自定义 `responseWriter` 未实现 `http.Flusher` 接口，导致 `Flusher` 断言失败
  - 已为包装器添加 `Flush()` 方法，正确代理到底层 `ResponseWriter`

### 📝 文档更新
- README 新增 serve 命令使用说明和端口说明
- 更新 CHANGELOG.md 记录版本历史
- README 添加品牌 Logo 和知识产权信息（商标、版权）

## [v0.4.2](https://gitee.com/lorock/miaokun-log/releases/v0.4.2) - 2025-12-15

### 🐛 问题修复
- 修复 `--no-banner` 选项不生效的问题（将 banner 打印逻辑移至 `PersistentPreRun` 回调）
- 修复搜索无结果时无提示信息的问题（添加友好提示）

### 📝 文档优化
- 优化 README 文档结构，合并重复内容
- 创建 CHANGELOG.md 分离版本历史
- 更新工具定位，支持多种语言日志

## [v0.4.1](https://gitee.com/lorock/miaokun-log/releases/v0.4.1) - 2025-11-20

### 🆕 新增功能
- 新增 Go 语言日志格式支持：zap JSON、zerolog JSON、logrus 文本/JSON、slog JSON、标准库 log
- 更新工具定位，支持多种语言日志（Java / Go / Python / Node.js 等）

### 🐛 问题修复
- 修复 ripgrep 单文件搜索时不输出文件名的问题（添加 -H 参数强制显示文件名）

### 📝 文档更新
- 更新文档，添加适用场景和不适用场景说明
- 大幅增强帮助信息，提升用户体验：添加特性列表、快速开始、进阶用法、完整示例

## [v0.4.0](https://gitee.com/lorock/miaokun-log/releases/v0.4.0) - 2025-10-15

### 🚀 性能优化
- 实现流式处理架构，实时输出搜索结果
- 优化内存使用，支持处理 100G+ 大文件
- 增强稳定性，避免大文件搜索时的内存溢出

### 🔧 功能改进
- 改进时间过滤集成，支持流式处理中的实时过滤
- 完善文档和使用说明

## [v0.3.0](https://gitee.com/lorock/miaokun-log/releases/v0.3.0) - 2025-09-20

### 🆕 新增功能
- 新增 `--jq` 参数，内置 jq 查询支持（基于 gojq）
- 新增 `-B/--before` 和 `-A/--after` 选项，支持上下文显示

### 🔧 功能改进
- 改进 JSON 输出格式，优化可读性

## [v0.2.0](https://gitee.com/lorock/miaokun-log/releases/v0.2.0) - 2025-08-15

### 🆕 新增功能
- 新增 `stats` 命令，支持日志级别统计和文件分布分析
- 新增 `--level` 选项，按日志级别过滤搜索
- 新增 `-i/--ignore-case` 选项，支持大小写不敏感搜索
- 新增 `grep` 命令作为 search 的别名

### 🎨 用户体验
- 改进输出格式，增加彩色进度条

## [v0.1.0](https://gitee.com/lorock/miaokun-log/releases/v0.1.0) - 2025-07-01

### 🎉 初始版本
- 支持基础搜索、trace 关联、list 文件
