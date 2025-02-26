#!/bin/sh
# 设置严格模式
set -eu

# 检查是否支持彩色输出
if [ -t 1 ] && command -v tput >/dev/null 2>&1; then
    RED=$(tput setaf 1)
    GREEN=$(tput setaf 2)
    BLUE=$(tput setaf 4)
    YELLOW=$(tput setaf 3)
    CYAN=$(tput setaf 6)
    GRAY=$(tput setaf 7)
    BOLD=$(tput bold)
    NC=$(tput sgr0)
else
    RED=""
    GREEN=""
    BLUE=""
    YELLOW=""
    CYAN=""
    GRAY=""
    BOLD=""
    NC=""
fi

log_debug() {
    if [ "${DEBUG_MODE}" -eq 1 ]; then
        printf "%s[DEBUG]%s %s\n" "$GRAY" "$NC" "$1" >&2
    fi
}

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

handle_error() {
    _exit_code=$?
    log_error "脚本发生错误，错误码: ${_exit_code}"
    exit "${_exit_code}"
}

VERSION="1.0.0"
REPO="alice39s/aqua-speed-tools"
USER_AGENT="aqua-speed-tools-script/${VERSION}"
DEBUG_MODE=0

show_help() {
    cat <<EOF
Usage: $0 [options]

Options:
    -h, --help              显示帮助信息
    -s, --smart-url URL     设置 SMART_BASE_URL
    -d, --debug             开启 debug 模式
EOF
}

SMART_BASE_URL=""
while [ $# -gt 0 ]; do
    case "$1" in
    -h | --help)
        show_help
        exit 0
        ;;
    -s | --smart-url)
        if [ -n "$2" ]; then
            SMART_BASE_URL="$2"
            shift 2
        else
            log_error "Error: --smart-url 需要一个URL参数"
            exit 1
        fi
        ;;
    -d | --debug)
        DEBUG_MODE=1
        shift
        ;;
    *)
        log_error "未知参数: $1"
        show_help
        exit 1
        ;;
    esac
done

RAW_BASE_URL=${SMART_BASE_URL:+"$SMART_BASE_URL/raw"}
RAW_BASE_URL=${RAW_BASE_URL:-"https://raw.githubusercontent.com"}

GITHUB_BASE_URL=${SMART_BASE_URL:+"$SMART_BASE_URL/base"}
GITHUB_BASE_URL=${GITHUB_BASE_URL:-"https://github.com"}

GITHUB_API_BASE_URL=${SMART_BASE_URL:+"$SMART_BASE_URL/api"}
GITHUB_API_BASE_URL=${GITHUB_API_BASE_URL:-"https://api.github.com"}

CONFIG_JSON_URL="$RAW_BASE_URL/$REPO/main/configs/base.json"
CONFIG_URL="${AQUA_SPEED_CONFIG_URL:-$CONFIG_JSON_URL}"
CONFIG_DIR="configs"
TEMP_DIR=""
EXIT_CODE=0

# 清理函数
cleanup() {
    _exit_status=$?
    [ "${_exit_status}" -ne 0 ] && EXIT_CODE=${_exit_status}

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

# 检查命令是否存在
check_command() {
    _cmd=$1
    if ! command -v "${_cmd}" >/dev/null 2>&1; then
        log_error "未找到命令: ${_cmd}"
        return 1
    fi
    return 0
}

# 检查文件权限和状态
check_file_access() {
    _path=$1
    _type=$2             # file 或 directory
    _check_write=${3:-0} # 是否检查写入权限，默认不检查

    if [ "${_type}" = "file" ]; then
        if [ ! -f "${_path}" ]; then
            log_error "文件不存在: ${_path}"
            return 1
        fi
        if [ ! -r "${_path}" ]; then
            log_error "文件无读取权限: ${_path}"
            return 1
        fi
    elif [ "${_type}" = "directory" ]; then
        if [ ! -d "${_path}" ]; then
            if ! mkdir -p "${_path}"; then
                log_error "创建目录失败: ${_path}"
                return 1
            fi
        fi
    fi

    if [ "${_check_write}" -eq 1 ] && [ ! -w "${_path}" ]; then
        log_error "${_type}无写入权限: ${_path}"
        return 1
    fi

    return 0
}

# 检查命令是否存在并尝试安装
check_and_install_command() {
    _cmd=$1
    _auto_install=${2:-0} # 是否自动安装，默认不自动安装

    if command -v "${_cmd}" >/dev/null 2>&1; then
        return 0
    fi

    if [ "${_auto_install}" -eq 0 ]; then
        log_error "未找到命令: ${_cmd}"
        return 1
    fi

    # 检查是否有root权限
    if [ "$(id -u)" -ne 0 ]; then
        log_error "安装命令需要root权限，请使用sudo运行此脚本"
        return 1
    fi

    log_warning "未找到命令: ${_cmd}, 尝试安装..."
    _package_managers="apt:apt install -y|yum:yum install -y|brew:brew install|pacman:pacman -S --noconfirm|apk:apk add --no-cache"
    _found_manager=0
    _install_success=0

    _OLD_IFS=$IFS
    IFS="|"
    for _pair in ${_package_managers}; do
        IFS=":"
        set -- ${_pair}
        _manager=$1
        _install_cmd=$2
        IFS=$_OLD_IFS

        if command -v "${_manager}" >/dev/null 2>&1; then
            _found_manager=1
            if eval "${_install_cmd} ${_cmd}" >/dev/null 2>&1; then
                if command -v "${_cmd}" >/dev/null 2>&1; then
                    log_success "${_cmd} 安装成功"
                    _install_success=1
                    break
                fi
            fi
        fi
    done

    IFS=$_OLD_IFS

    if [ "${_found_manager}" -eq 0 ]; then
        log_error "未找到支持的包管理器"
        return 1
    fi

    if [ "${_install_success}" -eq 0 ]; then
        log_error "安装失败，请尝试手动安装 ${_cmd}"
        return 1
    fi

    return 0
}

# 检查并创建临时目录
setup_temp_dir() {
    check_and_install_command "mktemp" || return 1

    TEMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t 'aquaspeed')
    if ! check_file_access "${TEMP_DIR}" "directory" 1; then
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
    check_file_access "${CONFIG_DIR}" "directory" 1
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

    if [ -n "${SMART_BASE_URL}" ]; then
        printf "%s正在使用镜像:%s %s\n" "$CYAN" "$NC" "${SMART_BASE_URL}"
    fi

    return 0
}

# 通用的 curl 请求函数
make_curl_request() {
    _url=$1
    _output=$2
    _attempt=1
    _max_attempts=3
    _curl_timeout=10
    _curl_max_time=30
    _is_api_call=0

    # 如果第三个参数存在且为1，则为API调用
    if [ $# -ge 3 ] && [ "$3" -eq 1 ]; then
        _is_api_call=1
        _curl_max_time=15
    fi

    log_debug "正在请求 ${_url}"

    while [ "${_attempt}" -le "${_max_attempts}" ]; do
        if [ "${_is_api_call}" -eq 1 ]; then
            log_debug "正在请求 API ${_url}"
            _response=$(curl -sL --retry 2 --retry-delay 1 \
                --connect-timeout "${_curl_timeout}" \
                --max-time "${_curl_max_time}" \
                -H "User-Agent: ${USER_AGENT}" \
                "${_url}" 2>/dev/null)
            _curl_exit_code=$?
        else
            if [ -n "${_output}" ]; then
                log_debug "正在下载 ${_url} 到 ${_output}"
                curl -sSL --connect-timeout "${_curl_timeout}" \
                    --max-time "${_curl_max_time}" \
                    -H "User-Agent: ${USER_AGENT}" \
                    -o "${_output}" "${_url}"
                _curl_exit_code=$?
            else
                _response=$(curl -sSL --connect-timeout "${_curl_timeout}" \
                    --max-time "${_curl_max_time}" \
                    -H "User-Agent: ${USER_AGENT}" \
                    "${_url}")
                _curl_exit_code=$?
            fi
        fi

        if [ "${_curl_exit_code}" -eq 0 ]; then
            if [ "${_is_api_call}" -eq 1 ] || [ -z "${_output}" ]; then
                echo "${_response}"
                return 0
            elif [ -s "${_output}" ]; then
                return 0
            fi
        fi

        log_warning "请求失败，尝试第 ${_attempt}/${_max_attempts} 次"
        _attempt=$((_attempt + 1))
        [ "${_attempt}" -le "${_max_attempts}" ] && sleep 2
    done

    log_error "请求失败: ${_url}"
    return 1
}

# 获取最新版本
get_latest_version() {
    api_result=$(make_curl_request "${GITHUB_API_BASE_URL}/repos/${REPO}/releases/latest" "" 1) || return 1

    if command -v jq >/dev/null 2>&1; then
        VERSION=$(echo "${api_result}" | jq -r '.tag_name // empty')
    fi
    # 如果版本号为空，则使用 grep 解析
    if [ -z "${VERSION}" ]; then
        log_debug "jq 解析失败，使用 grep 解析版本信息"
        VERSION=$(echo "${api_result}" | grep -o '"tag_name":[[:space:]]*"[^"]*"' | sed 's/"tag_name":[[:space:]]*"\([^"]*\)"/\1/')
    fi

    if [ -n "${VERSION}" ]; then
        log_success "最新版本: ${VERSION}"
        echo "${VERSION}"
        return 0
    fi

    log_debug "API 结果: ${api_result}"

    log_error "解析版本信息失败"
    return 1
}

# 下载文件
download_file() {
    url="$1"
    output="$2"

    if ! make_curl_request "${url}" "${output}"; then
        log_error "下载失败: ${url}"
        return 1
    fi
    return 0
}

# 验证配置文件
validate_config() {
    _config_file="${CONFIG_DIR}/base.json"

    if ! check_file_access "${_config_file}" "file"; then
        return 1
    fi

    if check_and_install_command "jq" 0; then
        if ! jq empty "${_config_file}" 2>/dev/null; then
            log_error "配置文件格式无效"
            return 1
        fi
    else
        # 基本的 JSON 格式检查
        if ! grep -q '^{.*}$' "${_config_file}"; then
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
    # 检查二进制文件
    if [ ! -f "./aqua-speed-tools" ] || [ ! -x "./aqua-speed-tools" ]; then
        log_error "下载失败，请检查网络连接"
        return 1
    fi

    # 列出所有节点
    if ! ./aqua-speed-tools \
        --github-base-url "${GITHUB_BASE_URL}" \
        --github-raw-base-url "${RAW_BASE_URL}" \
        --github-api-base-url "${GITHUB_API_BASE_URL}" \
        list; then
        log_error "列出节点失败"
        return 1
    fi

    # 获取用户输入
    printf "\n%s请输入要测试的节点 ID (支持数字序号或英文ID):%s " "$BLUE" "$NC"
    read -r node_id

    # 验证输入
    if [ -z "${node_id}" ]; then
        log_error "节点ID不能为空"
        return 1
    fi

    # 允许数字、字母和组合ID
    if ! echo "${node_id}" | grep -qE "^[0-9a-zA-Z_-]+$"; then
        log_error "无效的节点ID，请输入有效的序号或ID"
        return 1
    fi

    # 执行测试
    log_info "正在测试节点 ${node_id}..."
    if ! ./aqua-speed-tools \
        --github-base-url "${GITHUB_BASE_URL}" \
        --github-raw-base-url "${RAW_BASE_URL}" \
        --github-api-base-url "${GITHUB_API_BASE_URL}" \
        test "${node_id}"; then
        log_error "测试节点 ${node_id} 失败"
        return 1
    fi
    
    return 0
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
        printf "\n%s请输入节点 ID (支持数字序号或英文ID):%s\n" "$BLUE" "$NC"
        read -r node_id
        # 允许数字和英文ID
        if ! echo "${node_id}" | grep -qE "^[0-9]+$|^[a-z]+$"; then
            log_error "无效的节点ID，请输入数字序号或英文ID"
            return 1
        fi
        ./aqua-speed-tools \
            --github-base-url "${GITHUB_BASE_URL}" \
            --github-raw-base-url "${RAW_BASE_URL}" \
            --github-api-base-url "${GITHUB_API_BASE_URL}" \
            test "${node_id}"
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
            check_and_install_command "${cmd}" || exit 1
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
