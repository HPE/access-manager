#!/bin/bash
#
# SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
#

set -e

project_name="$1"
old_project_name="$2"

if [ -z "$project_name" ]; then
    echo "Please enter a project name"
    exit
fi

if [ -z "$old_project_name" ]; then
    old_project_name="golang-template"
fi

echo "setting project_name to $project_name, where old project_name is $old_project_name"

grep -rl  $old_project_name . --exclude-dir={scripts,.git} | xargs -t sed -i "s/$old_project_name/$project_name/g"     