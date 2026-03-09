#!/bin/bash

#
# SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
#

echo "Generating go protos for access-manager"
protoc \
  --grpc-gateway_out ./internal/services/access-manager \
  --openapiv2_out ./protobuf/am-proto/swagger \
  --openapiv2_opt use_go_templates=true \
  --proto_path=./protobuf/proto \
  --proto_path=./ \
  ./internal/services/access-manager/accessmanager.proto \
  --go_out=./ \
  --go_opt=paths=source_relative \
  --go-grpc_opt=paths=source_relative \
  --go-grpc_out=./ \
  --grpc-gateway_opt paths=source_relative \
  --grpc-gateway_out=./ \
  --grpc-gateway_opt generate_unbound_methods=true

# the protoc command above seems to leave trash files around
rm -rf ./internal/services/access-manager/internal

echo "Generating credential message formats"
protoc \
  --proto_path=./protobuf/proto \
  --proto_path=./ \
  ./internal/services/access-manager/credentials.proto \
  --go_out=./ \
  --go_opt=paths=source_relative

echo "Generating persistence formats"
protoc \
  --proto_path=./protobuf/proto \
  --proto_path=./internal/services/metadata \
  ./internal/services/metadata/meta.proto \
  --go_out=./internal/services/metadata \
  --go_opt=paths=source_relative
