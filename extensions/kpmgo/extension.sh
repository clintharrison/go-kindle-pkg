#!/bin/sh

set -eu

case "$1" in
reload-menu)
	if /mnt/us/kpmgo reload-menu >/mnt/us/extensions/kpmgo/menu.json.tmp; then
		mv /mnt/us/extensions/kpmgo/menu.json.tmp /mnt/us/extensions/kpmgo/menu.json
	else
		echo "Failed to generate kpmgo menu" >&2
	fi
	;;
install)
	package="$2"
	/mnt/us/kpmgo install "$package"
	;;
uninstall)
	package="$2"
	/mnt/us/kpmgo uninstall "$package"
	;;
launch)
	package="$2"
	nohup sh -lc "sleep 1; /mnt/us/kpmgo launch \"$package\" >/tmp/kpmgo-launch.log 2>&1" &
	;;
*)
	echo "Unknown command: $1" >&2
	exit 1
	;;
esac
