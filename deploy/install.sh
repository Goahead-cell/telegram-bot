#!/bin/sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
	echo "Run this installer as root." >&2
	exit 1
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
binary_path=${1:-"$script_dir/telegram-webhook-bot"}

if [ "$binary_path" = "-h" ] || [ "$binary_path" = "--help" ]; then
	echo "Usage: sudo sh install.sh [path-to-prebuilt-linux-binary]"
	exit 0
fi

for command_name in install systemctl useradd groupadd getent sha256sum uname grep mv; do
	if ! command -v "$command_name" >/dev/null 2>&1; then
		echo "Missing required command: $command_name" >&2
		exit 1
	fi
done

if [ ! -f "$binary_path" ] || [ ! -s "$binary_path" ]; then
	echo "Prebuilt binary not found or empty: $binary_path" >&2
	exit 1
fi

for required_file in telegram-bot.service telegram-bot.env.example Caddyfile.example; do
	if [ ! -f "$script_dir/$required_file" ]; then
		echo "Missing bundle file: $script_dir/$required_file" >&2
		exit 1
	fi
done

binary_dir=$(CDPATH= cd -- "$(dirname -- "$binary_path")" && pwd)
binary_name=$(basename -- "$binary_path")
checksum_path="$binary_dir/$binary_name.sha256"

if [ ! -f "$checksum_path" ]; then
	echo "Missing checksum file: $checksum_path" >&2
	exit 1
fi

(
	cd "$binary_dir"
	sha256sum -c "$binary_name.sha256"
)

if command -v file >/dev/null 2>&1; then
	binary_info=$(LC_ALL=C file -b "$binary_path")
	case "$(uname -m)" in
		x86_64)
			echo "$binary_info" | grep -F "ELF 64-bit" >/dev/null
			echo "$binary_info" | grep -F "x86-64" >/dev/null
			;;
		aarch64|arm64)
			echo "$binary_info" | grep -F "ELF 64-bit" >/dev/null
			echo "$binary_info" | grep -F "ARM aarch64" >/dev/null
			;;
		*)
			echo "Unsupported VPS architecture: $(uname -m)" >&2
			exit 1
			;;
	esac
fi

if ! getent group telegram-bot >/dev/null 2>&1; then
	groupadd --system telegram-bot
fi

if ! getent passwd telegram-bot >/dev/null 2>&1; then
	nologin_shell=/usr/sbin/nologin
	if [ ! -x "$nologin_shell" ]; then
		nologin_shell=/sbin/nologin
	fi
	if [ ! -x "$nologin_shell" ]; then
		nologin_shell=/bin/false
	fi
	useradd --system --gid telegram-bot --home-dir /nonexistent --shell "$nologin_shell" telegram-bot
fi

target_binary=/usr/local/bin/telegram-webhook-bot
candidate_binary=/usr/local/bin/telegram-webhook-bot.new

if [ -e "$target_binary" ]; then
	echo "$target_binary already exists; use update.sh instead of install.sh." >&2
	exit 1
fi

trap 'rm -f "$candidate_binary"' EXIT HUP INT TERM

install -o root -g root -m 0755 "$binary_path" "$candidate_binary"
mv -f "$candidate_binary" "$target_binary"
trap - EXIT HUP INT TERM

install -d -o root -g telegram-bot -m 0750 /etc/telegram-bot

if [ ! -e /etc/telegram-bot/env ]; then
	install -o root -g telegram-bot -m 0640 "$script_dir/telegram-bot.env.example" /etc/telegram-bot/env
	echo "Created /etc/telegram-bot/env; fill in its three TELEGRAM_* values before starting the service."
else
	echo "Kept existing /etc/telegram-bot/env."
fi

install -o root -g root -m 0644 "$script_dir/telegram-bot.service" /etc/systemd/system/telegram-bot.service
systemctl daemon-reload

echo "Installed prebuilt binary at $target_binary."
echo "No source code or Go compiler was used on this VPS."
echo "The installer intentionally did not edit Caddy, start, or restart the service."
