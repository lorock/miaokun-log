<p align="center">
  <img src="web/assets/logo.png" alt="喵坤® Logo" width="120" height="120">
</p>

# 🐾 喵坤<sup>®</sup> (MiaoKun)

**喵坤<sup>®</sup>在手，效率全有**  
为开发者与运维人打造的轻量生产力工具品牌，专注解决技术人日常工作中的高频痛点。坚持「单文件、零依赖、开箱即用」的设计理念，拒绝复杂部署与冗余功能，让工具回归实用本质。

**首款产品：喵坤<sup>®</sup>日志排查工具**  
轻量级、高性能日志检索与故障定位工具，支持 Java/Go/Python/Node.js 等多语言日志格式，命令行+Web 双端可用，百G大文件秒级响应，让线上排障快人一步。

---

## 品牌理念

- **轻量至上**：单二进制文件，无运行时依赖
- **性能优先**：高性能引擎，极致响应速度与资源效率
- **开箱即用**：默认配置覆盖绝大多数场景
- **持续进化**：围绕技术人工作流，打造一站式生产力工具箱

**产品路线**  
- ✅ 日志排查工具（已发布）
- 🔄 配置文件对比与格式化工具
- 🔄 服务器状态一键巡检工具
- 🔄 接口自动化测试轻量工具

---

## 核心优势

| 特性 | 价值 |
|------|------|
| 🚀 **极速搜索** | 基于 ripgrep 引擎，比传统 grep 快 5-10 倍，百G文件秒级响应 |
| 💨 **流式处理** | 实时输出，内存零溢出，轻松处理 100G+ 超大日志 |
| 🎨 **双端交互** | 命令行高效操作 + Web 可视化界面 |
| 📦 **零依赖部署** | 单二进制交付，前端资源内嵌 |
| 🧠 **智能缓存** | 自动解压并缓存 .gz 压缩日志 |
| 🔗 **全链路追踪** | 自动提取 traceId，跨文件追踪完整调用链 |
| ⏰ **精准过滤** | 时间窗口、日志级别、正则表达式多维度筛选 |
| 🔧 **灵活扩展** | 模块化架构，内置 jq 解析器 |

---

## 适用场景

**适合人群**  
- 后端开发者：快速定位代码异常与接口报错
- DevOps/SRE 工程师：生产环境日志排查与故障溯源
- 测试工程师：自动化测试日志分析与问题复现
- 所有需要频繁处理日志的技术人员

**不适用场景**  
- 需要复杂日志可视化分析（推荐 ELK/Graylog）
- 实时告警触发与监控（推荐 Prometheus/Grafana）
- PB 级海量日志聚合分析（推荐专业大数据平台）

---

## 快速开始

### 前置依赖

```bash
# macOS
brew install ripgrep

# Ubuntu/Debian
sudo apt install ripgrep

# CentOS/RHEL
sudo yum install ripgrep

# 其他系统：https://github.com/BurntSushi/ripgrep#installation
```

### 安装方式

```bash
# 方式1：源码安装
git clone https://gitee.com/lorock/miaokun-log.git
cd miaokun-log
make install

# 方式2：脚本安装
./scripts/install.sh
```

### 30秒上手

```bash
# 1. 列出默认路径下的日志文件
mk list

# 2. 搜索最近1天的 ERROR 日志
mk search "ERROR" --since 1 --level ERROR

# 3. 按 traceId 追踪完整调用链
mk trace abc123def456

# 4. 启动 Web 可视化服务（默认端口 9528）
mk serve
```

---

## 命令详解

### 全局选项

```bash
--no-banner   # 不显示启动 banner
--no-color    # 禁用彩色输出
--json        # 输出 JSON 格式
--jq 'xxx'    # 内置 jq 查询（配合 --json 使用）
--config      # 指定自定义配置文件（默认：$HOME/.miaokun.yaml）
```

### list - 列出日志文件

```bash
mk list                          # 列出默认路径日志
mk list /var/log/app             # 指定目录扫描
mk list --since 7                # 仅显示最近7天的文件
```

### serve - 启动 Web 可视化服务

前端资源已内嵌二进制文件，单端口统一提供 API 与静态资源服务。

```bash
mk serve                        # 启动服务（默认端口 9528）
mk serve --port 8080            # 自定义端口
mk serve -v                     # 显示 API 请求日志
mk serve -vv                    # 调试模式
```

**Web 界面功能**  
- 实时日志搜索（支持正则表达式）
- 按日志级别一键过滤
- 精确时间范围筛选与多目录切换
- TraceId 跨文件全链路追踪
- 搜索结果自动高亮

**核心 API**  

| 接口 | 用途 |
|------|------|
| GET /api/v1/health | 健康检查 |
| GET /api/v1/files | 获取日志文件列表 |
| POST /api/v1/search/stream | SSE 流式搜索 |
| POST /api/v1/trace | TraceId 全链路追踪 |
| POST /api/v1/stats | 日志级别与文件分布统计 |

### search - 日志搜索（别名：grep）

```bash
# 基础搜索
mk search "NullPointerException"                # 搜索关键词
mk search "WARN" /var/log/apps                  # 指定目录

# 高级过滤
mk search "ERROR" --since 1                     # 最近1天
mk search "." --level ERROR                     # 指定级别
mk search "error" -i                            # 大小写不敏感
mk search "ERROR" --from "2026-06-01 10:00" --to "2026-06-01 12:00"  # 精确时间

# 结果增强
mk search "ERROR" -B 2 -A 2                     # 显示上下文
mk search "ERROR" --stats                       # 统计信息
mk search "ERROR" --count                       # 仅显示行数
mk search "ERROR" --json --jq '.[].message'     # 提取 JSON 字段
```

### trace - TraceId 全链路追踪

自动跨文件聚合同一 traceId 的所有日志，还原完整调用流程。

```bash
mk trace abc123def456                           # 全局追踪
mk trace 7a8b9c0d1e2f3a4b5c6d /var/log/app      # 指定目录
mk trace ABC123DEF -i                           # 大小写不敏感
```

### stats - 日志统计分析

```bash
mk stats                                        # 统计默认路径
mk stats /var/log/app                           # 指定目录
mk stats --since 7                              # 最近7天
```

---

## 配置说明

配置文件路径：`$HOME/.miaokun.yaml`，可通过 `--config` 指定自定义路径。

```yaml
default_paths:
  - /var/log
  - /opt/logs
  - /var/log/app

since_days: 7

cache_dir: /tmp/miaokun-cache
```

示例配置：`.miaokun.example.yaml`

---

## 项目结构

```
miaokun-log/
├── cmd/mk/                    # 主程序入口
├── internal/                  # 核心业务逻辑
│   ├── config/                # 配置管理
│   ├── discover/              # 日志文件发现
│   ├── cache/                 # 压缩文件缓存
│   ├── searcher/              # 流式搜索核心
│   ├── trace/                 # TraceId 链路追踪
│   ├── timefilter/            # 时间过滤
│   └── output/                # 输出格式化
├── pkg/types/                 # 公共类型定义
├── scripts/                   # 安装与构建脚本
└── Makefile                   # 编译构建规则
```

---

## 更多资源

- 📖 [CHANGELOG.md](./CHANGELOG.md)
- 🐛 [Gitee Issues](https://gitee.com/lorock/miaokun-log/issues)
- 📦 [Gitee Releases](https://gitee.com/lorock/miaokun-log/releases)
- 💬 加入微信群/QQ群获取技术支持

---

## 知识产权

**商标信息**  
喵坤® 已获得中华人民共和国国家知识产权局商标注册证（第78682220号），核定使用商品/服务项目（国际分类：9），有效期至2034年11月06日。

**版权信息**  
喵坤Logo作品已获得中华人民共和国国家版权局著作权登记（国作登字-2024-F-00181372），著作权人：徐保金，创作完成日期：2024年05月07日。

---

**喵坤<sup>®</sup>，让技术人的工作更轻松**