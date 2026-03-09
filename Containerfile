# FROM --platform=linux/amd64 golang:1.25
FROM golang:1.25

WORKDIR /...

ENV GO111MODULE=on

ENV DEBIAN_FRONTEND=noninteractive
ENV PROTOC_VERSION=29.0
ENV PROTOC_GEN_GO_VERSION=v1.36.5
ENV PROTOC_GEN_GO_GRPC_VERSION=v1.6.1
# ENV ARCH=aarch_64
ENV ARCH=x86_64

RUN apt -y update && apt install -y unzip && apt install -y --no-install-recommends \
        gettext-base \
    && rm -rf /var/lib/apt/lists/*

ADD https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-${ARCH}.zip /usr
RUN unzip /usr/protoc-${PROTOC_VERSION}-linux-${ARCH}.zip -d /usr

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION}
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION}
RUN go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
RUN go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

RUN go env -w GOFLAGS="-buildvcs=false"

RUN CGO_ENABLED=0

ARG DEVELOPER
