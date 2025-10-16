#!/bin/bash
# This file exists to be `PATH_add`-ed in the .envrc

exec go build -o gtk2 "${@}" ./cmd/gtk2/main.go
