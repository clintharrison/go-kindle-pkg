#!/bin/sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# Run kterm with:
# -k 1: keyboard on
# -o U: orientation Up
# -s 7: font size 7
/mnt/us/extensions/kterm/bin/kterm \
    -e "sh -c \"$SCRIPT_DIR/pfetch/run.sh\"" \
    -k 1 \
    -o U \
    -s 7
