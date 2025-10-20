#!/bin/sh

SCRIPT_PATH=$(dirname "$0")

sh "$SCRIPT_PATH/pfetch"
echo "Press enter to exit..."
read -r _
