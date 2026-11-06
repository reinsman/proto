#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# 生成器固定版本；Cargo.lock 固定 Rust 生成器的传递依赖。
test "$(protoc --version)" = "libprotoc 25.3" || {
  echo 'protoc 25.3 is required' >&2; exit 1;
}
mkdir -p .tools gen/go
export GOBIN="$PWD/.tools"
export PATH="$GOBIN:$PATH"
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
protoc -I proto \
  --go_out=gen/go --go_opt=paths=source_relative \
  --go-grpc_out=gen/go --go-grpc_opt=paths=source_relative \
  proto/rein/hub/v1/*.proto
cargo run --locked --quiet -p generate-rust
