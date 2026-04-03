# Xray Panel

一个用 Go 编写的轻量 Xray 管理面板，支持 WebSocket 实时流量监控、用户管理、访问日志查看、Telegram 机器人控制，以及 VLESS+XHTTP+Reality / CDN 协议的订阅链接导出。

---

> **免责声明**：本项目代码主要由 AI 辅助生成，请自行审阅后使用。

## 快速开始

需要 Go 1.24 及以上版本。

克隆项目：

```bash
git clone https://github.com/coding-leaf/X-CorelittlePanel.git
cd X-CorelittlePanel
```

Windows 交叉编译 Linux 二进制：

```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o xray-panel-linux ./cmd/panel
```

Linux 本地编译：

```bash
go build -o xray-panel-linux ./cmd/panel
```

配置并运行：

```bash
cp config.example.json config.json
nano config.json
./xray-panel-linux
```

## 基础功能介绍

**流量监控**：支持查看每个用户的上行/下行流量，支持图表历史记录，数据自动持久化到本地文件，重启后不丢失。

**系统信息**：展示服务器 CPU、内存、磁盘、运行时长等指标，以及linux系统内部运行数据。

**日志查看**：在线浏览 Xray access log 和 error log，支持过滤和清除。

**Xray 配置管理**：通过 Web 界面直接读写 Xray 的 config.json，支持语法校验、备份恢复，修改后可重载/重启 Xray 服务。

**用户管理**：添加、删除 Xray 入站用户。

**订阅链接**：为每个用户生成 VLESS 订阅链接(配置后仅支持xhttp+reality/cdn配置,请自行探索)

**Telegram Bot**：配置后支持完整双向命令交互，可通过 Bot 查询流量、系统状态、网络延迟、证书有效期，以及远程重启/重载 Xray、生成订阅链接，同时推送证书到期告警。

**公开仪表盘**：可选开启只读公开页面，无需登录即可查看汇总流量数据。

**HTTPS 自动切换**：根据配置自动选择合法证书、Let's Encrypt 自动签发，或生成内存自签名证书保障传输加密,保底安全措施

---

## 已知限制

订阅链接目前只支持 VLESS+XHTTP+reality/cdn 入站类型，其他协议（VMess、Trojan 等）不会生成订阅链接。

面板是单管理员设计，所有功能共用同一个登录密码，不支持多账户权限分级。

Xray 必须在启动时已开启 gRPC API 入站（`dokodemo-door` 监听 `127.0.0.1:10085`），否则流量数据无法读取。

---

## config.json 字段说明

将 `config.example.json` 复制为 `config.json` 并按实际情况填写。
| 字段名 | 说明 | 示例 | 是否必填 |
|---|---|---|---|
| password | 主面板以及管理面板的登录密码 | 高强度随机字符串 | 必填 |
| listen_addr | 本面板部署监听地址，监听内网时默认 HTTP，监听公网时启用 HTTPS，默认 127.0.0.1:8880 | 127.0.0.1:8880 | 可选 |
| xray_api | Xray gRPC 接口地址，需与 Xray config.json 中的 api 入站监听地址一致，默认 127.0.0.1:10085 | 127.0.0.1:10085 | 可选 |
| access_log | Xray access log 文件的绝对路径，留空则不展示访问日志 | /var/log/xray/access.log | 可选 |
| error_log | Xray error log 文件的绝对路径，留空则不展示错误日志 | /var/log/xray/error.log | 可选 |
| xray_config_path | Xray 主配置文件路径，用于在线读写配置和生成订阅链接，留空则相关功能不可用 | /usr/local/etc/xray/config.json | 建议填写 |
| xray_bin_path | Xray 可执行文件路径，面板用于重启 Xray 进程，留空则无法通过面板重启 | /usr/local/bin/xray | 建议填写 |
| cert_path | Xray TLS 证书路径，仅用于在订阅链接附加证书信息，非面板自身 TLS | /usr/local/etc/xray/xray.crt | 可选 |
| traffic_data_path | 流量历史数据持久化文件路径，默认 traffic_data.json | traffic_data.json | 可选 |
| reality_addr | Reality 节点服务器 IP 或域名，用于生成 Reality 模式订阅链接中的连接地址 | 1.2.3.4 | 使用 Reality 订阅时必填 |
| reality_public_key | Reality 协议公钥，写入订阅链接的 `pbk` 参数。运行 `xray x25519` 后填入输出中的 `Password` 字段（即公钥） | 参考[命令参数](https://xtls.github.io/document/command.html#xray-vlessenc) | 使用 Reality 订阅时必填 |
| cdn_domain | CDN 模式订阅链接使用的域名，即 CDN 回源域名 | cdn.example.com | 使用 CDN 订阅时必填 |
| cdn_encryption | CDN 节点传输加密方式。填入 `xray vlessenc` 生成的 `encryption` 字段值，服务端填对应的 `decryption` 值，两端认证方式必须一致。官方参考 [VLESS 加密理念](https://github.com/XTLS/Xray-core/pull/5067) | 参考[命令参数](https://xtls.github.io/document/command.html#xray-vlessenc) | 使用 CDN 订阅时建议填写 |
| node_port | 订阅链接中节点端口，默认 443 | 443 | 可选 |
| node_fp | TLS 指纹，写入订阅链接，默认 random | chrome | 可选 |
| node_alpn | ALPN 参数，写入 CDN 模式订阅链接 | h2 | 可选,不推荐http1 |
| telegram_token | Telegram Bot API Token，留空则禁用通知功能 | 123456:ABC... | 使用 Telegram 通知时必填 |
| telegram_chat_id | 接收通知的 Telegram Chat ID | -100123456 | 使用 Telegram 通知时必填 |
| public_dashboard | 设为 true 后面板首页无需登录可访问 | false | 可选 |
| tls_cert_file | 面板自身 HTTPS 证书文件路径，与 tls_key_file 配合，优先级最高 | /etc/ssl/cert.pem | 三选一 TLS 方案 |
| tls_key_file | 面板自身 HTTPS 私钥文件路径 | /etc/ssl/key.pem | 三选一 TLS 方案 |
| domain | 填入后启用 Let's Encrypt 自动证书，需要 80/443 端口可被公网访问 | panel.example.com | 三选一 TLS 方案 |

---

## 部署方式

### 前置要求

- 已安装并运行 Xray，且 Xray config.json 中启用了 gRPC API（默认端口 10085）：

```json
{
  "api": {
    "tag": "api",
    "services": ["StatsService", "HandlerService"]
  },
  "stats": {},
  "policy": {
    "levels": { "0": { "statsUserUplink": true, "statsUserDownlink": true } },
    "system": { "statsInboundUplink": true, "statsInboundDownlink": true }
  },
  "inbounds": [
    {
      "tag": "api",
      "listen": "127.0.0.1",
      "port": 10085,
      "protocol": "dokodemo-door",
      "settings": { "address": "127.0.0.1" }
    }
  ]
}
```

### 下载与配置

```bash
# 下载编译好的二进制（Linux amd64）
wget https://github.com/coding-leaf/X-CorelittlePanel/releases/latest/download/xray-panel-linux
chmod +x xray-panel-linux

# 复制配置文件
cp config.example.json config.json

# 编辑配置
nano config.json
```

### 运行

```bash
./xray-panel-linux
```

### 使用 systemd 托管（推荐）

```ini
[Unit]
Description=Xray Panel
After=network.target

[Service]
WorkingDirectory=/opt/xray-panel
ExecStart=/opt/xray-panel/xray-panel-linux
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload
systemctl enable --now xray-panel
```

### 从源码编译

```bash
git clone https://github.com/coding-leaf/X-CorelittlePanel.git
cd X-CorelittlePanel
go build -o xray-panel-linux ./cmd/panel
```

### Nginx 反向代理参考（推荐）

```nginx
location /panel/ {
    proxy_pass http://127.0.0.1:8880/;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

---

## xray,nginx参考示例

可在本项目中参考配置文件示例，参考的 Xray 配置：
[防止偷流量的 reality](https://github.com/XTLS/Xray-examples/...)

---

## 安全措施推荐

**使用强密码**：`password` 字段应为随机生成的高强度字符串（建议 24 位以上），避免使用任何有规律的密码。

**不要暴露在公网**：建议将 `listen_addr` 设为 `127.0.0.1`，通过 Nginx 或 Cloudflare Tunnel 做前端代理，不要直接将面板端口开放在公网防火墙中。

**启用 HTTPS 传输**：如果前置代理已处理 TLS（Nginx + 合法证书，或 CF Tunnel），后端 HTTP 即可。如果直接暴露，必须配置 `tls_cert_file` + `tls_key_file` 或填写 `domain` 启用自动证书。

**防火墙规则**：只开放必要端口（如 443、8443 等），gRPC API 端口（10085）仅允许本地访问，不对外暴露。

**登录防爆破**：面板内置 IP 级登录限速，连续 5 次失败后封锁 15 分钟。建议在 Nginx 层额外配置 `limit_req` 进一步限制。

**定期轮换密码**：登出后会刷新 server-side token，所有旧 Cookie 立即失效。建议定期更换 `password` 并重启服务。

**不要在公开仓库中提交 config.json**：`.gitignore` 已默认排除，请勿手动 `git add config.json`。

**Telegram 通知**：Bot Token 属于高价值凭证，不要写入任何版本控制文件，只存在于服务器本地的 `config.json`。

## 界面参考
