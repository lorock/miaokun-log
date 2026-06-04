# 喵坤日志排查工具 - 开发文档

## 概述

本文档为喵坤日志排查工具的开发维护指南，旨在帮助开发者理解项目结构、代码规范和开发流程。

---

## 项目架构

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                      架构分层                               │
├─────────────────────────────────────────────────────────────┤
│  Web 层 (web/)                                              │
│    ├── Vue 3 + TypeScript                                  │
│    ├── Element Plus UI 组件                                │
│    └── Vite 构建工具                                       │
├─────────────────────────────────────────────────────────────┤
│  API 层 (internal/server/)                                  │
│    ├── HTTP 服务器                                          │
│    ├── REST API 接口                                       │
│    └── SSE 流式响应                                         │
├─────────────────────────────────────────────────────────────┤
│  业务层 (internal/)                                         │
│    ├── searcher/     - 日志搜索核心                         │
│    ├── trace/        - TraceId 链路追踪                     │
│    ├── discover/     - 日志文件发现                         │
│    ├── cache/        - 压缩文件缓存                         │
│    ├── timefilter/   - 时间过滤                            │
│    └── output/       - 输出格式化                           │
├─────────────────────────────────────────────────────────────┤
│  基础设施层 (pkg/)                                          │
│    ├── types/        - 公共类型定义                         │
│    └── version/      - 版本管理                            │
└─────────────────────────────────────────────────────────────┘
```

### 目录结构

```
miaokun-log/
├── cmd/                     # 命令行入口
│   └── mk/                  # 主程序入口
├── internal/                # 核心业务逻辑
│   ├── cache/               # 压缩文件缓存管理
│   ├── config/              # 配置管理
│   ├── discover/            # 日志文件发现
│   ├── output/              # 输出格式化
│   ├── searcher/            # 日志搜索核心
│   ├── server/              # HTTP 服务器
│   ├── timefilter/          # 时间范围过滤
│   └── trace/               # TraceId 链路追踪
├── pkg/                     # 公共包
│   ├── types/               # 类型定义
│   └── version/             # 版本管理
├── web/                     # 前端代码
│   ├── src/
│   │   ├── components/      # Vue 组件
│   │   ├── composables/     # 组合式函数
│   │   ├── types/           # TypeScript 类型
│   │   └── App.vue          # 主应用组件
│   └── dist/                # 构建产物
├── scripts/                 # 脚本文件
├── .gitignore              # Git 忽略配置
├── .miaokun.example.yaml    # 配置文件示例
├── CHANGELOG.md            # 变更日志
├── DEVELOPMENT.md          # 开发文档（本文档）
├── Makefile                # 构建脚本
├── README.md               # 项目说明文档
└── go.mod                  # Go 依赖管理
```

---

## 代码规范

### Go 代码规范

1. **命名规则**
   - 包名：小写，使用简短有意义的名称
   - 变量名：驼峰式（camelCase）
   - 函数名：驼峰式（CamelCase）
   - 常量名：全大写，下划线分隔（UPPER_CASE）

2. **格式规范**
   - 使用 `go fmt` 自动格式化
   - 每行不超过 120 字符
   - 函数注释使用 Go 标准格式

3. **错误处理**
   - 必须检查所有错误
   - 使用 `fmt.Errorf("xxx: %w", err)` 包装错误

4. **导入顺序**
   - 标准库
   - 第三方库
   - 内部包

### 前端代码规范

1. **命名规则**
   - 组件名：PascalCase，后缀为 `.vue`
   - 变量名：驼峰式（camelCase）
   - 函数名：驼峰式（camelCase）
   - 文件目录：小写，使用连字符分隔

2. **Vue 组件规范**
   - 使用 Composition API
   - `<script setup>` 语法
   - 模板中使用 2 空格缩进
   - 组件顺序：template → script → style

3. **TypeScript 规范**
   - 必须为所有变量添加类型注解
   - 使用 `interface` 定义类型
   - 避免使用 `any` 类型

---

## 开发流程

### 环境要求

| 依赖 | 版本 | 说明 |
|------|------|------|
| Go | >= 1.22 | 后端开发 |
| Node.js | >= 20.x | 前端开发 |
| npm | >= 10.x | 包管理 |
| ripgrep | >= 14.x | 搜索引擎 |

### 开发步骤

**1. 克隆仓库**

```bash
git clone https://gitee.com/lorock/miaokun-log.git
cd miaokun-log
```

**2. 安装依赖**

```bash
# 后端：Go 模块会自动下载
go mod download

# 前端
cd web
npm install
cd ..
```

**3. 开发模式**

```bash
# 方式1：仅后端（命令行）
go run ./cmd/mk --help

# 方式2：前端开发（热更新）
cd web
npm run dev

# 方式3：完整构建运行
make run
```

**4. 代码提交**

```bash
# 检查格式
go fmt ./...
cd web && npm run lint

# 运行测试
make test

# 提交代码（遵循 Conventional Commits）
git add .
git commit -m "feat: 新增 xxx 功能"
```

### 提交规范

遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

| 类型 | 说明 |
|------|------|
| `feat` | 新增功能 |
| `fix` | 修复 bug |
| `docs` | 文档更新 |
| `style` | 代码格式（不影响功能） |
| `refactor` | 重构（既不新增也不修复） |
| `test` | 测试相关 |
| `chore` | 构建/工具更新 |

---

## API 文档

### 基础信息

- **基础路径**: `/api/v1`
- **内容类型**: `application/json`
- **版本端点**: `/api/v1/version`

### 接口列表

| 方法 | 端点 | 功能 |
|------|------|------|
| GET | `/health` | 健康检查 |
| GET | `/version` | 获取版本信息 |
| GET | `/files` | 获取日志文件列表 |
| GET | `/paths` | 获取可用路径配置 |
| POST | `/search` | 同步搜索 |
| POST | `/search/stream` | SSE 流式搜索 |
| POST | `/trace` | TraceId 追踪 |
| POST | `/stats` | 日志统计 |

### 接口详情

#### GET /api/v1/health

**响应**:
```json
{
  "status": "ok",
  "timestamp": "2026-06-04T12:00:00Z"
}
```

#### GET /api/v1/version

**响应**:
```json
{
  "version": "0.5.1",
  "build_date": "2026-06-04",
  "git_commit": "abc123"
}
```

#### GET /api/v1/files

**请求参数**:
- `since` (可选): 只返回最近 N 天的文件

**响应**:
```json
[
  {
    "path": "/var/log/app.log",
    "size": 1048576,
    "mod_time": "2026-06-04T12:00:00Z"
  }
]
```

#### POST /api/v1/search/stream (SSE)

**请求体**:
```json
{
  "pattern": "ERROR",
  "paths": ["/var/log"],
  "level": "ERROR",
  "before": 3,
  "after": 5,
  "case_insensitive": true,
  "since_days": 1
}
```

**响应** (SSE 流):
```json
{
  "type": "match",
  "file": "/var/log/app.log",
  "line": 123,
  "content": "2026-06-04 12:00:00 ERROR Something went wrong",
  "level": "ERROR"
}
```

---

## 版本管理

### 版本号规则

使用语义化版本（Semantic Versioning）：
- `vX.Y.Z`
- `X`: 主版本号（不兼容的 API 变更）
- `Y`: 次版本号（新增功能，向后兼容）
- `Z`: 修订号（修复 bug，向后兼容）

### 版本更新流程

1. 修改 `Makefile` 中的 `VERSION` 变量
2. 运行 `make build` 自动注入版本信息
3. 更新 `CHANGELOG.md`
4. 提交代码并打标签

```bash
# 打标签
git tag -a v0.5.1 -m "Release v0.5.1"
git push origin v0.5.1
```

---

## 部署指南

### 本地部署

```bash
# 构建
make build

# 启动
./bin/mk serve

# 访问
open http://localhost:9528
```

### 生产部署

```bash
# 编译指定平台
make build-linux-amd64

# 复制到服务器
scp bin/mk-linux-amd64 user@server:/usr/local/bin/mk

# 启动服务（建议使用 systemd）
mk serve --port 9528
```

### Docker 部署（可选）

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN apk add --no-cache git make nodejs npm
RUN make build

FROM alpine:latest
RUN apk add --no-cache ripgrep
COPY --from=builder /app/bin/mk /usr/local/bin/
EXPOSE 9528
CMD ["mk", "serve"]
```

---

## 调试技巧

### 启用调试模式

```bash
# 命令行调试
mk search "ERROR" --json -vv

# Web 服务调试
mk serve -vv
```

### 日志级别

| 级别 | 说明 |
|------|------|
| `-v` | 显示请求/响应日志 |
| `-vv` | 显示详细调试信息 |

---

## 常见问题

### Q1: 构建失败，提示 ripgrep 未找到

**解决方案**:
```bash
# macOS
brew install ripgrep

# Ubuntu/Debian
sudo apt install ripgrep

# CentOS/RHEL
sudo yum install ripgrep
```

### Q2: 前端构建失败，提示依赖错误

**解决方案**:
```bash
cd web
rm -rf node_modules package-lock.json
npm install
npm run build
```

### Q3: 端口被占用

**解决方案**:
```bash
mk serve --port 8080
```

---

## 贡献指南

欢迎贡献代码！请遵循以下流程：

1. Fork 仓库
2. 创建特性分支 (`feature/xxx`)
3. 提交代码
4. 创建 Pull Request
5. 等待审核

---

**文档版本**: v0.1  
**最后更新**: 2026-06-04  
**作者**: 喵坤开发团队
