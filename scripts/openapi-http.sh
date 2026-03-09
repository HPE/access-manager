#!/bin/bash
#
# SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
#

set -e

if [[ $(command -v oapi-codegen) == "" ]]; then
    echo "Installing oapi-codegen"
    go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest
fi

readonly service="$1"
readonly output_dir="$2"
readonly package="$3"

# generate server code
for f in api/openapi/*.yaml
do
    filename=$(basename -- $f)
    filename=${filename%.*}
    echo "generating oapi server for $f"

    oapi-codegen -generate types -o "internal/$filename/api/openapi_types.gen.go" -package api "$f"
    oapi-codegen -generate gin,spec -o "internal/$filename/api/openapi_api.gen.go" -package api "$f"

done


# generate client code
for f in api/openapi/clients/*.yaml
do
    filename=$(basename -- $f)
    filename=${filename%.*}
    echo "generating oapi client for $f"
    # replace underscore with dash for package name
    oapi-codegen -generate types -o "internal/infra/$filename/openapi_types.gen.go" -package "$filename" "$f"
    oapi-codegen -generate client -o "internal/infra/$filename/openapi_client.gen.go" -package "$filename" "$f"

done


# oapi-codegen -generate types -o "$output_dir/openapi_types.gen.go" -package "$package" "api/openapi/$service.yaml"
# oapi-codegen -generate gin,spec -o "$output_dir/openapi_api.gen.go" -package "$package" "api/openapi/$service.yaml"
