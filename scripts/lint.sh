#!/bin/bash

#
# SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
#

echo "Checking go lint..."
golangci-lint run --timeout=10m
