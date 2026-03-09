#!/bin/bash
#
# SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
#

set -e

aws configure set aws_access_key_id 123
aws configure set aws_secret_access_key 123
aws configure set default.region us-west2

# aws dynamodb delete-table --table-name SampleService  --endpoint-url http://$LOCALSTACK_HOST:4566 --region us-west-2

aws dynamodb create-table --cli-input-json file://scripts/test/create-table-services.json --endpoint-url http://$LOCALSTACK_HOST:4566 --region us-west-2

