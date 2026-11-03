# XTUI (Xray VLESS Reality TUI)

极简纯终端（TUI）的 **VLESS + Reality** 代理管理工具。

* **单文件零依赖（Single Binary）**：内置 Xray-core，不需要额外安装或运行外部 Xray 守护进程，开箱即用。
* **全新多标签交互（Tab Navigation & Mouse Support）**：
  * 支持终端**鼠标直接点击**表单、切换标签与操作按钮。
  * 顶部清晰划分 **[1] 👥 用户管理**、**[2] 🛡️ Reality 配置**、**[3] ⚙️ VLESS 服务**、**[4] ⚡ 运行状态** 四大面板。
* **完整的参数配置能力**：
  * **用户管理**：添加用户、修改用户（UUID/流控/配额/到期时间）、删除用户、启用/停用、流量清零。
  * **Reality 参数深度调优**：预设目标模板一键填入（Apple / Microsoft / AWS / Cloudflare 等）、自定义 Dest 目标网站、伪装域名 SNI 列表、短 ID (ShortIds)、爬虫伪装路径 (SpiderX)、私钥与公钥查看/手动填写/一键重新生成。
  * **VLESS 服务配置**：监听端口（Port）、监听地址（Listen IP）、节点公网 IP/域名探测、uTLS 客户端指纹（Fingerprint）。
* **实时流量监控 & 限额统计**：
  * 进程内原子计数器，精确到 Byte 的实时上传/下载速率展示。
  * 累计流量定时持久化至 `xtui.json`，服务重启后流量不丢失。
  * 流量配额（Quota）超额与到期自动切断。
* **一键生成移动端扫码导入**：
  * 终端直接输出 **ASCII 二维码**，手机客户端（v2rayN、Shadowrocket、Sing-box、Streisand 等）扫码秒导。

---

## 界面与面板概览

启动后顶部提供 4 个功能面板（支持按键 `1-4` 或 `F1-F4` 切换，也可直接鼠标点击）：

### 1. `[1] 👥 用户管理 (User Management)`
* **用户表格**：显示用户名、运行状态、上行、下行、已用总量、配额上限、实时网速 (↑/↓)、到期时间。
* **快捷按键**：
  * `a`：添加新用户（用户名、自定义/随机 UUID、流控模式、流量配额、有效期）
  * `e`：**修改选中用户**（修改用户名、UUID、流控、配额上限、重置有效期、启停状态）
  * `d`：删除选中用户
  * `s` 或 `Enter`：展示选中用户的 **VLESS 导入链接** 与 **终端二维码**
  * `t`：快速切换选中用户的 启用 / 停用 状态
  * `r`：重置选中用户的已用流量为 0

### 2. `[2] 🛡️ Reality 配置 (Reality Settings)`
* **预设目标快捷选择**：Apple iCloud (`gateway.icloud.com:443`)、Microsoft (`www.microsoft.com:443`)、Amazon AWS (`aws.amazon.com:443`)、Cloudflare、Yahoo Japan 等。
* **目标网站 (Dest)**：例如 `gateway.icloud.com:443`
* **伪装域名 (SNI)**：支持多个域名，逗号分隔
* **短 ID (ShortIds)**：支持多个 Hex 短 ID，逗号分隔
* **爬虫伪装路径 (SpiderX)**：默认 `/`
* **私钥 & 公钥 (X25519)**：展示当前密钥对，支持手动修改或点击 `[随机重新生成密钥对]`

### 3. `[3] ⚙️ VLESS 服务 (VLESS Settings)`
* **监听端口 (Port)**：修改服务端口（默认 `4433`，可修改为 `443` 等）
* **监听地址 (Listen IP)**：默认 `0.0.0.0`
* **节点公网地址 (Public Host)**：可一键点击 `[自动检测公网 IP]` 获取 VPS 真实外网 IP
* **客户端指纹 (Fingerprint)**：`chrome` / `safari` / `firefox` / `edge` / `ios`

### 4. `[4] ⚡ 核心状态 (Core Status)`
* 核心运行时间、监听状态、用户总数、全站累计总流量、全站实时吞吐速率。
* 按 `R` 一键平滑重启 Xray 内核，按 `r` 一键清零全站所有用户流量。

---

## 快捷键速查表

| 操作 | 快捷键 | 说明 |
| :--- | :--- | :--- |
| **切换面板** | `1` / `2` / `3` / `4` (或 `F1`-`F4`) | 在四大功能面板之间无缝切换 |
| **表单项切换** | `Tab` / `Shift+Tab` | 在配置表单各输入框之间移动焦点 |
| **添加用户** | `a` | 弹出添加用户对话框 |
| **修改用户** | `e` | 弹出修改当前选中用户对话框 |
| **删除用户** | `d` | 弹出删除确认框 |
| **二维码/链接** | `s` 或 `Enter` | 弹出节点二维码及详细参数 |
| **切换启用/禁用** | `t` | 快速开关选中用户 |
| **重置流量** | `r` | 清零选中用户流量 |
| **重启代理内核** | `R` | 热重启底层 Xray 内核 |
| **退出** | `q` | 安全退出程序（自动保存流量数据） |

*(提示：程序已启用终端鼠标支持，所有输入框、下拉框、表单按钮均可直接使用鼠标点击操作)*

---

## 编译与部署

### 1. 本机运行
```bash
go run main.go
```

### 2. 编译本机二进制
```bash
make build
# 生成当前操作系统架构的单文件: ./xtui
```

### 3. 交叉编译为 Linux VPS 单文件（静态链接，零依赖）
```bash
# 适用于绝大多数 64 位 VPS (Ubuntu / Debian / CentOS / Alpine / Arch)
make build-linux
# 产物：xtui-linux-amd64 (约 30MB 静态二进制，无 glibc 依赖)

# 适用于 ARM64 架构 VPS (如 Oracle Cloud ARM)
make build-linux-arm64
# 产物：xtui-linux-arm64
```

### 4. VPS 后台长驻运行建议
将编译出来的 `xtui-linux-amd64` 传到 VPS 服务器，结合 `tmux` 使用：

```bash
# 赋予执行权限
chmod +x xtui-linux-amd64

# 在 tmux 会话中启动
tmux new -s xtui
./xtui-linux-amd64

# 离开后台：按 Ctrl+B 然后按 D
# 随时重新查看/管理：
tmux attach -t xtui
```
