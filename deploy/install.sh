#!/bin/sh
set -eu

repository="Goahead-cell/telegram-bot"
service_name="telegram-bot.service"
target_binary="/usr/local/bin/telegram-webhook-bot"
updater_path="/usr/local/sbin/telegram-bot-update"
env_dir="/etc/telegram-bot"
env_file="$env_dir/env"
caddy_config="/etc/caddy/Caddyfile"
caddy_managed_config="/etc/caddy/telegram-bot.caddy"
webhook_path="/telegram/webhook"
listen_addr="127.0.0.1:18082"
traffic_state_file="/var/lib/hy2-aggregator/state.json"
over_limit_file="/var/lib/hy2-auth/over-limit.json"

release_version="latest"
bot_domain=""
admin_user_id=""
token_file=""
temp_dir=""
candidate_binary="$target_binary.new"
caddy_prepared=false
caddy_committed=false
caddy_main_changed=false
caddy_managed_existed=false

usage() {
	cat <<'EOF'
Usage: install.sh [options]

Options:
  --domain DOMAIN      Dedicated bot domain (otherwise prompted)
  --admin-user-id ID   Numeric Telegram user ID (otherwise prompted)
  --token-file PATH    Read the BotFather token from the first line of PATH
  --version TAG        Install a specific release tag instead of latest
  -h, --help           Show this help

Typical public-repository install:
  curl -fsSL https://github.com/Goahead-cell/telegram-bot/releases/latest/download/install.sh | sudo sh
EOF
}

die() {
	echo "Install failed: $*" >&2
	exit 1
}

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		die "missing required command: $1"
	fi
}

read_token_from_tty() {
	[ -r /dev/tty ] || die "no terminal available; pass --token-file"
	tty_state=$(stty -g </dev/tty) || die "could not read terminal settings"
	trap 'stty "$tty_state" </dev/tty; printf "\n" >/dev/tty; exit 1' HUP INT TERM

	printf 'Telegram BotFather token: ' >/dev/tty
	stty -echo </dev/tty || die "could not hide terminal input"
	if ! IFS= read -r bot_token </dev/tty; then
		stty "$tty_state" </dev/tty
		printf '\n' >/dev/tty
		die "could not read the Telegram token"
	fi
	stty "$tty_state" </dev/tty
	printf '\n' >/dev/tty
	trap - HUP INT TERM
}

read_admin_user_id_from_tty() {
	[ -r /dev/tty ] || die "no terminal available; pass --admin-user-id"
	printf 'Telegram administrator user ID: ' >/dev/tty
	if ! IFS= read -r admin_user_id </dev/tty; then
		die "could not read the Telegram administrator user ID"
	fi
}

validate_admin_user_id() {
	[ "${#1}" -le 18 ] && printf '%s\n' "$1" | grep -Eq '^[1-9][0-9]*$'
}

restore_caddy() {
	if [ "$caddy_prepared" != true ]; then
		return
	fi

	if [ "$caddy_main_changed" = true ]; then
		install -o root -g root -m 0644 "$temp_dir/Caddyfile.original" "$caddy_config"
	fi

	if [ "$caddy_managed_existed" = true ]; then
		install -o root -g root -m 0644 "$temp_dir/telegram-bot.caddy.original" "$caddy_managed_config"
	else
		rm -f "$caddy_managed_config"
	fi

	caddy_prepared=false
}

cleanup() {
	rm -f "$candidate_binary"
	if [ "$caddy_prepared" = true ] && [ "$caddy_committed" != true ]; then
		restore_caddy
		if systemctl is-active --quiet caddy.service; then
			systemctl reload caddy.service >/dev/null 2>&1 || true
		fi
	fi
	if [ -n "$temp_dir" ] && [ -d "$temp_dir" ]; then
		rm -rf "$temp_dir"
	fi
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		--domain)
			[ "$#" -ge 2 ] || die "--domain requires a value"
			bot_domain=$2
			shift 2
			;;
		--admin-user-id)
			[ "$#" -ge 2 ] || die "--admin-user-id requires a value"
			admin_user_id=$2
			shift 2
			;;
		--token-file)
			[ "$#" -ge 2 ] || die "--token-file requires a value"
			token_file=$2
			shift 2
			;;
		--version)
			[ "$#" -ge 2 ] || die "--version requires a value"
			release_version=$2
			shift 2
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			die "unknown option: $1"
			;;
	esac
done

if [ "$(id -u)" -ne 0 ]; then
	die "run the installer as root (for example: curl ... | sudo sh)"
fi

for command_name in caddy cat cp curl getent grep groupadd id install journalctl mktemp mv openssl rm sha256sum sleep stty systemctl tar uname useradd; do
	require_command "$command_name"
done

[ -f "$caddy_config" ] || die "$caddy_config does not exist; install and configure Caddy first"
systemctl cat caddy.service >/dev/null 2>&1 || die "caddy.service is not installed"

if [ -e "$target_binary" ]; then
	die "$target_binary already exists; update with: sudo telegram-bot-update"
fi

if [ "$release_version" != latest ] && ! printf '%s\n' "$release_version" | grep -Eq '^v[0-9A-Za-z._-]+$'; then
	die "invalid release tag: $release_version"
fi

if [ -z "$bot_domain" ]; then
	[ -r /dev/tty ] || die "no terminal available; pass --domain, --admin-user-id and --token-file"
	printf 'Telegram Bot dedicated domain (for example bot.example.com): ' >/dev/tty
	if ! IFS= read -r bot_domain </dev/tty; then
		die "could not read the bot domain"
	fi
fi

if [ "${#bot_domain}" -gt 253 ] || ! printf '%s\n' "$bot_domain" | grep -Eq '^([A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.)+[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?$'; then
	die "invalid domain: $bot_domain"
fi

if [ -z "$admin_user_id" ]; then
	read_admin_user_id_from_tty
fi

if ! validate_admin_user_id "$admin_user_id"; then
	die "Telegram administrator user ID must be a positive integer"
fi

if [ -n "$token_file" ]; then
	[ -r "$token_file" ] || die "cannot read token file: $token_file"
	IFS= read -r bot_token <"$token_file" || die "token file is empty: $token_file"
else
	read_token_from_tty
fi

if ! printf '%s\n' "$bot_token" | grep -Eq '^[0-9]+:[A-Za-z0-9_-]{20,}$'; then
	die "the Telegram token format is invalid"
fi

case "$(uname -m)" in
	x86_64)
		release_arch="amd64"
		;;
	aarch64|arm64)
		release_arch="arm64"
		;;
	*)
		die "unsupported CPU architecture: $(uname -m)"
		;;
esac

if [ "$release_version" = latest ]; then
	release_url="https://github.com/$repository/releases/latest/download"
else
	release_url="https://github.com/$repository/releases/download/$release_version"
fi

archive_name="telegram-bot-linux-$release_arch.tar.gz"
bundle_name="telegram-bot-linux-$release_arch"
temp_dir=$(mktemp -d)
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

echo "Downloading $repository release ($release_arch)..."
curl --fail --location --retry 3 --connect-timeout 10 \
	--output "$temp_dir/$archive_name" "$release_url/$archive_name"
curl --fail --location --retry 3 --connect-timeout 10 \
	--output "$temp_dir/$archive_name.sha256" "$release_url/$archive_name.sha256"

(
	cd "$temp_dir"
	sha256sum --check "$archive_name.sha256"
)
tar -xzf "$temp_dir/$archive_name" -C "$temp_dir"
bundle_dir="$temp_dir/$bundle_name"

for required_file in telegram-webhook-bot telegram-webhook-bot.sha256 update.sh telegram-bot.service; do
	[ -f "$bundle_dir/$required_file" ] || die "release bundle is missing $required_file"
done

(
	cd "$bundle_dir"
	sha256sum --check telegram-webhook-bot.sha256
)

cp -p "$caddy_config" "$temp_dir/Caddyfile.original"
if [ -e "$caddy_managed_config" ]; then
	cp -p "$caddy_managed_config" "$temp_dir/telegram-bot.caddy.original"
	caddy_managed_existed=true
fi
caddy_prepared=true

cat >"$temp_dir/telegram-bot.caddy" <<EOF
$bot_domain {
	@telegram_webhook {
		method POST
		path $webhook_path
	}

	handle @telegram_webhook {
		reverse_proxy $listen_addr
	}

	@telegram_webhook_wrong_method path $webhook_path
	handle @telegram_webhook_wrong_method {
		respond "Method Not Allowed" 405
	}

	handle {
		respond "Not Found" 404
	}
}
EOF
install -o root -g root -m 0644 "$temp_dir/telegram-bot.caddy" "$caddy_managed_config"

if ! grep -Eq '^[[:space:]]*import[[:space:]]+/etc/caddy/telegram-bot\.caddy[[:space:]]*$' "$caddy_config"; then
	printf '\nimport /etc/caddy/telegram-bot.caddy\n' >>"$caddy_config"
	caddy_main_changed=true
fi

if ! caddy validate --config "$caddy_config"; then
	restore_caddy
	die "Caddy rejected the generated configuration; use a dedicated, unused bot domain"
fi

if systemctl is-active --quiet caddy.service; then
	if ! systemctl reload caddy.service; then
		restore_caddy
		systemctl reload caddy.service >/dev/null 2>&1 || true
		die "could not reload Caddy"
	fi
else
	if ! systemctl enable --now caddy.service; then
		restore_caddy
		die "could not start Caddy"
	fi
fi

if ! getent group telegram-bot >/dev/null 2>&1; then
	groupadd --system telegram-bot
fi

if ! getent passwd telegram-bot >/dev/null 2>&1; then
	nologin_shell="/usr/sbin/nologin"
	[ -x "$nologin_shell" ] || nologin_shell="/sbin/nologin"
	[ -x "$nologin_shell" ] || nologin_shell="/bin/false"
	useradd --system --gid telegram-bot --home-dir /nonexistent --shell "$nologin_shell" telegram-bot
fi

webhook_secret=$(openssl rand -hex 32)
umask 077
cat >"$temp_dir/telegram-bot.env" <<EOF
TELEGRAM_BOT_TOKEN=$bot_token
TELEGRAM_ADMIN_USER_ID=$admin_user_id
TELEGRAM_WEBHOOK_SECRET=$webhook_secret
TELEGRAM_WEBHOOK_URL=https://$bot_domain$webhook_path
LISTEN_ADDR=$listen_addr
HY2_AGGREGATOR_STATE_FILE=$traffic_state_file
HY2_OVER_LIMIT_FILE=$over_limit_file
EOF

install -d -o root -g telegram-bot -m 0750 "$env_dir"
if [ -e "$env_file" ]; then
	cp -p "$env_file" "$env_file.preinstall"
	echo "Existing environment backed up to $env_file.preinstall"
fi
install -o root -g telegram-bot -m 0640 "$temp_dir/telegram-bot.env" "$env_file"
install -o root -g root -m 0644 "$bundle_dir/telegram-bot.service" "/etc/systemd/system/$service_name"
install -o root -g root -m 0755 "$bundle_dir/update.sh" "$updater_path"
install -o root -g root -m 0755 "$bundle_dir/telegram-webhook-bot" "$candidate_binary"
mv -f "$candidate_binary" "$target_binary"

systemctl daemon-reload
if ! systemctl enable --now "$service_name"; then
	systemctl status "$service_name" --no-pager || true
	die "the bot service could not be started; inspect: journalctl -u $service_name"
fi

healthy=false
check_count=0
while [ "$check_count" -lt 30 ]; do
	if curl --fail --silent --show-error "http://$listen_addr/healthz" >/dev/null 2>&1; then
		healthy=true
		break
	fi
	if ! systemctl is-active --quiet "$service_name"; then
		break
	fi
	check_count=$((check_count + 1))
	sleep 1
done

if [ "$healthy" != true ]; then
	systemctl status "$service_name" --no-pager || true
	journalctl -u "$service_name" -n 50 --no-pager || true
	die "the bot did not become healthy"
fi

caddy_committed=true

echo
echo "Telegram Bot installed successfully."
echo "Webhook: https://$bot_domain$webhook_path"
echo "Service: $service_name"
echo "Update later with: sudo telegram-bot-update"
