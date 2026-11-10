# Rein Hub v1 交互语义（框架阶段）

`.proto` 是跨语言协议唯一来源；以下是实现必须满足的约束，不代表占位服务已经实现。

## 边界与身份

- CLI 通过 Unix socket 操作 reind；只有 reind 使用 gRPC/TLS 连接 Hub。
- reinhub 保存 Agent、Runtime、Session、Run 和执行记录的权威状态。
- Runtime 绑定 workspace、所有者和 daemon。Native 在 daemon 内直接管理子进程及管道；A2A 调用独立 Agent 服务。
- StartLogin/GetLoginStatus 是异步登录。login_secret 是 daemon 生成的至少 32 字节随机值；Hub 只保存其摘要。查询须校验秘密、有效期和请求所有权，并限流。
- 登录成功返回绑定用户和 daemon 的连接凭据。只由 daemon 安全保存；不得进入 IPC 响应、Agent 环境或日志。凭据续期、撤销和浏览器授权仍待实现。
- 除登录 RPC，所有 RPC/流在接入时验证凭据；每次操作还须校验 workspace、Runtime/Run 的权限及 daemon 绑定。流期间撤销和到期同样必须生效。
- Hub 凭据、模型凭据和 A2A 凭据独立。敏感配置由 daemon 本地解析，不通过 Agent/Runtime 的公开描述返回。
- 框架的认证器默认拒绝请求，业务方法返回 UNIMPLEMENTED；不发行假凭据、不提供测试登录后门。

## 普通 RPC 与流

- AuthService：StartLogin/GetLoginStatus。
- RuntimeService：RegisterRuntime / RegisterAgent / UnregisterAgent / ListAgents；RunService：CreateRun/ClaimRuns/GetRun 及控制意图 RPC。
- DaemonService.Connect 由 daemon 主动建立。第一帧 Hello，附带本地 attempt 检查点与未完成命令确认；Hub 返回 ReconcileResponse，完成对账后才能领取和继续执行。
- WorkAvailable 只是提示；daemon 根据本地空闲容量调用 ClaimRuns。心跳维持连接可见性，不作为某个 Run 已完成的依据。
- 断线退避重连，重新 Hello/对账；定时 GetRun 可补偿通知丢失。流发送和接收不能互相阻塞。短 RPC 必须有 deadline；长流按心跳/空闲策略检测连接，不套用执行总时限。
- 本地 IPC 保持有界短请求；登录和长执行通过后台任务与查询分离。

## Run、Attempt 与状态

- CLI 使用 up/down 表达 Agent 上下线。RegisterAgent 上线或更新配置；UnregisterAgent 校验 daemon 所有权后设置 Agent.disabled，不删除描述、Runtime、会话或 Run 历史。列表保留已下线 Agent，online=false；CreateRun 和 ClaimRuns 拒绝调度。queued、执行中及等待交互的 Run 均阻止下线，检查与停用必须原子完成。
- UnregisterAgent 的 request_id 绑定 daemon、Agent 和请求内容，持久化幂等；旧请求重试不得撤销后续明确的上线。daemon 先保存下线意图，再请求 Hub；暂时失败、响应丢失或重启后复用同一 request_id，确认前不报告成功、不重新发布该 Agent。明确拒绝后恢复本地启用状态。再次 up 同名 Agent 保留 ID，裸 up 使用保存配置。

```
queued → dispatched → running → completed / failed / cancelled
                          ↔ waiting_input / waiting_approval
```

- Run 是一次平台调用；Session 表示连续对话。Native 会话、A2A context/task 标识与平台 Session/Run 分别保存。
- ClaimRuns 在事务中按容量分配 Runtime，建立新的 attempt_id 与启动租约。request_id 的重复请求返回原领取结果；相同 ID 携带不同参数返回冲突。
- daemon 先持久化领取和准备信息，再上报 running StateEvent。Hub 校验当前 attempt、租约及权限后事务提交并 ACK；daemon 取得此启动确认后才允许 Backend 开始有副作用的执行。
- 等待输入/审批保持同一 attempt。回到 running 需要明确的有效回复；普通文本不能隐式成为审批授权。
- 终态不再迁移。新的重试使用新的 attempt_id；旧 attempt 的启动、事件和控制不得覆盖新 attempt。租约过期不证明旧执行已停止。
- 连接状态与 RunState 分开保存。断线/失联不自动宣告完成、取消或允许重跑；无活动执行也不能仅由 socket 状态推断。

## 事件顺序、ACK 与恢复

- sequence 作用域为 `(run_id, attempt_id)`，从 1 连续递增，所有事件类型共享该序列。SQLite 在同一事务中分配序号并插入 outbox，重启后不能复用序号。
- Hub 按 `(run_id, attempt_id, sequence)` 唯一约束幂等接收。同键不同内容拒绝，不重复累计用量或结果。UsageEvent 表示本次 attempt 的用量增量。
- EventAck.through_sequence 是**已提交事务的最高连续序号**，不只是收到消息。它覆盖事件入库、状态更新和相关计数；空洞不得越过。daemon 持久化 ACK 后才删除对应 outbox。
- TextEvent.complete=true 是 item 的完整替换快照；false 是增量。ToolEvent 以 call_id 关联；重连快照不能重复追加。
- 终态 ResultEvent 在发送前持久化，通信失败后重发同一事件；先确认此前事件，再累计 ACK。不能因发送失败重新运行 Agent。
- 重连对账返回 Hub 当前 Run、持久化 ACK 和待执行控制。UNKNOWN/缺失/冲突的执行须停止新派发并查询确认，不静默创建新 Run。
- A2A 已取得 taskId 时使用 GetTask/订阅恢复；未取得回执的提交结果未知。只有服务明确保证幂等时才可重投相同 messageId；不得默认所有 A2A 服务都去重。
- Native 进程是否存活、会话是否可恢复和之前执行是否已完成须分别核实。有副作用但结果未知的执行不得自动重跑。

## 控制与取消

- command_id 标识一条逻辑控制命令，重投保持不变。去重记录绑定 attempt 和内容；同 ID 不同内容拒绝。
- RECEIVED ACK 表示 daemon 已将命令持久化到 inbox，不表示已执行。APPLIED 表示操作已获执行端确认，REJECTED 表示明确拒绝；Hub 必须持久化 ACK 后才清除对应投递记录，对账重传安全。
- RequestCancel 响应仅表示取消意图已存储。cancel_requested 与 RunState 分开，不能直接标记 cancelled。
- Native 取消必须确认 Agent 执行、所属工具及子进程清理；A2A 必须确认远端 Task 已取消。之后才发送 execution_stopped=true 的 cancelled ResultEvent。
- 取消与完成竞态以已确认的实际终态为准；无法确认停止时保留取消意图与未知连接状态，不伪造成功。超时同样不等于已停止。
- 审批回复同时验证 Run、attempt、interaction_id、回复结构、用户权限和是否已消费。
- 高级控制先检查 runtime capability，再由 Backend 专属接口验证。Native 的会话恢复、审批、取消、steer 等能力不受 A2A 能力集合限制。

## 版本与本轮范围

- v1 包内只做兼容增加，删除字段须 reserved，禁止复用 field number；破坏性变更使用新版本包。
- 生成器版本固定于工具脚本和 Cargo.lock；生成产物只能通过生成器修改。CI 重生成比较内容，并对 PR 基线执行 Buf FILE breaking 检查。
- 开发允许 Go replace / Cargo path；发布必须引用固定协议版本，并移除本地路径依赖。
- 当前为本机免登录执行闭环：CLI 注册，Hub REST + gRPC/TLS，Native/A2A，Console 与 SQLite outbox。Hub 的 ACK 表示原子快照 fsync/rename 和状态更新已完成；当前使用单进程文件存储；后续 Hub 重构时接入认证授权与事务数据库。
- 当前 Hub 仅绑定 loopback，不区分开发/生产模式；原生路径、A2A 服务地址及其凭据仍由 daemon 管理。浏览器不通过 IPC 或原生管道直接调用 Agent。
- A2A v1 JSON-RPC 当前使用 GetTask 轮询和完整快照事件，Task/context 单独保存；原生 delta 与快照语义保持独立。支持标准 InputRequired 文本和服务声明的 a2alinker 结构化交互扩展。
- 收到 A2A 控制回复的传输错误时保留 delivery_unknown 回执，禁止自动重复投递；以 Task 后续查询确认状态。
- 生产认证、权限、凭据续期、数据库迁移、高级控制及未知执行人工恢复流程仍需实现。
