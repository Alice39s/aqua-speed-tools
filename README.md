# :ocean: Aqua Speed Tools

一个基于 Golang 的轻量级测速命令行工具，使用高性能的 [Aqua Speed][aqua-speed] 测速内核，内置多种 CDN 节点测试预设。

## :sparkles: 功能特点

- :arrows_counterclockwise: 内置多种 CDN 节点测试
- :rocket: 自动更新程序版本
- :bar_chart: 支持单节点或批量测试
- :art: 美观的表格输出格式
- :wrench: 多线程并发下载测试
- :electric_plug: 支持 Patch 动态更新 (TODO)
- :globe_with_meridians: 支持自定义镜像源
- :shield: 支持 DNS over HTTPS

## :inbox_tray: 安装方式

### :package: 下载预编译版本

#### Linux :penguin: / MacOS :apple:

```bash
# 需安装 curl
curl -sL "https://github.com/alice39s/aqua-speed-tools/releases/latest/download/aqua-speed-tools-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')" -o aqua-speed-tools

# 国内用户可选镜像
curl -sL "https://s3-lb01.000000039.xyz/download/Alice39s/aqua-speed-tools/latest/download/aqua-speed-tools-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')" -o aqua-speed-tools

chmod +x aqua-speed-tools

./aqua-speed-tools
```

#### Windows :computer:

```bash
# 64位, 需安装 curl
curl -sL https://github.com/alice39s/aqua-speed-tools/releases/latest/download/aqua-speed-tools-windows-amd64.exe -o aqua-speed-tools.exe

# 国内用户可选镜像
curl -sL https://s3-lb01.000000039.xyz/download/Alice39s/aqua-speed-tools/latest/download/aqua-speed-tools-windows-amd64.exe -o aqua-speed-tools.exe

./aqua-speed-tools.exe
```

各平台预编译版本下载链接：

| 平台                       | 架构  | 下载链接                                                    |
| :------------------------- | :---- | :---------------------------------------------------------- |
| :penguin: Linux            | amd64 | [GitHub 源][linux-amd64] / [镜像源][linux-amd64-mirror]     |
| :penguin: Linux            | arm64 | [GitHub 源][linux-arm64] / [镜像源][linux-arm64-mirror]     |
| :desktop_computer: Windows | amd64 | [GitHub 源][windows-amd64] / [镜像源][windows-amd64-mirror] |
| :desktop_computer: Windows | arm64 | ×[^1]                                                       |
| :apple: macOS              | amd64 | [GitHub 源][darwin-amd64] / [镜像源][darwin-amd64-mirror]   |
| :apple: macOS              | arm64 | [GitHub 源][darwin-arm64] / [镜像源][darwin-arm64-mirror]   |

### :hammer_and_wrench: 从源码编译

```bash
# 克隆仓库
git clone https://github.com/alice39s/aqua-speed-tools.git
cd aqua-speed-tools

# 编译
go build -o aqua-speed-tools cmd/tools/main.go
```

## :rocket: 使用方法

### :computer: 交互式模式

直接运行程序即可进入交互式模式：

```bash
./aqua-speed-tools
```

### :keyboard: 命令行模式

```bash
# 列出所有可用节点
./aqua-speed-tools list

# 测试指定节点速度
./aqua-speed-tools test <节点ID>
```

### :gear: 高级选项

```bash
# 开启调试模式
./aqua-speed-tools -d

# 使用自定义 GitHub Raw 镜像
./aqua-speed-tools --github-raw-magic-url https://raw.example.com

# 使用自定义 GitHub API 镜像
./aqua-speed-tools --github-api-magic-url https://api.example.com

# 使用自定义 DNS over HTTPS 端点
./aqua-speed-tools --doh-endpoint https://doh.pub/dns-query

# 查看帮助
./aqua-speed-tools -h
```

## :wrench: 配置文件

程序会自动在以下位置创建配置文件:

- Windows: `%APPDATA%/aqua-speed/config.json`
- Linux: `/etc/aqua-speed/config.json`
- MacOS: `~/Library/Application Support/aqua-speed/config.json`

### :clipboard: 配置格式

配置文件包含以下主要部分：

#### 基本配置

| 字段              | 说明               | 类型     | 示例                 |
| :---------------- | :----------------- | :------- | :------------------- |
| `script.version`  | 程序版本号         | `string` | `"3.0.0"`            |
| `script.prefix`   | 程序前缀           | `string` | `"aqua-speed-tools"` |
| `downloadTimeout` | 下载超时时间（秒） | `number` | `30`                 |

#### GitHub 配置

| 字段                   | 说明         | 类型       | 示例                                    |
| :--------------------- | :----------- | :--------- | :-------------------------------------- |
| `githubRepo`           | 主仓库       | `string`   | `"alice39s/aqua-speed"`                 |
| `githubToolsRepo`      | 工具仓库     | `string`   | `"alice39s/aqua-speed-tools"`           |
| `github_raw_magic_set` | Raw 镜像列表 | `string[]` | `["https://raw.githubusercontent.com"]` |

#### DNS over HTTPS 配置

| 字段                 | 说明           | 类型       | 示例       |
| :------------------- | :------------- | :--------- | :--------- |
| `dns_over_https_set` | DoH 服务器配置 | `object[]` | 见下方示例 |

每个 DoH 配置包含：

| 字段       | 说明         | 类型     | 示例                                     |
| :--------- | :----------- | :------- | :--------------------------------------- |
| `endpoint` | 服务器端点   | `string` | `"https://cloudflare-dns.com/dns-query"` |
| `timeout`  | 超时时间(秒) | `number` | `10`                                     |
| `retries`  | 重试次数     | `number` | `3`                                      |

### :pushpin: 配置示例

```json
{
  "script": {
    "version": "3.0.0",
    "prefix": "aqua-speed-tools"
  },
  "github_raw_magic_set": [
    "https://raw.githubusercontent.com",
    "https://raw.fastgit.org",
    "https://raw.staticdn.net",
    "https://raw.githubusercontents.com"
  ],
  "dns_over_https_set": [
    {
      "endpoint": "https://cloudflare-dns.com/dns-query",
      "timeout": 10,
      "retries": 3
    },
    {
      "endpoint": "https://dns.google/dns-query",
      "timeout": 10,
      "retries": 3
    }
  ],
  "downloadTimeout": 30,
  "githubRepo": "alice39s/aqua-speed",
  "githubToolsRepo": "alice39s/aqua-speed-tools"
}
```

## :clipboard: TODO

- [ ] :dizzy: 支持将结果上传到服务器，并生成一个易于分享的网页和 OpenGraph 图片
- [ ] :bar_chart: list 和 test 命令输出为 Markdown, CSV, JSON 等格式
- [ ] :arrows_counterclockwise: 支持 Patch 动态更新
- [x] :art: 优化表格输出
- [ ] :speech_balloon: 多语言支持
- [x] :shield: 支持 DNS over HTTPS

## :page_facing_up: 许可证

本项目采用 [AGPL-3.0](LICENSE) 开源许可证。

[aqua-speed]: https://github.com/alice39s/aqua-speed
[linux-amd64]: https://github.com/alice39s/aqua-speed-tools/releases/latest/download/aqua-speed-tools-linux-amd64
[linux-arm64]: https://github.com/alice39s/aqua-speed-tools/releases/latest/download/aqua-speed-tools-linux-arm64
[windows-amd64]: https://github.com/alice39s/aqua-speed-tools/releases/latest/download/aqua-speed-tools-windows-amd64.exe
[darwin-amd64]: https://github.com/alice39s/aqua-speed-tools/releases/latest/download/aqua-speed-tools-darwin-amd64
[darwin-arm64]: https://github.com/alice39s/aqua-speed-tools/releases/latest/download/aqua-speed-tools-darwin-arm64
[linux-amd64-mirror]: https://s3-lb01.000000039.xyz/download/Alice39s/aqua-speed-tools/latest/download/aqua-speed-tools-linux-amd64
[linux-arm64-mirror]: https://s3-lb01.000000039.xyz/download/Alice39s/aqua-speed-tools/latest/download/aqua-speed-tools-linux-arm64
[windows-amd64-mirror]: https://s3-lb01.000000039.xyz/download/Alice39s/aqua-speed-tools/latest/download/aqua-speed-tools-windows-amd64.exe
[darwin-amd64-mirror]: https://s3-lb01.000000039.xyz/download/Alice39s/aqua-speed-tools/latest/download/aqua-speed-tools-darwin-amd64
[darwin-arm64-mirror]: https://s3-lb01.000000039.xyz/download/Alice39s/aqua-speed-tools/latest/download/aqua-speed-tools-darwin-arm64

[^1]: 由于测速客户端主程序 [aqua-speed] 使用 Bun 编写，而 Bun 暂不支持 Linux 交叉编译至 Windows ARM64 架构，如有需要，请自行 [编译安装](#hammer_and_wrench-从源码编译)。
