#!/bin/bash

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# 清理残余
cleanup() {
    echo -e "\n${YELLOW}清理临时文件中，请勿强制关闭...${NC}"
    rm -rf "$TEMP_DIR"
    exit
}

# 处理 SIGINT (Ctrl+C)
trap cleanup SIGINT

# 创建临时目录
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR" || exit 1

# 检测系统架构
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case $ARCH in
x86_64) ARCH="amd64" ;;
aarch64) ARCH="arm64" ;;
arm64) ARCH="arm64" ;;
esac

# 从 GitHub API 获取最新版本
REPO="alice39s/aqua-speed-tools"
VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

# Determine download URL
case $OS in
linux)
    BINARY_URL="https://github.com/$REPO/releases/latest/download/aqua-speed-tools-linux-$ARCH"
    ;;
darwin)
    BINARY_URL="https://github.com/$REPO/releases/latest/download/aqua-speed-tools-darwin-$ARCH"
    ;;
*)
    echo -e "${RED}Unsupported operating system${NC}"
    cleanup
    ;;
esac

# 下载二进制文件
echo -e "${YELLOW}下载主程序中...${NC}"
curl -L -o aqua-speed-tools "$BINARY_URL"
chmod +x aqua-speed-tools

cat <<"EOF"
    ___                        _____                     __   ______            __    
   /   | ____ ___  ______ _   / ___/____  ___  ___  ____/ /  /_  __/___  ____  / /____
  / /| |/ __ `/ / / / __ `/   \__ \/ __ \/ _ \/ _ \/ __  /    / / / __ \/ __ \/ / ___/
 / ___ / /_/ / /_/ / /_/ /   ___/ / /_/ /  __/  __/ /_/ /    / / / /_/ / /_/ / (__  ) 
/_/  |_\__, /\__,_/\__,_/   /____/ .___/\___/\___/\__,_/    /_/  \____/\____/_/____/  
         /_/                    /_/                                                   
EOF

echo -e "\n${CYAN}仓库:${NC} https://github.com/$REPO"
echo -e "${CYAN}版本:${NC} $VERSION"
echo -e "${CYAN}作者:${NC} Alice39s\n"

# 交互菜单
while true; do
    echo -e "${GREEN}请输入要执行选项的数字:${NC}"
    echo -e "1) ${BOLD}列出所有节点${NC}"
    echo -e "2) ${BOLD}测试指定节点${NC}"
    echo -e "3) ${BOLD}退出${NC}"
    read -r choice

    case $choice in
    1)
        echo -e "\n${BLUE}列出所有节点...${NC}"
        ./aqua-speed-tools list
        ;;
    2)
        echo -e "\n${BLUE}请输入节点ID:${NC}"
        read -r node_id
        ./aqua-speed-tools test "$node_id"
        ;;
    3)
        cleanup
        ;;
    *)
        echo -e "${RED}无效选项，请重新输入${NC}"
        ;;
    esac
    echo
done
