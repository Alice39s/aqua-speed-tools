# Node Health Status Report

Generated on: 2026-09-05 18:08:52

## Configuration File Analysis

Configuration file: `presets/config.json`

## Test Results

| ID | Node Name | ISP | Type | ICMP Ping | TCP Ping | HTTP GET | 8-Thread GET | Notes |
|----|-----------|-----|------|-----------|----------|----------|--------------|-------|
| cf | Cloudflare (Cloudflare) | AS13335 (AS13335) | SingleFile | ✅ PASS (2/3 nodes OK) | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | All tests passed |
| ustc | 中国科学技术大学 (USTC) | 教育网 (CERNET) | LibreSpeed | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | ICMP: Check-Host result polling timeout |
| nuaa | 南京航空航天大学 (NUAA) | 教育网 (CERNET) | LibreSpeed | ✅ PASS (2/3 nodes OK) | ❌ FAIL | ❌ FAIL | ❌ FAIL (0/8) | TCP: Port 80 connection failed; HTTP: HTTP request failed; Multi: Multi-thread test failed: only 0/8 threads succeeded |
| xcc | 四川西昌学院 (XCC) | 教育网 (CERNET) | LibreSpeed | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (6/8) | ICMP: Check-Host result polling timeout |
| baiduyun | 百度云盘 (Baidu Netdisk) | 百度云 (Baidu Cloud) | SingleFile | ✅ PASS (2/3 nodes OK) | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | All tests passed |
| bilibili | 哔哩哔哩 (Bilibili) | 阿里云 (Alibaba Cloud) | SingleFile | ✅ PASS (3/3 nodes OK) | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | All tests passed |
| arknights | 明日方舟 (Arknights) | 阿里云OSS (Alibaba Cloud OSS) | SingleFile | ✅ PASS (2/3 nodes OK) | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | All tests passed |
| sina | 新浪主站 (Sina) | 新浪混合云 (Sina CDN) | SingleFile | ✅ PASS (2/3 nodes OK) | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | All tests passed |
| wangyi | 网易主站 (Netease) | 网易混合云 (Netease CDN) | SingleFile | ✅ PASS (2/3 nodes OK) | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | All tests passed |
| danzai | 蛋仔派对 (DanZai) | 阿里云 (Alibaba Cloud) | SingleFile | ✅ PASS (2/3 nodes OK) | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | All tests passed |
| yuanshen | 原神官网 (Genshin Impact) | 阿里云 (Alibaba Cloud) | SingleFile | ✅ PASS (1/3 nodes OK) | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | All tests passed |
| xqtd | 星穹铁道官网 (Star Rail) | 阿里云 (Alibaba Cloud) | SingleFile | ✅ PASS (2/3 nodes OK) | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | All tests passed |
| zzz | 绝区零 (Zenless Zone Zero) | 阿里云 (Alibaba Cloud) | SingleFile | ✅ PASS (3/3 nodes OK) | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | All tests passed |
| iqiyi | 爱奇艺 (iQIYI) | 爱奇艺 (iQIYI) | SingleFile | ✅ PASS (1/3 nodes OK) | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | All tests passed |
| xiaohongshu | 小红书 (RED) | 腾讯云 (Tencent Cloud CDN) | SingleFile | ✅ PASS (3/3 nodes OK) | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | All tests passed |
| migu | 咪咕快游 (MIGU Quick Game) | 中国移动云 (China Mobile Cloud) | SingleFile | ✅ PASS (1/3 nodes OK) | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | All tests passed |
| pdd | 拼多多 (Pinduoduo) | 网宿 (ChinaNetCenter) | SingleFile | ✅ PASS (3/3 nodes OK) | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | All tests passed |
| alipay | 支付宝 (Alipay) | 阿里云OSS (Alibaba Cloud OSS) | SingleFile | ✅ PASS (2/3 nodes OK) | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | All tests passed |
| weixin | 微信 (WeChat) | 上海腾讯云 (Tencent Cloud Shanghai) | SingleFile | ✅ PASS (2/3 nodes OK) | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | All tests passed |

## Statistics

- Total Nodes: 19
- Total Tests: 76
- Passed: 71
- Failed: 5
- Success Rate: 93%

## Health Status

🟢 **HEALTHY** - Success rate: 93%
