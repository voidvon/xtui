# XTUI

轻量、单文件的终端服务管理工具。通过命令行控制服务，通过交互面板管理用户、调整配置、查看运行状态与用量。

## 快速开始

从 [Releases](https://github.com/voidvon/xtui/releases/latest) 下载适合系统架构的文件：

| 文件 | 平台 |
| --- | --- |
| `xtui-linux-amd64` | Linux x86_64 |
| `xtui-linux-arm64` | Linux ARM64 |
| `SHA256SUMS` | 文件校验值 |

以 Linux x86_64 为例：

```bash
mkdir -p ~/xtui && cd ~/xtui
curl -fLO https://github.com/voidvon/xtui/releases/latest/download/xtui-linux-amd64
curl -fLO https://github.com/voidvon/xtui/releases/latest/download/SHA256SUMS
sha256sum --ignore-missing -c SHA256SUMS
mv xtui-linux-amd64 xtui
chmod +x xtui
./xtui ui
```

ARM64 设备将上述命令中的 `xtui-linux-amd64` 替换为 `xtui-linux-arm64`。

首次运行会在当前目录生成 `config.json`。打开管理面板时，如服务尚未启动，会自动启动后台进程；退出面板后，服务继续运行。

## 命令用法

```bash
./xtui ui          # 打开管理面板，按需启动后台服务
./xtui status      # 查看服务状态
./xtui restart     # 重启服务内核
./xtui stop        # 停止服务
./xtui             # 在当前终端持续运行服务，Ctrl+C 停止
./xtui -v          # 查看版本
./xtui -h          # 查看帮助
```

`ui` 也可写作 `tui`、`menu` 或 `config`。

### 指定配置文件

```bash
./xtui ui -c /opt/xtui/config.json
./xtui -c /opt/xtui/config.json
```

配置支持 JSONC 注释。建议使用固定目录或绝对路径，方便重启和维护。

### 指定管理接口

`-s` 用于指定本地 Socket 路径，默认是系统临时目录下的 `xtui.sock`。管理同一个服务时，各命令使用相同的 Socket 路径：

```bash
./xtui ui -c /opt/xtui/config.json -s /opt/xtui/xtui.sock
./xtui status -s /opt/xtui/xtui.sock
./xtui restart -s /opt/xtui/xtui.sock
./xtui stop -s /opt/xtui/xtui.sock
```

## 管理面板

支持鼠标点击、表单编辑和键盘快捷键。

| 按键 | 操作 |
| --- | --- |
| `F1`–`F4` | 切换面板 |
| `Tab` | 切换表单输入框 |
| `Esc` | 返回用户列表 |
| `a` / `e` / `d` | 在用户列表中添加 / 修改 / 删除用户 |
| `t` | 在用户列表中启用或停用选中用户 |
| `r` | 在用户列表中清零选中用户用量 |
| `s` | 在用户列表中查看导入信息与二维码 |
| `R` | 在运行状态面板中重启服务内核 |
| `q` | 在用户列表或运行状态面板中退出，服务继续运行 |

面板提供用户配额、到期时间、服务参数、实时速率和累计用量管理。用量定时保存到配置文件，重启后继续累计。

## 使用 systemd

将可执行文件放到 `/opt/xtui/xtui`，创建 `/etc/systemd/system/xtui.service`：

```ini
[Unit]
Description=XTUI Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/xtui
ExecStart=/opt/xtui/xtui -c /opt/xtui/config.json -s /opt/xtui/xtui.sock
Restart=always
RestartSec=3s
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

启用并管理服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now xtui
sudo systemctl status xtui
sudo journalctl -u xtui -f
```

打开管理面板：

```bash
sudo /opt/xtui/xtui ui -c /opt/xtui/config.json -s /opt/xtui/xtui.sock
```

交由 systemd 管理时，使用 `sudo systemctl stop xtui` 停止服务，使用 `sudo systemctl restart xtui` 重启进程。

## 从源码构建

Go 版本要求见 [go.mod](go.mod)。

```bash
git clone https://github.com/voidvon/xtui.git
cd xtui
make build                # 当前平台：./xtui
make build-linux          # Linux x86_64：./xtui-linux-amd64
make build-linux-arm64    # Linux ARM64：./xtui-linux-arm64
```
