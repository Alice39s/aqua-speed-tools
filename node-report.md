# Node Health Status Report

Generated on: 2025-07-12 18:06:58

## Configuration File Analysis

Configuration file: `presets/config.json`

## Test Results

| ID | Node Name | ISP | Type | ICMP Ping | TCP Ping | HTTP GET | 8-Thread GET | Notes |
|----|-----------|-----|------|-----------|----------|----------|--------------|-------|
| cf | Cloudflare (Cloudflare) | AS13335 (AS13335) | SingleFile | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | ICMP: ICMP ping timeout (10s) |
| zju | 浙江大学 (Zhejiang University) | 教育网 (CERNET) | LibreSpeed | ❌ FAIL | ❌ FAIL | ❌ FAIL | ❌ FAIL (0/8) | ICMP: ICMP ping timeout (10s); TCP: Port 80 connection timeout (3s); HTTP: HTTP request failed; Multi: Multi-thread test failed: only 0/8 threads succeeded |
| ustc | 中国科学技术大学 (USTC) | 教育网 (CERNET) | LibreSpeed | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | ICMP: ICMP ping timeout (10s) |
| nuaa | 南京航空航天大学 (NUAA) | 教育网 (CERNET) | LibreSpeed | ❌ FAIL | ❌ FAIL | ❌ FAIL | ❌ FAIL (0/8) | ICMP: ICMP ping timeout (10s); TCP: Port 80 connection failed; HTTP: HTTP request failed; Multi: Multi-thread test failed: only 0/8 threads succeeded |
| xcc | 四川西昌学院 (XCC) | 教育网 (CERNET) | LibreSpeed | ❌ FAIL | ❌ FAIL | ❌ FAIL | ❌ FAIL | ICMP: DNS resolution failed: [Errno -2] Name or service not known; TCP: DNS resolution failed: [Errno -2] Name or service not known; HTTP: DNS resolution failed: [Errno -2] Name or service not known; Multi: DNS resolution failed: [Errno -2] Name or service not known |
| baiduyun | 百度云盘 (Baidu Netdisk) | 百度云 (Baidu Cloud) | SingleFile | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | ICMP: ICMP ping timeout (10s) |
| bilibili | 哔哩哔哩 (Bilibili) | 阿里云 (Alibaba Cloud) | SingleFile | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | ICMP: ICMP ping timeout (10s) |
| arknights | 明日方舟 (Arknights) | 阿里云OSS (Alibaba Cloud OSS) | SingleFile | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | ICMP: ICMP ping timeout (10s) |
| sina | 新浪主站 (Sina) | 新浪混合云 (Sina CDN) | SingleFile | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | ICMP: ICMP ping timeout (10s) |
| wangyi | 网易主站 (Netease) | 网易混合云 (Netease CDN) | SingleFile | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | ICMP: ICMP ping timeout (10s) |
| danzai | 蛋仔派对 (DanZai) | 阿里云 (Alibaba Cloud) | SingleFile | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | ICMP: ICMP ping timeout (10s) |
| yuanshen | 原神官网 (Genshin Impact) | 阿里云 (Alibaba Cloud) | SingleFile | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (7/8) | ICMP: ICMP ping timeout (10s) |
| xqtd | 星穹铁道官网 (Star Rail) | 阿里云 (Alibaba Cloud) | SingleFile | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | ICMP: ICMP ping timeout (10s) |
| zzz | 绝区零 (Zenless Zone Zero) | 阿里云 (Alibaba Cloud) | SingleFile | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | ICMP: ICMP ping timeout (10s) |
| iqiyi | 爱奇艺 (iQIYI) | 爱奇艺 (iQIYI) | SingleFile | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | ICMP: ICMP ping timeout (10s) |
| xiaohongshu | 小红书 (RED) | 腾讯云 (Tencent Cloud CDN) | SingleFile | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | ICMP: ICMP ping timeout (10s) |
| migu | 咪咕快游 (MIGU Quick Game) | 中国移动云 (China Mobile Cloud) | SingleFile | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | ICMP: ICMP ping timeout (10s) |
| pdd | 拼多多 (Pinduoduo) | 网宿 (ChinaNetCenter) | SingleFile | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | ICMP: ICMP ping timeout (10s) |
| alipay | 支付宝 (Alipay) | 阿里云OSS (Alibaba Cloud OSS) | SingleFile | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | ICMP: ICMP ping timeout (10s) |
| weixin | 微信 (WeChat) | 上海腾讯云 (Tencent Cloud Shanghai) | SingleFile | ❌ FAIL | ✅ PASS | ✅ PASS | ✅ PASS (8/8) | ICMP: ICMP ping timeout (10s) |

## Statistics

- Total Nodes: 40
- Total Tests: 80
- Passed: 51
- Failed: 29
- Success Rate: 63%

## Health Status

🔴 **CRITICAL** - Success rate: 63%
