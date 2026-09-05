# Telegram Webhook Bot

基于 [`go-telegram/bot`](https://github.com/go-telegram/bot) 的 Telegram webhook Bot。Caddy 负责公网 HTTPS，Bot 服务只监听本机 `127.0.0.1:18082`。

```text
Telegram ──HTTPS──► Caddy ──HTTP──► telegram-webhook-bot
                  :443             127.0.0.1:18082
```

仓库：[`Goahead-cell/telegram-bot`](https://github.com/Goahead-cell/telegram-bot)

## Bot 指令

Bot 只响应 `TELEGRAM_ADMIN_USER_ID` 配置的管理员：

```text
/start       显示欢迎信息
/help        显示指令列表
/status      查看 CPU、系统负载、内存、磁盘和运行时间
/version     查看当前 Bot 版本
/services    查看 caddy、telegram-bot 和 hysteria-server 服务状态
/id          显示当前 Telegram 用户 ID
```

`/status` 只执行固定的只读系统命令。`/services` 使用 `systemctl is-active` 查询以下单元，不会启动、停止或重启服务：

```text
caddy.service
telegram-bot.service
hysteria-server.service
```

## 一、VPS 首次安装

### 安装前准备

VPS 需要：

- Linux + systemd；
- 已安装并能正常启动的 Caddy；
- `curl`、`openssl`、`tar`、`sha256sum`；
- 一个专门给 Bot 使用、尚未被其他 Caddy 站点占用的域名，例如 `bot.example.com`；
- 管理员本人的 Telegram 数字用户 ID；
- 域名 A/AAAA 记录已经指向 VPS，公网 80、443 端口可访问；
- 本仓库至少已经通过 GitHub Actions 发布一个版本。

VPS 不需要安装 Go，也不需要下载源码。

### 一键安装

登录 VPS，执行：

```sh
curl -fsSL https://github.com/Goahead-cell/telegram-bot/releases/latest/download/install.sh | sudo sh
```

安装器会询问三项内容：

```text
Telegram Bot dedicated domain: bot.example.com
Telegram administrator user ID: 123456789
Telegram BotFather token: [隐藏输入]
```

- 域名填写纯域名，不要添加 `https://` 或路径；
- 管理员 ID 必须是 Telegram 用户的正整数 ID；Bot 只响应这个用户发送的消息；
- Token 从 BotFather 获取，安装器直接从当前 SSH 终端隐藏读取，不会显示，也不会进入 Shell 历史；
- webhook 地址固定为 `https://你的域名/telegram/webhook`。

不知道自己的数字 ID 时，可以先在 Telegram 中使用 ID 查询 Bot（例如 `@userinfobot`）查看。数字 ID 本身不是密码，但必须确认填写的是你自己的用户 ID，不是 Bot ID 或群组的 Chat ID。

安装器会自动完成：

1. 检测 VPS 是 amd64 还是 arm64；
2. 下载最新 GitHub Release；
3. 校验压缩包和二进制 SHA-256；
4. 创建低权限 `telegram-bot` 系统用户；
5. 自动生成 webhook secret；
6. 写入 Token、secret 和 webhook URL；
7. 生成并验证 Caddy 配置；
8. 安装、启动 systemd 服务；
9. 检查 Bot 健康状态；
10. 安装 `telegram-bot-update` 更新命令。

安装成功后，在 Telegram 中给机器人发送 `/start` 测试。

### 先检查脚本再安装

不想直接执行远程脚本时，可以先下载检查：

```sh
curl -fsSL https://github.com/Goahead-cell/telegram-bot/releases/latest/download/install.sh \
  -o /tmp/telegram-bot-install.sh

less /tmp/telegram-bot-install.sh
sudo sh /tmp/telegram-bot-install.sh
```

### 无人值守安装

把 Token 单独放入只有 root 能读取的文件，不要把 Token 写进命令参数：

```sh
sudo install -m 0600 /dev/null /root/telegram-bot-token
sudoedit /root/telegram-bot-token

sudo sh /tmp/telegram-bot-install.sh \
  --domain bot.example.com \
  --admin-user-id 123456789 \
  --token-file /root/telegram-bot-token
```

安装后可以删除临时 Token 文件：

```sh
sudo rm -f /root/telegram-bot-token
```

## 二、配置说明

### Bot 环境配置

安装器自动创建：

```text
/etc/telegram-bot/env
```

文件权限为 `0640 root:telegram-bot`，内容格式如下：

```text
TELEGRAM_BOT_TOKEN=从_BotFather_获取的_token
TELEGRAM_ADMIN_USER_ID=你的_Telegram_数字用户_ID
TELEGRAM_WEBHOOK_SECRET=安装器自动生成的64位字符串
TELEGRAM_WEBHOOK_URL=https://bot.example.com/telegram/webhook
LISTEN_ADDR=127.0.0.1:18082
```

不要把这个文件复制到仓库、聊天记录或日志中。

`TELEGRAM_ADMIN_USER_ID` 使用发送者的 `update.Message.From.ID` 验证身份，而不是聊天窗口的 `Chat.ID`。修改管理员后需要重启服务。

修改 Token 或其他配置：

```sh
sudoedit /etc/telegram-bot/env
sudo systemctl restart telegram-bot
```

通常不需要手动修改 `TELEGRAM_WEBHOOK_SECRET`。如果必须更换：

```sh
openssl rand -hex 32
sudoedit /etc/telegram-bot/env
sudo systemctl restart telegram-bot
```

### Caddy 配置

安装器创建：

```text
/etc/caddy/telegram-bot.caddy
```

并在 `/etc/caddy/Caddyfile` 末尾加入：

```caddyfile
import /etc/caddy/telegram-bot.caddy
```

生成的站点配置类似：

```caddyfile
bot.example.com {
	@telegram_webhook {
		method POST
		path /telegram/webhook
	}

	handle @telegram_webhook {
		reverse_proxy 127.0.0.1:18082
	}

	@telegram_webhook_wrong_method path /telegram/webhook
	handle @telegram_webhook_wrong_method {
		respond "Method Not Allowed" 405
	}

	handle {
		respond "Not Found" 404
	}
}
```

Caddy 验证或加载失败时，安装器会恢复修改前的配置。

### 更换域名

先把新域名 DNS 指向 VPS，然后同时修改两个文件：

```sh
sudoedit /etc/telegram-bot/env
sudoedit /etc/caddy/telegram-bot.caddy
```

例如将两处域名都改成 `newbot.example.com`，再执行：

```sh
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl reload caddy
sudo systemctl restart telegram-bot
```

`TELEGRAM_WEBHOOK_URL` 中的域名和路径必须与 Caddy 配置完全一致。

### 文件位置

```text
/usr/local/bin/telegram-webhook-bot       # 当前 Bot 程序
/usr/local/sbin/telegram-bot-update       # 一键更新命令
/etc/telegram-bot/env                     # Token、secret、webhook URL
/etc/systemd/system/telegram-bot.service  # systemd 服务
/etc/caddy/telegram-bot.caddy             # Bot 专用 Caddy 配置
```

## 三、一键更新与回滚

### 更新到最新版本

```sh
sudo telegram-bot-update
```

如果当前 sudo 配置找不到 `/usr/local/sbin`，使用完整路径：

```sh
sudo /usr/local/sbin/telegram-bot-update
```

### 更新或回退到指定版本

```sh
sudo telegram-bot-update v0.3.0
```

参数必须是已经存在的 GitHub Release 标签。

### 更新过程

更新器会自动：

1. 检查 `/etc/telegram-bot/env` 中的管理员 ID，缺少时从当前终端询问并写入；
2. 检测 VPS CPU 架构；
3. 下载并校验目标 Release；
4. 备份当前程序和 systemd 服务单元；
5. 同步更新 systemd 服务单元；
6. 原子替换程序并重启服务；
7. 持续检查 systemd 和 `/healthz` 20 秒；
8. 新版本失败时自动恢复旧程序和旧服务单元；
9. 更新 VPS 上的 `telegram-bot-update` 命令本身。

相关文件：

```text
/usr/local/bin/telegram-webhook-bot.previous  # 更新前版本
/usr/local/bin/telegram-webhook-bot.failed    # 启动失败的新版本
```

除缺少时补写 `TELEGRAM_ADMIN_USER_ID` 外，更新不会修改 `/etc/telegram-bot/env` 或 Caddy 配置。如果服务在更新前已经停止，更新器只替换文件，不会擅自启动。

旧安装还没有 `telegram-bot-update` 命令时，可执行：

```sh
curl -fsSL https://github.com/Goahead-cell/telegram-bot/releases/latest/download/update.sh | sudo sh
```

从不包含管理员 ID 的旧版本升级时，VPS 上现有的旧更新器还不知道这项新配置。发布包含本次修改的新版本后，第一次请直接运行上面的远程 `update.sh`；它会询问管理员 ID。此后继续使用 `sudo telegram-bot-update` 即可。

## 四、检查状态与排错

检查服务：

```sh
sudo systemctl status telegram-bot --no-pager
```

查看日志：

```sh
sudo journalctl -u telegram-bot -n 100 --no-pager
```

检查本地健康状态：

```sh
curl --fail http://127.0.0.1:18082/healthz
```

正常输出：

```text
ok
```

检查 Caddy：

```sh
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl status caddy --no-pager
sudo journalctl -u caddy -n 100 --no-pager
```

常见问题：

- 安装地址返回 `404`：仓库还没有 Release，或者最新 Release 没有上传 `install.sh`；
- Caddy 无法申请证书：检查域名 DNS 和 VPS 的 80/443 防火墙；
- Bot 服务不断重启：检查 Token、服务器能否访问 Telegram API，以及 `journalctl` 日志；
- webhook 不生效：确认环境文件和 Caddy 中的域名、`/telegram/webhook` 路径一致。

## 五、发布新版本

提交代码并推送以 `v` 开头的标签：

```powershell
git push origin main
git tag -a v0.3.0 -m "v0.3.0"
git push origin v0.3.0
```

GitHub Actions 会自动运行测试，构建 `linux-amd64`、`linux-arm64`，生成 SHA-256，并上传到 GitHub Releases。等待工作流成功后，VPS 才能安装或更新到该版本。

## 六、本地开发与构建

需要 Go 1.22 或更高版本：

```powershell
go test ./...
go vet ./...
```

本地交叉编译：

```powershell
.\build-linux.ps1 -Arch amd64 -Version v0.3.0
.\build-linux.ps1 -Arch arm64 -Version v0.3.0
```

构建产物位于被 Git 忽略的 `dist/`。Token、webhook secret、`.env`、私钥和构建产物都不应进入 Git。
