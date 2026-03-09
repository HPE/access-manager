#!/bin/bash
#
# SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
#

set -e

if [[ $(command -v wire) == "" ]]; then
    echo "Installing wire"
    go install github.com/google/wire/cmd/wire@latest
fi

wire ./...
