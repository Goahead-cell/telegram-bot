# go-telegram/bot + Caddy webhook 项目

这是一个分层的 Telegram webhook Bot 骨架。框架接入和业务函数已经分开：`go-telegram/bot`、HTTP webhook、Caddy 接入都在基础设施层，日常业务开发集中在 `internal/handlers`。

## 项目结构

```text
telegram-bot/
├── build-linux.ps1                 # 本地测试、交叉编译并生成发布包
├── cmd/bot/main.go                 # 启动入口，只负责组装各层
├── internal/config/
│   ├── config.go                   # 环境变量读取与安全校验
│   └── config_test.go
├── internal/telegram/
│   ├── server.go                   # go-telegram/bot、webhook、HTTP 服务
│   └── server_test.go
├── internal/handlers/
│   ├── router.go                   # 将更新分派到业务函数
│   ├── start.go                    # /start 业务
│   ├── echo.go                     # 普通文字业务
│   └── router_test.go
├── deploy/
│   ├── Caddyfile.example
│   ├── telegram-bot.env.example
│   ├── telegram-bot.service
│   ├── install.sh                  # VPS 首次安装预编译二进制
│   └── update.sh                   # VPS 更新、检查并自动回滚
├── go.mod
└── go.sum
```

调用关系：

```text
Telegram → Caddy → internal/telegram → internal/handlers
```

## 如何编写业务

`cmd/bot/main.go` 把 `handlers.HandleUpdate` 交给 Telegram 服务；`internal/telegram/server.go` 再将这个外部回调传给 `go-telegram/bot`。一般不需要修改这两个文件。

命令分派写在 `internal/handlers/router.go`：

```go
if isStartCommand(update.Message.Text) {
	Start(ctx, b, update)
	return
}
```

具体业务写在独立文件，例如 `internal/handlers/start.go`：

```go
func Start(ctx context.Context, b *bot.Bot, update *models.Update) {
	sendText(ctx, b, update.Message.Chat.ID, "欢迎使用")
}
```

增加 `/help` 时，可以新建 `internal/handlers/help.go`：

```go
package handlers

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func Help(ctx context.Context, b *bot.Bot, update *models.Update) {
	sendText(ctx, b, update.Message.Chat.ID, "这里是帮助信息")
}
```

然后在 `router.go` 中增加 `/help` 分支即可。框架层、Caddy 和 systemd 都不需要修改。

## 1. 在本地生成 Linux 发布包

先在 VPS 执行 `uname -m` 确认架构：`x86_64` 使用 `amd64`，`aarch64` 使用 `arm64`。

Windows PowerShell 本地构建 amd64 发布包：

```powershell
cd "C:\Users\熊术\Documents\VPS管理\telegram-bot"
.\build-linux.ps1 -Arch amd64
```

ARM64 VPS 使用：

```powershell
.\build-linux.ps1 -Arch arm64
```

构建脚本会先运行测试和 `go vet`，然后输出：

```text
dist/telegram-bot-linux-amd64/
├── telegram-webhook-bot
├── telegram-webhook-bot.sha256
├── install.sh
├── update.sh
├── telegram-bot.service
├── telegram-bot.env.example
└── Caddyfile.example
```

同时生成 `dist/telegram-bot-linux-amd64.zip`。发布包不包含源码、Token 或 secret。

## 2. 上传并首次安装

上传发布目录：

```powershell
scp -r .\dist\telegram-bot-linux-amd64 root@VPS_HOST:/tmp/
```

登录 VPS 后执行：

```sh
cd /tmp/telegram-bot-linux-amd64
sudo sh ./install.sh ./telegram-webhook-bot
```

安装脚本只会校验 SHA-256 和二进制架构，再安装文件；不会在 VPS 上编译，也不要求 VPS 安装 Go。首次安装后生成：

```text
/usr/local/bin/telegram-webhook-bot
/etc/telegram-bot/env
/etc/systemd/system/telegram-bot.service
```

生成 webhook secret 并编辑环境文件：

```sh
openssl rand -hex 32
sudoedit /etc/telegram-bot/env
```

填写：

```text
TELEGRAM_BOT_TOKEN=从_BotFather_获取的_token
TELEGRAM_WEBHOOK_SECRET=上一步生成的64位十六进制字符串
TELEGRAM_WEBHOOK_URL=https://bot.example.com/telegram/webhook
LISTEN_ADDR=127.0.0.1:18082
```

参考发布包中的 `Caddyfile.example` 配置 Caddy。若域名已有站点块，把 Telegram handler 放在 Xray、静态站等兜底 handler 之前，不要创建第二个同名站点块。

```sh
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl reload caddy
sudo systemctl enable --now telegram-bot
```

验证：

```sh
curl --fail http://127.0.0.1:18082/healthz
sudo systemctl status telegram-bot --no-pager
sudo journalctl -u telegram-bot -n 50 --no-pager
```

## 3. 后续更新程序

本地修改 `internal/handlers` 后重新运行 `build-linux.ps1`，再把新发布目录上传到 VPS。进入新目录执行：

```sh
sudo sh ./update.sh ./telegram-webhook-bot
```

更新脚本会：

1. 校验新二进制的 SHA-256 和 CPU 架构；
2. 将当前版本保存为 `/usr/local/bin/telegram-webhook-bot.previous`；
3. 原子替换二进制；
4. 如果服务原本正在运行，则重启并观察 20 秒；
5. 新版本启动失败时自动恢复旧版本，并把失败版本保存为 `.failed`。

如果服务更新前处于停止状态，脚本只替换二进制，不会擅自启动。`/etc/telegram-bot/env`、systemd 单元和 Caddyfile 都不会被更新脚本修改。
