#!/bin/sh
# 设置严格模式
set -eu

# 默认变量
RAW_BASE_URL="${AQUA_SPEED_RAW_URL:-https://raw.githubusercontent.com}"
GITHUB_BASE_URL="${AQUA_SPEED_GITHUB_BASE_URL:-https://github.com}"
GITHUB_API_BASE_URL="${AQUA_SPEED_GITHUB_API_BASE_URL:-https://api.github.com}"
REPO="alice39s/aqua-speed-tools"

CONFIG_JSON_URL="$RAW_BASE_URL/$REPO/main/configs/base.json"
CONFIG_URL="${AQUA_SPEED_CONFIG_URL:-$CONFIG_JSON_URL}"
CONFIG_DIR="configs"
TEMP_DIR=""
EXIT_CODE=0

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
    printf "%s[INFO]%s %s\n" "$BLUE" "$NC" "$1" >&2
}

log_success() {
    printf "%s[DONE]%s %s\n" "$GREEN" "$NC" "$1" >&2
}

log_warning() {
    printf "%s[WARN]%s %s\n" "$YELLOW" "$NC" "$1" >&2
}

log_error() {
    printf "%s[ERROR]%s %s\n" "$RED" "$NC" "$1" >&2
    EXIT_CODE=1
}

# 清理函数
cleanup() {
    local exit_status=$?
    [ "${exit_status}" -ne 0 ] && EXIT_CODE=${exit_status}

    # BusyBox sh 不支持 trap ''，使用另一个 trap 清除之前的 traps
    trap "" INT TERM EXIT

    if [ "${EXIT_CODE}" -ne 0 ]; then
        log_error "程序异常退出，错误码: ${EXIT_CODE}"
    fi

    log_warning "清理临时文件中，请勿强制关闭..."
    if [ -n "${TEMP_DIR}" ] && [ -d "${TEMP_DIR}" ]; then
        rm -rf "${TEMP_DIR}" || log_error "删除临时目录失败"
    fi
    log_info "清理完成"
    exit "${EXIT_CODE}"
}

# 错误处理函数
handle_error() {
    local exit_code=$?
    log_error "脚本发生错误，错误码: ${exit_code}"
    exit "${exit_code}"
}

# 检查命令是否存在
check_command() {
    local cmd="$1"
    if ! command -v "${cmd}" >/dev/null 2>&1; then
        log_error "未找到命令: ${cmd}"
        return 1
    fi
    return 0
}

# 确认是否安装命令
confirm_install_command() {
    cmd="$1"
    # 使用 | 作为分隔符，确保 install_cmd 中的空格不会导致拆分
    package_managers="apt:apt install -y|yum:yum install -y|brew:brew install|pacman:pacman -S --noconfirm|apk:apk add --no-cache"

    # 检查命令是否已存在
    if ! command -v "${cmd}" >/dev/null 2>&1; then
        # 检查是否有root权限
        if [ "$(id -u)" -ne 0 ]; then
            log_error "安装命令需要root权限，请使用sudo运行此脚本"
            return 1
        fi

        # 询问用户是否安装
        log_warning "未找到命令: ${cmd}, 是否使用包管理器安装?"
        printf "请输入 y 或 n: "
        read -r confirm
        case "${confirm}" in
        [Yy]*) ;;
        *) return 1 ;;
        esac

        log_info "${cmd} 安装中..."

        found_manager=0
        install_success=0

        OLD_IFS="$IFS"
        IFS="|"
        for pair in ${package_managers}; do
            IFS=":"
            set -- ${pair}
            manager=$1
            install_cmd=$2
            IFS="$OLD_IFS"

            # 检查 manager 和 install_cmd 是否都已设置
            if [ -n "${manager}" ] && [ -n "${install_cmd}" ]; then
                if command -v "${manager}" >/dev/null 2>&1; then
                    found_manager=1
                    if eval "${install_cmd} ${cmd}" >/dev/null 2>&1; then
                        if command -v "${cmd}" >/dev/null 2>&1; then
                            log_success "${cmd} 安装成功"
                            install_success=1
                            break
                        else
                            log_warning "${cmd} 安装命令执行成功，但可能安装失败，请尝试手动安装 ${cmd}"
                        fi
                    else
                        log_error "使用 ${manager} 安装 ${cmd} 失败"
                    fi
                fi
            else
                log_warning "无效的包管理器配置: ${pair}"
            fi
        done

        IFS="$OLD_IFS"

        if [ "${found_manager}" -eq 0 ]; then
            log_error "无法安装命令: ${cmd}，未找到支持的包管理器"
            return 1
        fi

        if [ "${install_success}" -eq 0 ]; then
            log_error "无法安装命令: ${cmd}，所有包管理器安装尝试均失败"
            return 1
        fi
    fi

    return 0
}

# 检查并创建临时目录
setup_temp_dir() {
    if ! command -v mktemp >/dev/null 2>&1; then
        log_error "未找到mktemp命令"
        return 1
    fi

    TEMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t 'aquaspeed')
    if [ ! -d "${TEMP_DIR}" ]; then
        log_error "创建临时目录失败"
        return 1
    fi

    # 检查目录权限
    if [ ! -w "${TEMP_DIR}" ]; then
        log_error "临时目录无写入权限: ${TEMP_DIR}"
        return 1
    fi

    if ! cd "${TEMP_DIR}"; then
        log_error "无法切换到临时目录: ${TEMP_DIR}"
        return 1
    fi

    return 0
}

# 创建配置目录
setup_config_dir() {
    if [ -d "${CONFIG_DIR}" ]; then
        if [ ! -w "${CONFIG_DIR}" ]; then
            log_error "配置目录无写入权限: ${CONFIG_DIR}"
            return 1
        fi
    else
        if ! mkdir -p "${CONFIG_DIR}"; then
            log_error "创建配置目录失败: ${CONFIG_DIR}"
            return 1
        fi
    fi
    return 0
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
    linux | darwin | freebsd) : ;;
    *)
        log_error "不支持的操作系统: ${OS}"
        return 1
        ;;
    esac

    log_info "系统: ${OS}, 架构: ${ARCH}"
    return 0
}

# 获取最新版本
get_latest_version() {
    attempt=1
    max_attempts=3
    VERSION=""
    curl_timeout=10
    curl_max_time=15

    while [ "${attempt}" -le "${max_attempts}" ]; do
        api_result=$(curl -sSL --connect-timeout "${curl_timeout}" --max-time "${curl_max_time}" "${GITHUB_API_BASE_URL}/repos/${REPO}/releases/latest" 2>/dev/null) || {
            log_warning "API请求失败，尝试第 ${attempt}/${max_attempts} 次"
            attempt=$((attempt + 1))
            [ "${attempt}" -le "${max_attempts}" ] && sleep 2
            continue
        }

        if command -v jq >/dev/null 2>&1; then
            VERSION=$(echo "${api_result}" | jq -r '.tag_name // empty')
        else
            VERSION=$(echo "${api_result}" | grep -o '"tag_name":[[:space:]]*"[^"]*"' | sed 's/"tag_name":[[:space:]]*"\([^"]*\)"/\1/')
        fi

        if [ -n "${VERSION}" ]; then
            log_success "最新版本: ${VERSION}"
            echo "${VERSION}"
            return 0
        fi

        log_warning "解析版本信息失败，尝试第 ${attempt}/${max_attempts} 次"
        attempt=$((attempt + 1))
        [ "${attempt}" -le "${max_attempts}" ] && sleep 2
    done

    log_error "获取版本信息失败"
    return 1
}

# 下载文件
download_file() {
    url="$1"
    output="$2"
    attempt=1
    max_attempts=3
    curl_timeout=10
    curl_max_time=30

    while [ "${attempt}" -le "${max_attempts}" ]; do
        if curl -sSL --connect-timeout "${curl_timeout}" --max-time "${curl_max_time}" -o "${output}" "${url}"; then
            if [ -s "${output}" ]; then
                return 0
            else
                log_warning "下载的文件为空，尝试第 ${attempt}/${max_attempts} 次"
            fi
        else
            log_warning "下载失败，尝试第 ${attempt}/${max_attempts} 次"
        fi
        attempt=$((attempt + 1))
        [ "${attempt}" -le "${max_attempts}" ] && sleep 2
    done

    log_error "下载失败: ${url}"
    return 1
}

# 验证配置文件
validate_config() {
    if [ ! -f "${CONFIG_DIR}/base.json" ]; then
        log_error "配置文件不存在: ${CONFIG_DIR}/base.json"
        return 1
    fi

    if [ ! -r "${CONFIG_DIR}/base.json" ]; then
        log_error "配置文件无读取权限"
        return 1
    fi

    if command -v jq >/dev/null 2>&1; then
        if ! jq empty "${CONFIG_DIR}/base.json" 2>/dev/null; then
            log_error "配置文件格式无效"
            return 1
        fi
    else
        # 基本的 JSON 格式检查
        if ! grep -q '^{.*}$' "${CONFIG_DIR}/base.json"; then
            log_error "配置文件格式无效"
            return 1
        fi
    fi

    return 0
}

# 下载二进制文件
download_binary() {
    binary_url="${GITHUB_BASE_URL}/${REPO}/releases/latest/download/aqua-speed-tools-${OS}-${ARCH}"
    binary_path="aqua-speed-tools"

    log_info "下载主程序中..."
    if ! download_file "${binary_url}" "${binary_path}"; then
        return 1
    fi

    if [ ! -s "${binary_path}" ]; then
        log_error "下载的二进制文件为空"
        return 1
    fi

    if ! chmod +x "${binary_path}"; then
        log_error "设置执行权限失败"
        return 1
    fi

    return 0
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
    VERSION="$1"
    cat <<"EOF"
    ___                        _____                     __   ______            __    
   /   | ____ ___  ______ _   / ___/____  ___  ___  ____/ /  /_  __/___  ____  / /____
  / /| |/ __ `/ / / / __ `/   \__ \/ __ \/ _ \/ _ \/ __  /    / / / __ \/ __ \/ / ___/
 / ___ / /_/ / /_/ / /_/ /   ___/ / /_/ /  __/  __/ /_/ /    / / / /_/ / /_/ / (__  ) 
/_/  |_\__, /\__,_/\__,_/   /____/ .___/\___/\___/\__,_/    /_/  \____/\____/_/____/  
         /_/                    /_/                                                   
EOF

    printf "\n%s仓库:%s https://github.com/%s\n" "$CYAN" "$NC" "${REPO}"
    if [ -n "${VERSION}" ]; then
        printf "%s版本:%s %s\n" "$CYAN" "$NC" "${VERSION}"
    fi
    printf "%s作者:%s Alice39s\n\n" "$CYAN" "$NC"
}

# 显示菜单
show_menu() {
    printf "%s请输入要执行选项的数字:%s\n" "$GREEN" "$NC"
    printf "1) %s列出所有节点%s\n" "$BOLD" "$NC"
    printf "2) %s测试指定节点%s\n" "$BOLD" "$NC"
    printf "3) %s退出%s\n" "$BOLD" "$NC"
}

# 列出节点并获取输入
list_and_get_input() {
    ./aqua-speed-tools list
    printf "\n%s请输入要测试的节点英文ID:%s\n" "$BLUE" "$NC"
    read -r node_id
    ./aqua-speed-tools test "${node_id}"
}

# 处理用户输入
handle_input() {
    read -r choice
    case "${choice}" in
    1)
        log_info "列出所有节点..."
        list_and_get_input
        ;;
    2)
        printf "\n%s请输入节点ID:%s\n" "$BLUE" "$NC"
        read -r node_id
        # 纯小写英文ID
        if ! echo "${node_id}" | grep -qE "^[a-z]+$"; then
            log_error "无效的节点ID，请输入英文ID"
            return 1
        fi
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
    VERSION=""
    # BusyBox sh 不支持 ERR trap，改为使用简单的 trap
    trap 'handle_error' INT TERM EXIT

    # 检查必要命令
    for cmd in curl grep sed jq; do
        if ! check_command "${cmd}"; then
            # 尝试安装缺失的命令
            confirm_install_command "${cmd}" || exit 1
        fi
    done

    # 初始化
    setup_temp_dir || exit 1
    setup_config_dir || exit 1
    detect_system || exit 1
    VERSION=$(get_latest_version) || exit 1
    if [ -n "${VERSION}" ]; then
        download_binary || exit 1
        download_config || exit 1
        validate_config || exit 1
    else
        log_error "获取最新版本失败"
        exit 1
    fi
    show_logo "${VERSION}"

    # 主循环
    while true; do
        show_menu
        handle_input || continue
    done
}

# 执行主函数
main
