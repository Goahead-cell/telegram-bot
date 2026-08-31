# Telegram Webhook Bot

基于 [`go-telegram/bot`](https://github.com/go-telegram/bot) 的 Telegram webhook Bot。Caddy 负责公网 HTTPS 和反向代理，Go 程序只监听本机回环地址。仓库地址：[`Goahead-cell/telegram-bot`](https://github.com/Goahead-cell/telegram-bot)。

## 架构

```text
Telegram
    │ HTTPS POST + secret header
    ▼
Caddy
    │ http://127.0.0.1:18082
    ▼
internal/telegram  ──►  internal/handlers
  webhook/HTTP            业务命令
```

```text
telegram-bot/
├── .github/workflows/
│   ├── ci.yml                     # main/PR 自动测试
│   └── release.yml                # v* 标签自动构建并发布
├── build-linux.ps1                # Windows 本地测试和交叉编译
├── cmd/bot/main.go                # 程序入口
├── internal/config/               # 环境变量读取和校验
├── internal/telegram/             # go-telegram/bot webhook 与 HTTP 服务
├── internal/handlers/             # /start 和 echo 业务
└── deploy/
    ├── Caddyfile.example
    ├── telegram-bot.env.example
    ├── telegram-bot.service
    ├── install.sh                 # VPS 首次安装
    └── update.sh                  # VPS 更新和失败回滚
```

## 本地开发和测试

需要 Go 1.22 或更高版本。

```powershell
go test ./...
go vet ./...
go run ./cmd/bot
```

程序通过环境变量读取配置：

| 变量 | 说明 |
| --- | --- |
| `TELEGRAM_BOT_TOKEN` | 从 BotFather 获取，只放在运行环境中 |
| `TELEGRAM_WEBHOOK_SECRET` | 32–256 位，仅使用字母、数字、`_`、`-` |
| `TELEGRAM_WEBHOOK_URL` | 完整 HTTPS 地址，例如 `https://bot.example.com/telegram/webhook` |
| `LISTEN_ADDR` | 仅允许回环 IP，默认 `127.0.0.1:18082` |

业务入口是 `internal/handlers/router.go`。添加命令时在该文件分派，并把具体实现放入 `internal/handlers` 下的新文件；通常不需要改 webhook、Caddy 或 systemd 层。

## Windows 本地交叉编译

构建 AMD64：

```powershell
.\build-linux.ps1 -Arch amd64
```

构建 ARM64：

```powershell
.\build-linux.ps1 -Arch arm64
```

脚本先执行 `go test ./...` 和 `go vet ./...`，然后在 `dist/` 生成发布目录、`.tar.gz` 包和包校验文件：

```text
dist/
├── telegram-bot-linux-amd64.tar.gz
├── telegram-bot-linux-amd64.tar.gz.sha256
└── telegram-bot-linux-amd64/
    ├── telegram-webhook-bot
    ├── telegram-webhook-bot.sha256
    ├── install.sh
    ├── update.sh
    ├── telegram-bot.service
    ├── telegram-bot.env.example
    └── Caddyfile.example
```

`dist/` 已被 Git 忽略，发布包不包含 Token 或 webhook secret。

## GitHub Actions 自动发布

向 `main` 推送或创建 Pull Request 时，`ci.yml` 会自动执行测试和 `go vet`。

推送以 `v` 开头的版本标签后，`.github/workflows/release.yml` 会：

1. 运行所有测试和 `go vet`；
2. 以 `CGO_ENABLED=0` 构建 `linux-amd64` 和 `linux-arm64`；
3. 为二进制和压缩包生成 SHA-256 校验文件；
4. 创建对应的 GitHub Release，并上传两个架构的包。

例如发布 `v0.1.0`：

```powershell
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

工作流使用仓库自带的 `GITHUB_TOKEN`，无需保存 Telegram Token 或额外的 GitHub 密钥。

## VPS 首次安装

VPS 需要 Linux、systemd、Caddy、`curl`、`tar` 和 `sha256sum`，不需要安装 Go。先根据 `uname -m` 下载正确架构；`x86_64` 对应 `amd64`，`aarch64`/`arm64` 对应 `arm64`。

以下命令自动选择架构并下载最新稳定 Release：

```sh
case "$(uname -m)" in
  x86_64) release_arch=amd64 ;;
  aarch64|arm64) release_arch=arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

asset="telegram-bot-linux-${release_arch}.tar.gz"
release_url="https://github.com/Goahead-cell/telegram-bot/releases/latest/download"

curl --fail --location --remote-name "${release_url}/${asset}"
curl --fail --location --remote-name "${release_url}/${asset}.sha256"
sha256sum --check "${asset}.sha256"
tar -xzf "${asset}"
cd "telegram-bot-linux-${release_arch}"
sudo sh ./install.sh ./telegram-webhook-bot
```

安装脚本会再次校验二进制，创建专用的 `telegram-bot` 系统用户，并安装：

```text
/usr/local/bin/telegram-webhook-bot
/etc/telegram-bot/env
/etc/systemd/system/telegram-bot.service
```

它不会覆盖已有程序，也不会自动修改 Caddy 或启动服务。生成 secret 并编辑仅保存在 VPS 上的环境文件：

```sh
openssl rand -hex 32
sudoedit /etc/telegram-bot/env
```

填写真实值：

```text
TELEGRAM_BOT_TOKEN=从_BotFather_获取的_token
TELEGRAM_WEBHOOK_SECRET=上一步生成的64位十六进制字符串
TELEGRAM_WEBHOOK_URL=https://bot.example.com/telegram/webhook
LISTEN_ADDR=127.0.0.1:18082
```

将发布包里的 `Caddyfile.example` 合并到现有 `/etc/caddy/Caddyfile`。环境变量中的 webhook 路径必须与 Caddy 的 `path` 一致；若已有同域名站点块，应把 handler 合并进去并放在静态站点、Xray 等兜底 handler 之前。

```sh
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl reload caddy
sudo systemctl enable --now telegram-bot
curl --fail http://127.0.0.1:18082/healthz
sudo systemctl status telegram-bot --no-pager
```

排查日志：

```sh
sudo journalctl -u telegram-bot -n 100 --no-pager
```

## VPS 后续更新与失败回滚

用与首次安装相同的下载命令取得最新 Release 并校验压缩包，然后执行新包内的更新脚本：

```sh
tar -xzf "telegram-bot-linux-${release_arch}.tar.gz"
cd "telegram-bot-linux-${release_arch}"
sudo sh ./update.sh ./telegram-webhook-bot
```

`update.sh` 会校验二进制及 CPU 架构，把当前程序备份为 `/usr/local/bin/telegram-webhook-bot.previous`，再原子替换程序。如果服务原本正在运行，它会重启并观察 20 秒；新版本未能保持运行时会自动恢复旧版本，并把失败版本保存为 `/usr/local/bin/telegram-webhook-bot.failed`。如果服务更新前已停止，脚本只替换文件，不会擅自启动。

更新不会改动 `/etc/telegram-bot/env`、systemd 单元或 Caddy 配置。

## 敏感信息规则

- 不要把真实 Token、secret、私钥或 VPS 环境文件写入源码、提交记录、构建参数或 GitHub Actions。
- 本地 `.env`、`*.env`、`secrets/`、`*.key`、`*.pem` 和 `dist/` 已在 `.gitignore` 中排除。
- 仓库只保留 `deploy/telegram-bot.env.example` 占位模板；真实值保存在 VPS 的 `/etc/telegram-bot/env`，安装权限为 `0640 root:telegram-bot`。
- 如果密钥曾误提交，单纯删除文件不够：应立即在 BotFather 撤销 Token、生成新 secret，并清理 Git 历史。
