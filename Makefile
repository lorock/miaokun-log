.PHONY: build install clean test run help release release-all
.PHONY: build-frontend copy-frontend
.PHONY: build-darwin build-linux build-windows
.PHONY: build-darwin-amd64 build-darwin-arm64
.PHONY: build-linux-amd64 build-linux-arm64 build-linux-arm
.PHONY: build-windows-amd64 build-windows-arm64

BINARY=mk
VERSION=0.5.1
BUILD_DIR=bin
WEB_DIR=web
EMBED_DIR=internal/server/web/dist
LD_FLAGS=-ldflags="-s -w -X gitee.com/lorock/miaokun-log/pkg/version.Version=$(VERSION)"

build-frontend:
	@echo "📦 构建前端资源..."
	@cd $(WEB_DIR) && npm run build
	@echo "✅ 前端构建完成"

copy-frontend: build-frontend
	@echo "📁 复制前端资源到嵌入目录..."
	@mkdir -p $(EMBED_DIR)
	@cp -r $(WEB_DIR)/dist/* $(EMBED_DIR)/
	@echo "✅ 前端资源复制完成"

build: copy-frontend
	@echo "📦 构建二进制文件..."
	@mkdir -p $(BUILD_DIR)
	go build $(LD_FLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/mk
	@echo "✅ 当前平台编译完成: $(BUILD_DIR)/$(BINARY)"

install: build
	@echo "🔧 安装到 /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/
	sudo ln -sf /usr/local/bin/$(BINARY) /usr/local/bin/miaokun
	sudo ln -sf /usr/local/bin/$(BINARY) /usr/local/bin/mklog
	@echo "✅ 安装完成"

clean:
	@echo "🧹 清理..."
	rm -rf $(BUILD_DIR)
	rm -rf $(WEB_DIR)/dist
	rm -rf $(EMBED_DIR)
	go clean
	@echo "✅ 清理完成"

test:
	@echo "🧪 运行测试..."
	go test ./...
	@echo "✅ 测试完成"

run: build
	@echo "🚀 启动服务..."
	./$(BUILD_DIR)/$(BINARY) serve

help:
	@echo "喵坤 (MiaoKun) - Makefile 帮助"
	@echo ""
	@echo "常用命令:"
	@echo "  make build              - 构建前端 + 编译当前平台（单文件包含前端资源）"
	@echo "  make build-frontend     - 仅构建前端"
	@echo "  make install            - 安装到 /usr/local/bin"
	@echo "  make clean              - 清理编译产物"
	@echo "  make test               - 运行测试"
	@echo "  make run                - 构建并启动 Web 服务 (端口 9528)"
	@echo ""
	@echo "跨平台编译:"
	@echo "  make build-darwin       - 编译 macOS (通用)"
	@echo "  make build-linux        - 编译 Linux (amd64 + arm64 + arm)"
	@echo "  make build-windows      - 编译 Windows (amd64 + arm64)"
	@echo "  make release            - 编译所有平台并打包"
	@echo "  make release-all        - 编译所有支持的平台并打包"
	@echo ""
	@echo "单平台编译:"
	@echo "  make build-darwin-amd64   - macOS Intel"
	@echo "  make build-darwin-arm64   - macOS Apple Silicon"
	@echo "  make build-linux-amd64    - Linux x86_64"
	@echo "  make build-linux-arm64    - Linux ARM64"
	@echo "  make build-linux-arm      - Linux ARMv7"
	@echo "  make build-windows-amd64  - Windows x86_64"
	@echo "  make build-windows-arm64  - Windows ARM64"
	@echo ""
	@echo "版本: $(VERSION)"
	@echo "服务默认端口: 9528"

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

build-darwin: build-darwin-amd64 build-darwin-arm64
	@echo "✅ macOS 编译完成"

build-darwin-amd64: copy-frontend $(BUILD_DIR)
	@echo "📦 编译 macOS (Intel)..."
	GOOS=darwin GOARCH=amd64 go build $(LD_FLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/mk
	cd $(BUILD_DIR) && tar -czf $(BINARY)-$(VERSION)-darwin-amd64.tar.gz $(BINARY)-darwin-amd64
	@echo "✅ macOS (Intel) 编译完成"

build-darwin-arm64: copy-frontend $(BUILD_DIR)
	@echo "📦 编译 macOS (Apple Silicon)..."
	GOOS=darwin GOARCH=arm64 go build $(LD_FLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/mk
	cd $(BUILD_DIR) && tar -czf $(BINARY)-$(VERSION)-darwin-arm64.tar.gz $(BINARY)-darwin-arm64
	@echo "✅ macOS (Apple Silicon) 编译完成"

build-linux: build-linux-amd64 build-linux-arm64 build-linux-arm
	@echo "✅ Linux 编译完成"

build-linux-amd64: copy-frontend $(BUILD_DIR)
	@echo "📦 编译 Linux (amd64)..."
	GOOS=linux GOARCH=amd64 go build $(LD_FLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/mk
	cd $(BUILD_DIR) && tar -czf $(BINARY)-$(VERSION)-linux-amd64.tar.gz $(BINARY)-linux-amd64
	@echo "✅ Linux (amd64) 编译完成"

build-linux-arm64: copy-frontend $(BUILD_DIR)
	@echo "📦 编译 Linux (arm64)..."
	GOOS=linux GOARCH=arm64 go build $(LD_FLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-arm64 ./cmd/mk
	cd $(BUILD_DIR) && tar -czf $(BINARY)-$(VERSION)-linux-arm64.tar.gz $(BINARY)-linux-arm64
	@echo "✅ Linux (arm64) 编译完成"

build-linux-arm: copy-frontend $(BUILD_DIR)
	@echo "📦 编译 Linux (armv7)..."
	GOOS=linux GOARCH=arm GOARM=7 go build $(LD_FLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-armv7 ./cmd/mk
	cd $(BUILD_DIR) && tar -czf $(BINARY)-$(VERSION)-linux-armv7.tar.gz $(BINARY)-linux-armv7
	@echo "✅ Linux (armv7) 编译完成"

build-windows: build-windows-amd64 build-windows-arm64
	@echo "✅ Windows 编译完成"

build-windows-amd64: copy-frontend $(BUILD_DIR)
	@echo "📦 编译 Windows (amd64)..."
	GOOS=windows GOARCH=amd64 go build $(LD_FLAGS) -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/mk
	@if command -v zip >/dev/null 2>&1; then \
		cd $(BUILD_DIR) && zip -q $(BINARY)-$(VERSION)-windows-amd64.zip $(BINARY)-windows-amd64.exe; \
	else \
		echo "⚠️  zip 命令未找到，跳过打包，仅生成二进制文件"; \
	fi
	@echo "✅ Windows (amd64) 编译完成"

build-windows-arm64: copy-frontend $(BUILD_DIR)
	@echo "📦 编译 Windows (arm64)..."
	GOOS=windows GOARCH=arm64 go build $(LD_FLAGS) -o $(BUILD_DIR)/$(BINARY)-windows-arm64.exe ./cmd/mk
	@if command -v zip >/dev/null 2>&1; then \
		cd $(BUILD_DIR) && zip -q $(BINARY)-$(VERSION)-windows-arm64.zip $(BINARY)-windows-arm64.exe; \
	else \
		echo "⚠️  zip 命令未找到，跳过打包，仅生成二进制文件"; \
	fi
	@echo "✅ Windows (arm64) 编译完成"

release: build-darwin build-linux build-windows
	@echo ""
	@echo "✅ 所有平台编译完成！"
	@echo "输出目录: $(BUILD_DIR)"
	@ls -lh $(BUILD_DIR)

release-all: release
	@echo ""
	@echo "📦 所有平台编译完成！"

.DEFAULT_GOAL := help
