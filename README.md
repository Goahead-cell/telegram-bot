# Telegram Webhook Bot

基于 [`go-telegram/bot`](https://github.com/go-telegram/bot) 的 webhook Bot。Caddy 负责公网 HTTPS，Go 服务只监听 `127.0.0.1:18082`。

仓库：[`Goahead-cell/telegram-bot`](https://github.com/Goahead-cell/telegram-bot)

## 架构

```text
Telegram ──HTTPS──► Caddy ──HTTP──► go-telegram/bot ──► handlers
                  :443             127.0.0.1:18082
```

## 发布版本

推送 `v*` 标签后，GitHub Actions 会自动测试并构建 `linux-amd64`、`linux-arm64`，然后创建 GitHub Release：

```powershell
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

Release 包含两种架构的压缩包、SHA-256 校验文件以及独立的 `install.sh`、`update.sh`。VPS 不需要安装 Go。

## VPS 一键安装

安装前只需满足：

- VPS 使用 systemd，并已安装 Caddy、`curl`、`openssl`、`tar` 和 `sha256sum`；
- 准备一个未被其他 Caddy 站点使用的 Bot 专用域名，例如 `bot.example.com`；
- 域名 A/AAAA 记录已指向 VPS，公网 80/443 端口可访问；
- GitHub Actions 已成功发布至少一个版本。

在 VPS 执行一条命令：

```sh
curl -fsSL https://github.com/Goahead-cell/telegram-bot/releases/latest/download/install.sh | sudo sh
```

安装器只会询问两项内容：

1. Bot 专用域名；
2. BotFather Token（通过 systemd 的密码输入界面读取，不显示，也不进入 Shell 历史）。

其余步骤全部自动完成：

- 检测 VPS 是 amd64 还是 arm64；
- 下载最新 GitHub Release；
- 校验压缩包和二进制 SHA-256；
- 创建低权限 `telegram-bot` 系统用户；
- 自动生成 64 位 webhook secret；
- 写入受保护的 VPS 环境文件；
- 生成 Bot 专用 Caddy 配置并导入主 Caddyfile；
- 验证、加载 Caddy；
- 安装并启动 systemd 服务；
- 检查 `/healthz`；
- 安装一键更新命令。

如果希望先检查脚本再执行：

```sh
curl -fsSL https://github.com/Goahead-cell/telegram-bot/releases/latest/download/install.sh \
  -o /tmp/telegram-bot-install.sh
less /tmp/telegram-bot-install.sh
sudo sh /tmp/telegram-bot-install.sh
```

无人值守安装可通过安全的 Token 文件传入，不要把 Token 直接写在命令参数中：

```sh
sudo sh /tmp/telegram-bot-install.sh \
  --domain bot.example.com \
  --token-file /root/telegram-bot-token
```

## 一键更新

安装完成后只需：

```sh
sudo telegram-bot-update
```

安装指定版本：

```sh
sudo telegram-bot-update v0.2.0
```

如果 VPS 是旧版安装、还没有 `telegram-bot-update` 命令，可以直接执行最新更新入口：

```sh
curl -fsSL https://github.com/Goahead-cell/telegram-bot/releases/latest/download/update.sh | sudo sh
```

更新器会自动选择架构、下载并校验 Release、备份当前程序、原子替换二进制并观察服务 20 秒。如果新版本退出或健康检查失败，它会自动恢复旧版本。

```text
/usr/local/bin/telegram-webhook-bot.previous  # 回滚版本
/usr/local/bin/telegram-webhook-bot.failed    # 启动失败的新版本
```

更新不会修改 Token、webhook secret 或 Caddy 配置。如果服务在更新前已经停止，更新器只替换文件，不会擅自启动。

## VPS 文件位置

```text
/usr/local/bin/telegram-webhook-bot       # 当前程序
/usr/local/sbin/telegram-bot-update       # 一键更新命令
/etc/telegram-bot/env                     # Token、secret、URL，权限 0640
/etc/systemd/system/telegram-bot.service  # systemd 服务
/etc/caddy/telegram-bot.caddy             # 安装器管理的 Caddy 站点
```

安装器只会在 `/etc/caddy/Caddyfile` 末尾加入：

```caddyfile
import /etc/caddy/telegram-bot.caddy
```

Caddy 配置验证或加载失败时，安装器会恢复修改前的配置。

## 常用命令

```sh
sudo systemctl status telegram-bot --no-pager
sudo journalctl -u telegram-bot -n 100 --no-pager
curl --fail http://127.0.0.1:18082/healthz
sudo systemctl restart telegram-bot
```

## Windows 本地交叉编译

AMD64：

```powershell
.\build-linux.ps1 -Arch amd64
```

ARM64：

```powershell
.\build-linux.ps1 -Arch arm64
```

脚本会先执行 `go test ./...` 和 `go vet ./...`，然后在忽略的 `dist/` 中生成与 GitHub Release 相同结构的包。

## 本地开发

需要 Go 1.22 或更高版本：

```powershell
go test ./...
go vet ./...
go run ./cmd/bot
```

业务代码位于 `internal/handlers`；webhook 和 Telegram 框架适配位于 `internal/telegram`。

## 敏感信息

- Token 和 webhook secret 只保存在 VPS 的 `/etc/telegram-bot/env`；
- Token 不应出现在命令参数、源码、构建参数或 GitHub Actions 中；
- `.env`、`*.env`、`secrets/`、`*.key`、`*.pem` 和 `dist/` 已被 Git 忽略；
- 仓库中的 `deploy/telegram-bot.env.example` 只有占位值；
- 如果 Token 曾误提交，应立即通过 BotFather 撤销并重新生成，而不是只删除文件。
