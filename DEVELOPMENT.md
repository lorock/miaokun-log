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
│    ├── Vite 构建工具                                       │
│    ├── Composables (useAuth, useLogStream, useFileList)   │
│    ├── Components (AuthGuard, FileBrowserModal, LogList)    │
│    └── 认证系统 (登录页、Token 管理)                        │
├─────────────────────────────────────────────────────────────┤
│  API 层 (internal/server/)                                  │
│    ├── HTTP 服务器                                          │
│    ├── REST API 接口                                       │
│    ├── SSE 流式响应                                         │
│    ├── 认证中间件 (JWT, Basic Auth, API Key)               │
│    └── 文件浏览 API                                         │
├─────────────────────────────────────────────────────────────┤
│  业务层 (internal/)                                         │
│    ├── searcher/     - 日志搜索核心                         │
│    ├── trace/        - TraceId 链路追踪                     │
│    ├── discover/     - 日志文件发现                         │
│    ├── cache/        - 压缩文件缓存                         │
│    ├── timefilter/   - 时间过滤                            │
│    ├── output/       - 输出格式化                           │
│    └── auth/         - 认证相关功能                         │
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
│   ├── auth/               # 认证相关功能（JWT、路径安全）
│   ├── cache/              # 压缩文件缓存管理
│   ├── config/             # 配置管理
│   ├── discover/           # 日志文件发现
│   ├── output/             # 输出格式化
│   ├── searcher/           # 日志搜索核心
│   ├── server/             # HTTP 服务器（含文件浏览 API）
│   ├── timefilter/         # 时间范围过滤
│   └── trace/              # TraceId 链路追踪
├── pkg/                     # 公共包
│   ├── types/              # 类型定义
│   └── version/             # 版本管理
├── web/                     # 前端代码
│   ├── src/
│   │   ├── components/      # Vue 组件
│   │   │   ├── AuthGuard.vue       # 认证守卫
│   │   │   ├── FileBrowserModal.vue # 文件浏览模态框
│   │   │   ├── FileList.vue        # 文件列表组件
│   │   │   ├── LogList.vue         # 日志列表组件
│   │   │   ├── LoginPage.vue       # 登录页面
│   │   │   └── SearchForm.vue      # 搜索表单
│   │   ├── composables/     # 组合式函数
│   │   │   ├── useAuth.ts          # 认证状态管理
│   │   │   ├── useFileList.ts      # 文件列表请求
│   │   │   └── useLogStream.ts     # 日志流处理
│   │   ├── types/           # TypeScript 类型
│   │   │   ├── auth.ts            # 认证相关类型
│   │   │   └── index.ts           # 通用类型
│   │   └── App.vue           # 主应用组件
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

### 核心依赖

| 依赖 | 版本 | 说明 |
|------|------|------|
| github.com/golang-jwt/jwt/v5 | v5.3.1 | JWT Token 生成与验证 |
| golang.org/x/crypto | v0.52.0 | 密码加密（bcrypt） |
| github.com/spf13/viper | v1.21.0 | 配置管理 |
| github.com/spf13/cobra | v1.10.2 | CLI 命令行 |
| github.com/itchyny/gojq | v0.12.19 | JSON 查询 |

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

| 方法 | 端点 | 功能 | 认证 |
|------|------|------|------|
| GET | `/health` | 健康检查 | 否 |
| GET | `/version` | 获取版本信息 | 否 |
| GET | `/files` | 获取日志文件列表 | 否 |
| GET | `/files/list` | **获取文件列表（分页+详情）** | API Key |
| GET | `/paths` | 获取可用路径配置 | 否 |
| POST | `/search` | 同步搜索 | 否 |
| POST | `/search/stream` | SSE 流式搜索 | 否 |
| POST | `/trace` | TraceId 追踪 | 否 |
| POST | `/stats` | 日志统计 | 否 |

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

#### GET /api/v1/files/list

**功能**: 获取指定目录下的文件列表，支持分页、时间过滤和详细文件信息。

**认证方式**: API Key (Header: `X-API-Key` 或 Query: `api_key`)

**请求参数**:

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `path` | string | 否 | `/` | 目录路径（默认根目录） |
| `page` | int | 否 | `1` | 页码 |
| `page_size` | int | 否 | `50` | 每页数量 (1-500) |
| `since` | float | 否 | `30` | 只显示最近 N 天修改的文件 |

**请求示例**:
```bash
# 使用 API Key (Header)
curl -H "X-API-Key: your-api-key" \
  "http://localhost:9528/api/v1/files/list?path=/var/log&page=1&page_size=20"

# 使用 API Key (Query)
curl "http://localhost:9528/api/v1/files/list?path=/var/log&api_key=your-api-key"
```

**成功响应** (200 OK):
```json
{
  "success": true,
  "data": [
    {
      "name": "app.log",
      "path": "/var/log",
      "full_path": "/var/log/app.log",
      "size": 1048576,
      "size_readable": "1.00 MB",
      "mod_time": "2026-06-04T12:00:00Z",
      "mod_time_str": "2026-06-04T12:00:00Z",
      "file_type": "log",
      "is_dir": false,
      "is_readable": true
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 50,
    "total": 100,
    "total_pages": 2,
    "has_next": true,
    "has_prev": false
  }
}
```

**错误响应**:

**401 Unauthorized** - 缺少或无效的 API Key:
```json
{
  "success": false,
  "error": {
    "code": "MISSING_API_KEY",
    "message": "API key required",
    "details": ""
  }
}
```

**403 Forbidden** - 路径不在允许列表中:
```json
{
  "success": false,
  "error": {
    "code": "PATH_NOT_ALLOWED",
    "message": "Access to the requested path is not allowed",
    "details": "Path '/etc' is not in the allowed paths list"
  }
}
```

**404 Not Found** - 路径不存在:
```json
{
  "success": false,
  "error": {
    "code": "PATH_NOT_FOUND",
    "message": "The requested path does not exist",
    "details": "/var/log/nonexistent"
  }
}
```

**400 Bad Request** - 参数错误:
```json
{
  "success": false,
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Failed to parse request parameters",
    "details": "invalid page number: abc"
  }
}
```

**安全特性**:
- **路径遍历防护**: 自动过滤 `..` 等路径遍历序列
- **敏感目录过滤**: 自动过滤系统敏感目录（`/etc`, `/proc`, `/sys`, `/root` 等）
- **动态权限检查**: 根据运行用户动态允许访问 `/root` 目录（root 用户可见）
- **认证中间件**: 支持 API Key 认证（可扩展为 Basic Auth）
- **常量时间比较**: 防止时序攻击

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

## 错误代码说明

API 返回错误时，响应体包含 `error` 对象：

```json
{
  "success": false,
  "data": null,
  "error": {
    "code": "ERROR_CODE",
    "message": "错误描述信息"
  }
}
```

### 通用错误代码

| 错误代码 | HTTP 状态码 | 中文描述 | 说明 |
|----------|-------------|----------|------|
| `AUTHENTICATION_REQUIRED` | 401 | 请先登录后再操作 | 未提供认证信息或认证失败 |
| `NOT_AUTHENTICATED` | 401 | 请先登录后再操作 | 认证已过期或无效 |
| `TOKEN_EXPIRED` | 401 | 登录已过期，请重新登录 | Token 已过期 |
| `INVALID_TOKEN` | 401 | Token 无效 | Token 格式错误或被篡改 |
| `PERMISSION_DENIED` | 403 | 您没有执行此操作的权限 | 用户权限不足 |
| `ROLE_REQUIRED` | 403 | 此操作需要 xxx 角色权限 | 需要特定角色 |
| `INVALID_PATH` | 400 | 路径无效或为空 | 请求路径参数无效 |
| `INVALID_PARAMETER` | 400 | 参数错误 | 请求参数格式错误 |
| `INTERNAL_ERROR` | 500 | 服务器内部错误 | 服务器处理失败 |

### 401 未认证错误处理

当前端收到 401 响应时，应：
1. 显示错误提示（"请先登录后再操作"）
2. 自动清除本地认证状态（localStorage）
3. 跳转到登录页面

```typescript
// 前端示例：401 响应处理
if (response.status === 401) {
  const data = await response.json();
  ElMessage.error(data.error.message || '请先登录后再操作');
  // 清除认证状态并跳转登录页
  logout();
  router.push('/login');
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

## 安全指南

### API 认证机制

#### API Key 认证

`/api/v1/files/list` 端点使用 API Key 认证：

**配置方式** (在代码中设置):
```go
authConfig := server.AuthConfig{
    Enabled:  true,
    Username: "admin",
    Password: "secure-password",
}

// 或使用 API Key
apiKeys := []string{"your-secret-api-key"}
```

**使用方式**:
```bash
# Header 方式
curl -H "X-API-Key: your-secret-api-key" \
  "http://localhost:9528/api/v1/files/list"

# Query 参数方式
curl "http://localhost:9528/api/v1/files/list?api_key=your-secret-api-key"
```

#### Basic Auth 认证

```bash
curl -u username:password \
  "http://localhost:9528/api/v1/files/list"
```

### 路径安全

#### 路径遍历防护

系统自动防护以下攻击：
- `../../../etc/passwd` → 被拒绝
- `..\windows\system32` → 被拒绝
- 包含空字节的路径 → 被拒绝

#### 路径白名单

只允许访问配置的默认路径：
```yaml
# ~/.miaokun.yaml
default_paths:
  - /var/log
  - /opt/logs
```

### 安全测试

#### 路径遍历测试

```bash
# 测试路径遍历防护
curl "http://localhost:9528/api/v1/files/list?path=../../../etc"
# 预期: 403 Forbidden

# 测试允许的路径
curl "http://localhost:9528/api/v1/files/list?path=/var/log"
# 预期: 200 OK
```

#### 认证测试

```bash
# 测试无认证访问
curl "http://localhost:9528/api/v1/files/list"
# 预期: 401 Unauthorized

# 测试错误认证
curl -H "X-API-Key: wrong-key" \
  "http://localhost:9528/api/v1/files/list"
# 预期: 401 Unauthorized
```

### 生产环境安全建议

1. **使用 HTTPS**: 生产环境必须启用 TLS
2. **强密码策略**: API Key 至少 32 位随机字符
3. **IP 白名单**: 限制可访问的 IP 地址
4. **日志审计**: 启用详细日志记录访问行为
5. **定期更新**: 及时更新依赖库修复安全漏洞

---

## 贡献指南

欢迎贡献代码！请遵循以下流程：

1. Fork 仓库
2. 创建特性分支 (`feature/xxx`)
3. 提交代码
4. 创建 Pull Request
5. 等待审核

---

**文档版本**: v0.3  
**最后更新**: 2026-06-06  
**作者**: 喵坤开发团队
