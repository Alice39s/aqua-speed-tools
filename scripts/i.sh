#!/bin/sh

# 设置严格模式
set -eu

RAW_BASE_URL="${AQUA_SPEED_RAW_URL:-https://raw.githubusercontent.com/}" 
GITHUB_BASE_URL="${AQUA_SPEED_GITHUB_BASE_URL:-https://github.com/}"
GITHUB_API_BASE_URL="${AQUA_SPEED_GITHUB_API_BASE_URL:-https://api.github.com/}"
REPO="alice39s/aqua-speed-tools"

CONFIG_JSON_URL="$RAW_BASE_URL/$REPO/main/configs/base.json"
CONFIG_URL="${AQUA_SPEED_CONFIG_URL:-$CONFIG_JSON_URL}"
CONFIG_DIR="configs"
TEMP_DIR=""

# 检查是否支持彩色输出
if [ -t 1 ] && command -v tput >/dev/null 2>&1; then
    RED=$(tput setaf 1)
    GREEN=$(tput setaf 2)
    BLUE=$(tput setaf 4)
    YELLOW=$(tput setaf 3)
    CYAN=$(tput setaf 6)
    BOLD=$(tput bold)
    NC=$(tput sgr0)
else
    RED=""
    GREEN=""
    BLUE=""
    YELLOW=""
    CYAN=""
    BOLD=""
    NC=""
fi

# 日志函数
log_info() {
    printf "${BLUE}[INFO] %s${NC}\n" "$1" >&2
}

log_success() {
    printf "${GREEN}[DONE] %s${NC}\n" "$1" >&2
}

log_warning() {
    printf "${YELLOW}[WARN] %s${NC}\n" "$1" >&2
}

log_error() {
    printf "${RED}[ERROR] %s${NC}\n" "$1" >&2
}

# 清理函数
cleanup() {
    exit_code=$?
    log_warning "清理临时文件中，请勿强制关闭..."
    if [ -d "${TEMP_DIR}" ]; then
        rm -rf "${TEMP_DIR}"
    fi
    log_info "清理完成"
    exit ${exit_code}
}

# 检查命令是否存在
check_command() {
    cmd=$1
    if ! command -v "${cmd}" >/dev/null 2>&1; then
        log_error "未找到命令: ${cmd}"
        return 1
    fi
    return 0
}

# 检查并创建临时目录
setup_temp_dir() {
    if command -v mktemp >/dev/null 2>&1; then
        TEMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t 'aquaspeed')
    else
        TEMP_DIR="/tmp/aquaspeed-$$"
        mkdir -p "${TEMP_DIR}"
    fi

    if [ ! -d "${TEMP_DIR}" ]; then
        log_error "创建临时目录失败"
        return 1
    fi

    if ! cd "${TEMP_DIR}"; then
        log_error "无法切换到临时目录"
        return 1
    fi
}

# 创建配置目录
setup_config_dir() {
    if ! mkdir -p "${CONFIG_DIR}"; then
        log_error "创建配置目录失败"
        return 1
    fi
}

# 检测系统信息
detect_system() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case ${ARCH} in
    x86_64 | amd64) ARCH="amd64" ;;
    aarch64 | arm64 | armv8*) ARCH="arm64" ;;
    armv7*) ARCH="armv7" ;;
    *)
        log_error "不支持的系统架构: ${ARCH}"
        return 1
        ;;
    esac

    case ${OS} in
    linux | darwin) : ;;
    *)
        log_error "不支持的操作系统: ${OS}"
        return 1
        ;;
    esac
}

# 获取最新版本
get_latest_version() {
    attempt=1
    max_attempts=3

    while [ ${attempt} -le ${max_attempts} ]; do
        if api_result=$(curl -sSL --connect-timeout 10 --max-time 15 "${GITHUB_API_BASE_URL}/repos/${REPO}/releases/latest"); then
            VERSION=$(echo "${api_result}" | grep -o '"tag_name": *"[^"]*"' | cut -d'"' -f4)
            if [ -n "${VERSION}" ]; then
                return 0
            fi
        fi
        log_warning "获取版本信息失败，尝试第 ${attempt}/${max_attempts} 次"
        attempt=$((attempt + 1))
        sleep 2
    done

    log_error "获取版本信息失败"
    return 1
}

# 下载文件
download_file() {
    url=$1
    output=$2
    attempt=1
    max_attempts=3

    while [ ${attempt} -le ${max_attempts} ]; do
        if curl -sSL --connect-timeout 10 --max-time 30 -o "${output}" "${url}"; then
            return 0
        fi
        log_warning "下载失败，尝试第 ${attempt}/${max_attempts} 次"
        attempt=$((attempt + 1))
        sleep 2
    done

    return 1
}

# 下载二进制文件
download_binary() {
    binary_url="${GITHUB_BASE_URL}/${REPO}/releases/latest/download/aqua-speed-tools-${OS}-${ARCH}"

    log_info "下载主程序中..."
    if ! download_file "${binary_url}" "aqua-speed-tools"; then
        log_error "下载主程序失败"
        return 1
    fi

    if ! chmod +x aqua-speed-tools; then
        log_error "设置执行权限失败"
        return 1
    fi
}

# 下载配置文件
download_config() {
    log_info "下载配置文件中..."
    if ! download_file "${CONFIG_URL}" "${CONFIG_DIR}/base.json"; then
        log_error "下载配置文件失败"
        return 1
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

    printf "\n${CYAN}仓库:${NC} https://github.com/%s\n" "${REPO}"
    printf "${CYAN}版本:${NC} %s\n" "${VERSION}"
    printf "${CYAN}作者:${NC} Alice39s\n\n"
}

# 显示菜单
show_menu() {
    printf "${GREEN}请输入要执行选项的数字:${NC}\n"
    printf "1) ${BOLD}列出所有节点${NC}\n"
    printf "2) ${BOLD}测试指定节点${NC}\n"
    printf "3) ${BOLD}退出${NC}\n"
}

# 列出节点并获取输入
list_and_get_input() {
    ./aqua-speed-tools list
    printf "\n${BLUE}请输入要测试的节点ID:${NC}\n"
    read -r node_id
    ./aqua-speed-tools test "${node_id}"
}

# 处理用户输入
handle_input() {
    read -r choice
    case ${choice} in
    1)
        log_info "列出所有节点..."
        list_and_get_input
        ;;
    2)
        printf "\n${BLUE}请输入节点ID:${NC}\n"
        read -r node_id
        ./aqua-speed-tools test "${node_id}"
        ;;
    3)
        cleanup
        ;;
    *)
        log_error "无效选项，请重新输入"
        return 1
        ;;
    esac
    printf "\n"
}

# 主函数
main() {
    trap cleanup INT TERM

    # 检查必要命令
    for cmd in curl grep sed; do
        check_command "${cmd}" || exit 1
    done

    # 初始化
    setup_temp_dir || exit 1
    setup_config_dir || exit 1
    detect_system || exit 1
    get_latest_version || exit 1
    download_binary || exit 1
    download_config || exit 1
    show_logo

    # 主循环
    while true; do
        show_menu
        handle_input || continue
    done
}

# 执行主函数
main
