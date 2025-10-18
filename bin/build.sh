#!/bin/bash
# This file exists to be `PATH_add`-ed in the .envrc

exec go build -o build/cli "${@}" ./cmd/cli/
