#!/bin/sh
set -eu

repository="Goahead-cell/telegram-bot"
service_name="telegram-bot.service"
target_binary="/usr/local/bin/telegram-webhook-bot"
previous_binary="$target_binary.previous"
failed_binary="$target_binary.failed"
candidate_binary="$target_binary.new"
updater_path="/usr/local/sbin/telegram-bot-update"
updater_candidate="$updater_path.new"
health_url="http://127.0.0.1:18082/healthz"
release_version="latest"
temp_dir=""

usage() {
	cat <<'EOF'
Usage: telegram-bot-update [VERSION]

Without VERSION, download and install the latest stable GitHub Release.
Pass a tag such as v0.2.0 to install that specific release.
EOF
}

die() {
	echo "Update failed: $*" >&2
	exit 1
}

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		die "missing required command: $1"
	fi
}

cleanup() {
	rm -f "$candidate_binary" "$updater_candidate"
	if [ -n "$temp_dir" ] && [ -d "$temp_dir" ]; then
		rm -rf "$temp_dir"
	fi
}

observe_service() {
	observe_limit=$1
	observe_count=0
	health_seen=false

	while [ "$observe_count" -lt "$observe_limit" ]; do
		sleep 1
		if ! systemctl is-active --quiet "$service_name"; then
			return 1
		fi
		if curl --fail --silent --show-error "$health_url" >/dev/null 2>&1; then
			health_seen=true
		fi
		observe_count=$((observe_count + 1))
	done

	[ "$health_seen" = true ]
}

case "${1:-}" in
	-h|--help)
		usage
		exit 0
		;;
	"")
		;;
	*)
		[ "$#" -eq 1 ] || die "only one VERSION argument is accepted"
		release_version=$1
		;;
esac

if [ "$(id -u)" -ne 0 ]; then
	die "run the updater as root: sudo telegram-bot-update"
fi

for command_name in cat curl grep id install journalctl mktemp mv rm sha256sum sleep systemctl tar uname; do
	require_command "$command_name"
done

if [ "$release_version" != latest ] && ! printf '%s\n' "$release_version" | grep -Eq '^v[0-9A-Za-z._-]+$'; then
	die "invalid release tag: $release_version"
fi

[ -f "$target_binary" ] || die "the bot is not installed; run install.sh first"
[ -f /etc/telegram-bot/env ] || die "/etc/telegram-bot/env is missing"
systemctl cat "$service_name" >/dev/null 2>&1 || die "$service_name is not installed"

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

echo "Downloading $repository release ($release_arch, $release_version)..."
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

for required_file in telegram-webhook-bot telegram-webhook-bot.sha256 update.sh; do
	[ -f "$bundle_dir/$required_file" ] || die "release bundle is missing $required_file"
done

(
	cd "$bundle_dir"
	sha256sum --check telegram-webhook-bot.sha256
)

was_active=false
if systemctl is-active --quiet "$service_name"; then
	was_active=true
fi

install -o root -g root -m 0755 "$target_binary" "$previous_binary"
install -o root -g root -m 0755 "$bundle_dir/telegram-webhook-bot" "$candidate_binary"
mv -f "$candidate_binary" "$target_binary"

if [ "$was_active" = false ]; then
	install -o root -g root -m 0755 "$bundle_dir/update.sh" "$updater_candidate"
	mv -f "$updater_candidate" "$updater_path"
	echo "Bot binary updated. The service was already stopped, so it was not started."
	echo "Start it when ready: sudo systemctl start $service_name"
	exit 0
fi

update_ok=true
if ! systemctl restart "$service_name"; then
	update_ok=false
elif ! observe_service 20; then
	update_ok=false
fi

if [ "$update_ok" = true ]; then
	install -o root -g root -m 0755 "$bundle_dir/update.sh" "$updater_candidate"
	mv -f "$updater_candidate" "$updater_path"
	echo "Telegram Bot updated successfully."
	echo "Rollback binary: $previous_binary"
	exit 0
fi

echo "The new release failed its health check; rolling back." >&2
systemctl status "$service_name" --no-pager || true
journalctl -u "$service_name" -n 50 --no-pager || true
install -o root -g root -m 0755 "$target_binary" "$failed_binary"
install -o root -g root -m 0755 "$previous_binary" "$candidate_binary"
mv -f "$candidate_binary" "$target_binary"

if systemctl restart "$service_name" && observe_service 10; then
	echo "Rollback succeeded. Failed binary: $failed_binary" >&2
else
	echo "Rollback binary was restored, but the service is still unhealthy." >&2
	echo "Inspect: journalctl -u $service_name" >&2
fi

exit 1
