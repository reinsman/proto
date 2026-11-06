fn main() -> Result<(), Box<dyn std::error::Error>> {
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
    let sources = ["auth", "runtime", "run", "daemon"]
        .map(|name| root.join(format!("proto/rein/hub/v1/{name}.proto")));
    tonic_prost_build::configure()
        // Connect 是双向流 RPC 名；禁用同名客户端建连便捷方法，连接由 HubClient 管理。
        .build_transport(false)
        .out_dir(root.join("gen/rust/rein-proto/src"))
        .compile_protos(&sources, &[root.join("proto")])?;
    Ok(())
}
