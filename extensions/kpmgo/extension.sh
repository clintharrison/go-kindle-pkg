#!/bin/sh

set -eu

reload_menu() {
	if /mnt/us/kpmgo reload-menu >/mnt/us/extensions/kpmgo/menu.json.tmp; then
		mv /mnt/us/extensions/kpmgo/menu.json.tmp /mnt/us/extensions/kpmgo/menu.json
	else
		echo "Failed to generate kpmgo menu" >&2
	fi
}

case "$1" in
reload-menu)
	reload_menu
	;;
install)
	package="$2"
	/mnt/us/kpmgo install "$package"
	reload_menu
	;;
uninstall)
	package="$2"
	/mnt/us/kpmgo uninstall "$package" >>/tmp/kpmgo-uninstall.log 2>&1
	reload_menu
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
