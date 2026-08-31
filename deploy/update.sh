#!/bin/sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
	echo "Run this updater as root." >&2
	exit 1
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
binary_path=${1:-"$script_dir/telegram-webhook-bot"}

if [ "$binary_path" = "-h" ] || [ "$binary_path" = "--help" ]; then
	echo "Usage: sudo sh update.sh [path-to-prebuilt-linux-binary]"
	exit 0
fi

for command_name in install systemctl sha256sum uname grep mv sleep; do
	if ! command -v "$command_name" >/dev/null 2>&1; then
		echo "Missing required command: $command_name" >&2
		exit 1
	fi
done

target_binary=/usr/local/bin/telegram-webhook-bot
service_name=telegram-bot.service

if [ ! -f "$target_binary" ]; then
	echo "Bot is not installed. Run install.sh first." >&2
	exit 1
fi
if [ ! -f "$binary_path" ] || [ ! -s "$binary_path" ]; then
	echo "Prebuilt binary not found or empty: $binary_path" >&2
	exit 1
fi
if ! systemctl cat "$service_name" >/dev/null 2>&1; then
	echo "$service_name is not installed. Run install.sh first." >&2
	exit 1
fi

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

candidate_binary=/usr/local/bin/telegram-webhook-bot.new
previous_binary=/usr/local/bin/telegram-webhook-bot.previous
failed_binary=/usr/local/bin/telegram-webhook-bot.failed
trap 'rm -f "$candidate_binary"' EXIT HUP INT TERM

was_active=false
if systemctl is-active --quiet "$service_name"; then
	was_active=true
fi

install -o root -g root -m 0755 "$target_binary" "$previous_binary"
install -o root -g root -m 0755 "$binary_path" "$candidate_binary"
mv -f "$candidate_binary" "$target_binary"

if [ "$was_active" = false ]; then
	trap - EXIT HUP INT TERM
	echo "Updated $target_binary. The service was inactive, so it was not started."
	echo "Start it when ready: sudo systemctl start $service_name"
	exit 0
fi

update_ok=true
if ! systemctl restart "$service_name"; then
	update_ok=false
else
	check_count=0
	while [ "$check_count" -lt 20 ]; do
		sleep 1
		if ! systemctl is-active --quiet "$service_name"; then
			update_ok=false
			break
		fi
		check_count=$((check_count + 1))
	done
fi

if [ "$update_ok" = true ]; then
	trap - EXIT HUP INT TERM
	echo "Updated and restarted $service_name successfully."
	echo "Rollback binary: $previous_binary"
	exit 0
fi

echo "The new service did not stay active; rolling back." >&2
systemctl status "$service_name" --no-pager || true
install -o root -g root -m 0755 "$target_binary" "$failed_binary"
install -o root -g root -m 0755 "$previous_binary" "$candidate_binary"
mv -f "$candidate_binary" "$target_binary"

if systemctl restart "$service_name" && systemctl is-active --quiet "$service_name"; then
	echo "Rollback succeeded. Failed binary: $failed_binary" >&2
else
	echo "Rollback was installed, but the service is still unhealthy. Check journalctl -u $service_name." >&2
fi

trap - EXIT HUP INT TERM
exit 1
