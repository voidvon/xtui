# XTUI (Xray VLESS Reality TUI)

极简纯终端（TUI）的 **VLESS + Reality** 代理管理工具。

* **单文件可执行二进制**：内置 Xray-core，不需要额外安装或运行 Xray 守护进程，真正实现开箱即用。
* **纯 TUI 交互**：无 Web 面板，所有操作均在终端完成，内存占用极低（~20MB-30MB）。
* **实时流量监控 & 限额统计**：
  * 精准统计每位用户的实时上传/下载速率（Byte 级）。
  * 累计上行、下行与总流量持久化保存（写入本地配置文件），重启不丢失。
  * 支持流量配额（Quota）和到期时间控制，超额自动切断连接。
* **开箱即用 Reality**：
  * 自动生成 X25519 密钥对与 ShortId。
  * 自动探测公网 IP。
  * 支持直接在终端生成并打印 **ASCII 二维码**，手机客户端（v2rayN、Shadowrocket、Sing-box 等）可直接扫码导入。

---

## 目录结构

```text
xtui/
├── main.go             # 程序入口、信号监听与生命周期
├── config/             # 配置管理 (config.json 加载、保存、默认配置生成、公网IP检测)
├── core/               # 内嵌 Xray 核心控制与流量统计 (Service)
├── pkg/
│   └── keys/           # X25519 密钥对与 UUID/ShortId 工具
├── ui/
│   ├── app.go          # 主界面布局、表格、键盘事件、实时刷新
│   ├── modals.go       # 添加用户、二维码展示、服务器配置等弹窗
│   └── format.go       # 流量单位转换 (KB/MB/GB)、速度格式化
├── Makefile            # 跨平台静态编译命令
└── README.md
```

---

## 快捷键说明

进入界面后，使用键盘即可完成所有操作：

| 按键 | 说明 |
| :--- | :--- |
| `↑` / `↓` / `j` / `k` | 上下选择用户 |
| `a` | 添加新用户（设置用户名、流量上限 GB、有效天数） |
| `s` 或 `Enter` | 查看当前选中用户的 **VLESS 链接** 及 **终端二维码** |
| `t` | 启用 / 停用当前用户 |
| `d` | 删除当前选中的用户 |
| `r` | 重置当前用户的已用流量为 0 |
| `c` | 服务器设置（修改端口、公网 IP/域名、Reality Target 域名、重新生成密钥等） |
| `R` | 重启内核（热载入所有配置） |
| `q` | 退出程序 |

---

## 编译与运行

### 1. 本地运行（测试开发）
```bash
go run main.go
```
首次运行会在当前目录下自动创建 `xtui.json`，并默认生成一个 `user_1` 用户与一组 Reality 密钥对。

### 2. 本机编译
```bash
make build
# 或者
go build -ldflags="-s -w" -trimpath -o xtui .
```

### 3. 交叉编译为 Linux VPS 单文件二进制（静态链接，零外部依赖）
在 macOS / Windows 开发机上直接编译给远程 Linux 服务器使用：

```bash
# 适用于绝大多数 64 位 VPS (Ubuntu / Debian / CentOS / Alpine)
make build-linux
# 产物：xtui-linux-amd64 (静态编译，无 libc 依赖)

# 适用于 ARM64 架构 VPS (如 Oracle Cloud ARM)
make build-linux-arm64
# 产物：xtui-linux-arm64
```

---

## VPS 部署推荐方式

将编译好的 `xtui-linux-amd64` 上传至服务器，例如 `/opt/xtui/xtui`：

```bash
chmod +x /opt/xtui/xtui
```

推荐搭配 `tmux` 或 `screen` 在后台保持运行：

```bash
# 启动一个后台会话
tmux new -s xtui

# 在会话中启动
/opt/xtui/xtui

# 按 Ctrl+B 然后按 D 脱离后台会话
# 需要管理时随时连回：
tmux attach -t xtui
```
