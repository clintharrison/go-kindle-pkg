#!/bin/sh

# Run kterm with:
# -k 1: keyboard on
# -o U: orientation Up
# -s 7: font size 7
/mnt/us/extensions/kterm/bin/kterm \
    -e "sh $PWD/pfetch/run.sh" \
    -k 1 \
    -o U \
    -s 7
