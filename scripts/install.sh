#!/bin/bash
# 喵坤 (MiaoKun) 智能安装脚本

set -e

VERSION="0.1.0"
INSTALL_DIR="/usr/local/bin"
BIN_NAME="mk"
ALT_NAME="miaokun"
COMPAT_NAME="mklog"

echo "🐾 喵坤 (MiaoKun) v${VERSION} 安装脚本"
echo "========================================"

if ! command -v go &> /dev/null; then
    echo "❌ 错误: 未找到 Go。请先安装 Go 1.22+"
    echo "   下载: https://golang.org/dl/"
    exit 1
fi

if ! command -v rg &> /dev/null; then
    echo "⚠️  警告: 未找到 ripgrep (rg)"
    echo "   建议安装: "
    echo "     Ubuntu/Debian: sudo apt install ripgrep"
    echo "     CentOS/RHEL:   sudo yum install ripgrep"
    echo "     macOS:         brew install ripgrep"
fi

echo "🔨 构建中..."
go build -ldflags="-s -w" -o ${BIN_NAME} ./cmd/mk

if command -v mk &> /dev/null && [ "$(command -v mk)" != "${INSTALL_DIR}/${BIN_NAME}" ]; then
    echo "⚠️  检测到系统已存在 mk 命令"
    echo "    喵坤将安装为: ${ALT_NAME}"
    BIN_NAME="${ALT_NAME}"
fi

echo "📦 安装到 ${INSTALL_DIR}..."
sudo mv ${BIN_NAME} ${INSTALL_DIR}/${BIN_NAME}

sudo ln -sf ${INSTALL_DIR}/${BIN_NAME} ${INSTALL_DIR}/${ALT_NAME}
sudo ln -sf ${INSTALL_DIR}/${BIN_NAME} ${INSTALL_DIR}/${COMPAT_NAME}

echo ""
echo "✅ 安装完成！"
echo ""
echo "试试看:"
echo "  ${BIN_NAME} --version"
echo "  ${ALT_NAME} --version"
echo "  ${COMPAT_NAME} --version"
echo ""
echo "示例:"
echo "  ${BIN_NAME} search \"ERROR\" --since 1"
echo "  ${BIN_NAME} trace abc123def456 /var/log/apps/"
echo ""
echo "🐾 喵坤已就绪，开始狩猎吧！"
