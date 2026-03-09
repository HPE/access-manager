export
include Makefile.variables

# replace BINARY_NAME with your service name
AM_BINARY_NAME=access-manager
CLI_BINARY_NAME=am

GOCMD=go

local: | quiet
	@$(eval DOCKRUN= )
	@mkdir -p tmp
	@touch tmp/dev_image_id
quiet:
	@:

prepare: dependencies
dependencies:
	@mkdir -p tmp
	@podman rmi -f ${DEV_IMAGE} > /dev/null 2>&1 || true
	@echo ${DEV_IMAGE}
	@podman build -t ${DEV_IMAGE} -f Containerfile .
	@podman inspect -f "{{ .ID }}" ${DEV_IMAGE} > tmp/dev_image_id

all: openapi test build

test: check
	${DOCKRUN} bash ./scripts/test.sh

check:format
	${DOCKRUN} bash ./scripts/lint.sh

format:
	${DOCKRUN} bash ./scripts/goimports.sh

.PHONY: openapi
openapi:
	@./scripts/openapi-http.sh

.PHONY: build
build: build-am build-cli

build-am:
	@echo -----------------------Build binary for $(AM_BINARY_NAME)-------------------------------
	GO111MODULE=on $(GOCMD) build -o bin/$(AM_BINARY_NAME) ./cmd/$(AM_BINARY_NAME)

build-cli:
	@echo -----------------------Build binary for $(CLI_BINARY_NAME)-------------------------------
	GO111MODULE=on $(GOCMD) build -o bin/$(CLI_BINARY_NAME) ./cmd/$(CLI_BINARY_NAME)

.PHONY: clean
clean:
	rm -rf ./bin ./tmp ./vendor

# Main entry point for this makefile. Uses podman to generate code from protos
# It will create a podman image with all the required dependencies to generate protos,
# mount the root folder to a temp repo_path under which the make gen-podman command will be executed.
proto-gen: ## generates protos for golang
	rm -rf ./protobuf/proto/**.pb.go
	${DOCKRUN} bash ./scripts/proto-gen.sh

