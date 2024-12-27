#!/bin/bash

# 常量定义
REPO="alice39s/aqua-speed-tools"
CONFIG_URL="https://raw.githubusercontent.com/$REPO/main/configs/base.json"
TEMP_DIR=""
CONFIG_DIR="configs"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # 无颜色
BOLD='\033[1m'

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO] $1${NC}"
}

log_success() {
    echo -e "${GREEN}[DONE] $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}[WARN] $1${NC}"
}

log_error() {
    echo -e "${RED}[ERROR] $1${NC}" >&2
}

# 清理函数
cleanup() {
    log_warning "清理临时文件中，请勿强制关闭..."
    if [ -d "$TEMP_DIR" ]; then
        rm -rf "$TEMP_DIR"
    fi
    exit 0
}

# 检查命令是否存在
check_command() {
    if ! command -v "$1" &> /dev/null; then
        log_error "未找到命令: $1"
        exit 1
    fi
}

# 检查并创建临时目录
setup_temp_dir() {
    TEMP_DIR=$(mktemp -d)
    if [ ! -d "$TEMP_DIR" ]; then
        log_error "创建临时目录失败"
        exit 1
    fi
    cd "$TEMP_DIR" || exit 1
}

# 创建配置目录
setup_config_dir() {
    mkdir -p "$CONFIG_DIR"
}

# 检测系统信息
detect_system() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *)
            log_error "不支持的系统架构: $ARCH"
            cleanup
            ;;
    esac
}

# 获取最新版本
get_latest_version() {
    VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        log_error "获取版本信息失败"
        cleanup
    fi
}

# 下载二进制文件
download_binary() {
    case $OS in
        linux)
            BINARY_URL="https://github.com/$REPO/releases/latest/download/aqua-speed-tools-linux-$ARCH"
            ;;
        darwin)
            BINARY_URL="https://github.com/$REPO/releases/latest/download/aqua-speed-tools-darwin-$ARCH"
            ;;
        *)
            log_error "不支持的操作系统: $OS"
            cleanup
            ;;
    esac

    log_info "下载主程序中..."
    if ! curl -L -o aqua-speed-tools "$BINARY_URL"; then
        log_error "下载主程序失败"
        cleanup
    fi
    chmod +x aqua-speed-tools
}

# 下载配置文件
download_config() {
    log_info "下载配置文件中..."
    if ! curl -L -o "$CONFIG_DIR/base.json" "$CONFIG_URL"; then
        log_error "下载配置文件失败"
        cleanup
    fi
}

# 显示 LOGO
show_logo() {
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
}

# 显示菜单
show_menu() {
    echo -e "${GREEN}请输入要执行选项的数字:${NC}"
    echo -e "1) ${BOLD}列出所有节点${NC}"
    echo -e "2) ${BOLD}测试指定节点${NC}"
    echo -e "3) ${BOLD}退出${NC}"
}

# 处理用户输入
handle_input() {
    read -r choice
    case $choice in
        1)
            log_info "列出所有节点..."
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
            log_error "无效选项，请重新输入"
            ;;
    esac
    echo
}

# 主函数
main() {
    # 检查必要命令
    check_command "curl"
    check_command "grep"
    check_command "sed"

    # 注册清理函数
    trap cleanup SIGINT SIGTERM

    # 初始化
    setup_temp_dir
    setup_config_dir
    detect_system
    get_latest_version
    download_binary
    download_config
    show_logo

    # 主循环
    while true; do
        show_menu
        handle_input
    done
}

# 执行主函数
main