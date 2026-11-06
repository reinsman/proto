# Rein protocol

`proto/rein/hub/v1/*.proto` 是 Rust reind 与 Go reinhub 的唯一跨语言契约。
本地 Unix socket IPC 独立；原生 Codex/Pi 协议与 A2A 协议分别由 Backend 适配。

- `auth.proto`：daemon 主导的异步登录。
- `runtime.proto`：Agent、Runtime、Backend 与能力声明。
- `run.proto`：Session、Run、领取租约及控制意图 RPC。
- `daemon.proto`：双向流、事件、控制命令、持久化确认和重连对账。

状态、权限、去重、取消与恢复约束见 [docs/semantics.md](docs/semantics.md)。当前是框架阶段，不代表这些语义已经实现。

## 生成与检查

需要 protoc **25.3**、Go **1.25.1**、Rust **1.94.0**、Python 3；Buf 检查使用 **1.57.2**。
Go 插件版本固定在 `tools/generate.sh`，Rust 生成器固定在 Cargo manifests 和 Cargo.lock。

```sh
bash tools/generate.sh
python3 tools/check-generated.py
buf lint
buf format --diff --exit-code
go test ./...
cargo test --locked -p rein-proto
```

Go 产物位于 `gen/go`，Rust crate 位于 `gen/rust/rein-proto`，均纳入版本管理；禁止手改生成代码。
CI 对 PR 基线执行 breaking 检查，首次仓库建立时没有历史基线。

开发阶段：reinhub 使用 `replace github.com/reinsman/proto => ../proto`，rein 使用 Cargo path。
发布前固定协议 tag/crate 版本并移除本地路径，不跟随浮动分支。
