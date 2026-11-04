# XTUI (Xray VLESS Reality TUI)

极简单文件（Single Binary）、纯终端交互的 **VLESS + Reality** 代理服务与管理工具。

* **分离式轻量架构**：
  * **日常运行**：直接执行 `./xtui` 即作为代理后台服务常驻运行（极适合 systemd、tmux、后台运行或 Docker）。
  * **配置管理**：随时执行 **`./xtui ui`**（或别名 `xtui tui`、`xtui menu`）打开全功能 TUI 界面进行可视化配置，**退出 TUI 后代理服务依然在后台不间断运行**！
* **单文件零依赖（Single Binary）**：内置 Xray-core，不需要额外安装或运行外部 Xray 守护进程，开箱即用。
* **全新多标签与全鼠标交互**：
  * 支持终端**鼠标直接点击**表单、切换标签与操作按钮。
  * 顶部清晰划分 **[1] 👥 用户管理**、**[2] 🛡️ Reality 配置**、**[3] ⚙️ VLESS 服务**、**[4] ⚡ 运行状态** 四大面板。
* **完整的参数配置能力**：
  * **用户管理**：添加用户、修改用户（UUID/流控/配额/到期时间）、删除用户、启用/停用、流量清零。
  * **Reality 参数深度调优**：自定义 Dest 目标网站、伪装域名 SNI 列表、短 ID (ShortIds)、爬虫伪装路径 (SpiderX)、私钥与公钥查看/手动填写/一键重新生成（推荐根据 VPS 邻近网络自定义合规目标）。
  * **VLESS 服务配置**：监听端口（Port）、监听地址（Listen IP）、节点公网 IP/域名探测、uTLS 客户端指纹（Fingerprint）。
* **实时流量监控 & 限额统计**：
  * 进程内原子计数器，精确到 Byte 的实时上传/下载速率展示。
  * 累计流量定时持久化至 `config.json`（支持带注释的 JSONC 格式），服务重启后流量不丢失。
  * 流量配额（Quota）超额与到期自动切断。
* **一键生成移动端扫码导入**：
  * 终端直接输出 **ASCII 二维码**，手机客户端（v2rayN、Shadowrocket、Sing-box、Streisand 等）扫码秒导。

---

## 命令行使用方法

```bash
# 1. 启动代理后台服务 (默认模式，常驻运行，适合 systemd / 后台运行)
./xtui

# 2. 打开交互式 TUI 管理面板 (别名: xtui tui, xtui menu, xtui config)
./xtui ui

# 3. 快速查看后台服务运行状态 (不进入 TUI)
./xtui status

# 4. 平滑热重启代理服务
./xtui restart

# 5. 停止后台服务
./xtui stop
```

> **自动守护特性**：如果后台服务尚未启动，直接执行 `./xtui ui` 时会自动在后台唤醒服务，退出 TUI 界面（按 `q` 或点击退出）时代理服务依然在后台持续运作，互不干扰！

---

## TUI 界面功能与操作

通过 `./xtui ui` 进入后，界面顶部提供 4 个功能面板（**支持鼠标直接点击**，也支持 `F1-F4` 切换面板，按 `Esc` 快速返回用户列表）：

### 1. `[1] 👥 用户管理 (User Management)`
* **用户表格**：显示用户名、运行状态、上行、下行、已用总量、配额上限、实时网速 (↑/↓)、到期时间。
* **实体操作按钮（均可鼠标点击）**：
  * `[ ➕ 添加用户 (a) ]`：添加新用户（用户名、自定义/随机 UUID、流控模式、流量配额、有效期）
  * `[ ✏️ 修改选中 (e) ]`：修改当前选中用户（修改用户名、UUID、流控、配额上限、重置有效期、启停状态）
  * `[ 📱 二维码/链接 (s) ]`：展示选中用户的 **VLESS 导入链接** 与 **终端二维码**
  * `[ ⚡ 启用/停用 (t) ]`：快速切换选中用户的启用/停用状态
  * `[ 🗑️ 删除用户 (d) ]`：删除选中用户
  * `[ 🔄 流量清零 (r) ]`：重置选中用户的已用流量为 0

### 2. `[2] 🛡️ Reality 配置 (Reality Settings)`
* **目标网站 (Dest)**：例如 VPS 邻近中型合规站点 `target-domain.com:443` 或自定义目标。
* **伪装域名 (SNI)**：支持多个域名，英文逗号分隔。
* **短 ID (ShortIds)**：支持多个 Hex 短 ID，英文逗号分隔。
* **爬虫伪装路径 (SpiderX)**：默认 `/`。
* **私钥 & 公钥 (X25519)**：展示当前密钥对，支持手动修改或点击 `[随机重新生成密钥对]`。
* 点击 `[保存并重启生效]` 即可热重载应用。

### 3. `[3] ⚙️ VLESS 服务 (VLESS Settings)`
* **监听端口 (Port)**：修改服务端口（默认 `4433`，可修改为 `443` 等）。
* **监听地址 (Listen IP)**：默认 `0.0.0.0`。
* **节点公网地址 (Public Host)**：点击 `[自动检测公网 IP]` 一键抓取 VPS 真实外网 IP。
* **客户端指纹 (Fingerprint)**：`chrome` / `safari` / `firefox` / `edge` / `ios`。

### 4. `[4] ⚡ 运行状态 (Core Status)`
* 核心运行时间、监听状态、用户总数、全站累计总流量、全站实时吞吐速率。
* 底部提供 `[ 🔄 重启代理内核 (R) ]`、`[ 🛑 停止/启动服务 ]`、`[ 🧹 清零全站流量 ]`、`[ ⬅️ 返回用户列表 ]` 按钮。

---

## Linux VPS 后台运行建议 (Systemd 服务)

编译出来的单文件极适合配置为标准 Systemd 系统服务：

```ini
# /etc/systemd/system/xtui.service
[Unit]
Description=XTUI VLESS Reality Proxy Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/xtui
ExecStart=/opt/xtui/xtui -c /opt/xtui/config.json
Restart=always
RestartSec=3s
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

配置后：
```bash
# 启动并开机自启
systemctl daemon-reload
systemctl enable --now xtui

# 需要管理时随时敲命令打开面板：
xtui ui
```

---

## 编译命令

```bash
cd /Users/yytest/Documents/projects/xtui

# 本机编译
make build
# 产物: ./xtui

# Linux VPS 静态单文件编译 (Ubuntu / Debian / CentOS / Alpine / Arch)
make build-linux
# 产物: ./xtui-linux-amd64 (约 30MB 静态二进制，无 glibc 依赖)

# Linux ARM64 编译 (如 Oracle Cloud ARM)
make build-linux-arm64
```
