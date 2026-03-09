#!/bin/bash

#
# SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
#

echo "Checking go imports..."
goimports -e -d -w ./internal ./cmd
