#!/bin/bash

GENERATED_CODE_DIRECTORY="internal/adapters/primary/grpc/gen"

mkdir -p "$GENERATED_CODE_DIRECTORY"

protoc --go_out="$GENERATED_CODE_DIRECTORY" \
       --go_opt=paths=source_relative \
       --go-grpc_out="$GENERATED_CODE_DIRECTORY" \
       --go-grpc_opt=paths=source_relative \
       proto/chat.proto
