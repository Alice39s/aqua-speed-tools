# :ocean: Aqua Speed Tools

一个基于 Golang 的轻量级测速命令行工具，使用高性能的 [Aqua Speed][aqua-speed] 测速内核，内置多种 CDN 节点测试预设。

## :sparkles: 功能特点

- :arrows_counterclockwise: 内置多种 CDN 节点测试
- :rocket: 自动更新程序版本
- :bar_chart: 支持单节点或批量测试
- :art: 美观的表格输出格式
- :wrench: 多线程并发下载测试
- :electric_plug: 支持 Patch 动态更新 (TODO)

## :inbox_tray: 使用方式

### :book: 一键脚本

```bash
curl -fsSL https://raw.githubusercontent.com/alice39s/aqua-speed-tools/main/scripts/i.sh | bash
```

### :hammer_and_wrench: 从源码编译

#### :fork_and_knife: 克隆仓库

```bash
git clone https://github.com/alice39s/aqua-speed-tools.git
cd aqua-speed-tools
```

#### :hammer: 编译

```bash
# (Linux / macOS)
go build -o aqua-speed-tools cmd/speedtest/main.go

# (Windows)
go build -o aqua-speed-tools.exe cmd/speedtest/main.go
```

#### :running: 运行

```bash
# (Linux / macOS)
./aqua-speed-tools

# (Windows 需要带 .exe 后缀, 下文不再重复)
./aqua-speed-tools.exe
```

### :package: 下载预编译版本

也可以直接下载预编译版本：

| 平台                       | 架构  | 下载链接                  |
| :------------------------- | :---- | :------------------------ |
| :penguin: Linux            | amd64 | [点我下载][linux-amd64]   |
| :penguin: Linux            | arm64 | [点我下载][linux-arm64]   |
| :desktop_computer: Windows | amd64 | [点我下载][windows-amd64] |
| :desktop_computer: Windows | arm64 | ×[^1]                     |
| :apple: macOS              | amd64 | [点我下载][darwin-amd64]  |
| :apple: macOS              | arm64 | [点我下载][darwin-arm64]  |

[linux-amd64]: https://github.com/alice39s/aqua-speed-tools/releases/latest/download/aqua-speed-tools-linux-amd64
[linux-arm64]: https://github.com/alice39s/aqua-speed-tools/releases/latest/download/aqua-speed-tools-linux-arm64
[windows-amd64]: https://github.com/alice39s/aqua-speed-tools/releases/latest/download/aqua-speed-tools-windows-amd64.exe
[darwin-amd64]: https://github.com/alice39s/aqua-speed-tools/releases/latest/download/aqua-speed-tools-darwin-amd64
[darwin-arm64]: https://github.com/alice39s/aqua-speed-tools/releases/latest/download/aqua-speed-tools-darwin-arm64

### :gear: 配置

程序会自动在以下位置创建配置文件:

- Windows: `%APPDATA%/aqua-speed/config.json`
- Linux: `/etc/aqua-speed/config.json`
- MacOS: `~/Library/Application Support/aqua-speed/config.json`

## :memo: 配置文件

配置文件是一个 JSON 文件，你可以根据需要自行编辑。

### :clipboard: 格式

配置文件中每个节点包含以下字段：

#### 基本信息

| 字段      | 说明             | 类型     | 示例           |
| :-------- | :--------------- | :------- | :------------- |
| `节点ID`  | 节点的唯一标识符 | `string` | `"cf"`         |
| `name.zh` | 节点中文名称     | `string` | `"Cloudflare"` |
| `name.en` | 节点英文名称     | `string` | `"Cloudflare"` |
| `size`    | 测试文件大小(MB) | `number` | `100`          |

#### ISP 信息

| 字段     | 说明        | 类型     | 示例        |
| :------- | :---------- | :------- | :---------- |
| `isp.zh` | ISP中文名称 | `string` | `"AS13335"` | **** |
| `isp.en` | ISP英文名称 | `string` | `"AS13335"` |

#### 测试配置

| 字段      | 说明                                                   | 类型     | 示例                              |
| :-------- | :----------------------------------------------------- | :------- | :-------------------------------- |
| `url`     | 测试URL<br>*单文件测试需填写具体文件URL*               | `string` | `"https://speed.cloudflare.com/"` |
| `threads` | 并发测试线程数                                         | `number` | `10`                              |
| `type`    | 测试类型:<br>`SingleFile`/`LibreSpeed`/`Ookla`(开发中) | `string` | `"SingleFile"`                    |

#### 地理位置

| 字段                  | 说明                                       | 类型          | 示例        |
| :-------------------- | :----------------------------------------- | :------------ | :---------- |
| `geoInfo.countryCode` | 国家ISO-3166-1代码<br>*Anycast节点请填UN*  | `string`      | `"UN"`      |
| `geoInfo.region`      | 地区<br>*Anycast节点请填null*              | `string/null` | `null`      |
| `geoInfo.city`        | 城市<br>*Anycast节点请填null*              | `string/null` | `null`      |
| `geoInfo.type`        | 节点类型:<br>`Anycast`/`CDN`/`IDC`/`OSS`等 | `string`      | `"Anycast"` |

### :pushpin: 配置文件示例

```json
{
    "cf": {
        "name": {
            "zh": "Cloudflare",
            "en": "Cloudflare"
        },
        "size": 40,
        "isp": {
            "zh": "AS13335",
            "en": "AS13335"
        },
        "url": "https://speed.cloudflare.com/",
        "threads": 10,
        "type": "SingleFile",
        "geoInfo": {
            "countryCode": "US",
            "region": null,
            "city": null,
            "type": "Anycast"
        }
    }
}
```

## :rocket: 使用方法

```bash
# 列出所有可用节点
./aqua-speed-tools list

# 测试指定节点速度
./aqua-speed-tools test <节点英文ID>

# 测试所有节点 (不推荐)
./aqua-speed-tools test all
```

## :wrench: 技术栈

- Go 1.22.10+
- cobra (命令行框架)
- go-pretty (表格输出)
- zap (日志记录)

## :clipboard: TODO

- :dizzy: 支持将结果上传到服务器，并生成一个易于分享的网页和 OpenGraph 图片
- :bar_chart: list 和 test 命令输出为 Markdown, CSV, JSON 等格式
- :arrows_counterclockwise: 支持 Patch 动态更新
- :art: 优化表格输出
- :speech_balloon: 多语言支持

## :page_facing_up: 许可证

本项目采用 [AGPL-3.0](LICENSE) 开源许可证。

[aqua-speed]: https://github.com/alice39s/aqua-speed

[^1]: 由于测速客户端主程序 [aqua-speed] 使用 Bun 编写，而 Bun 暂不支持 Linux 交叉编译至 Windows ARM64 架构，如有需要，请自行 [编译安装](#hammer_and_wrench-从源码编译)。
