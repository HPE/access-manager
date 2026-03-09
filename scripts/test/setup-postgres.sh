#!/bin/bash
#
# SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
#

set -e

# dropdb -h ccs-pg -U postgres sample_service
createdb -h ccs-pg -U postgres sample_service