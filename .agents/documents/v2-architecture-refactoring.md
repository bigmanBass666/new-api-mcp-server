# v2 架构重构计划

## 概述

从"自动生成工具的幻觉"中醒来，建立以**真实 API 数据 + 业务路径驱动**的新架构。

核心转变：
- 旧：手写 OpenAPI JSON → 解析 → 残缺工具 → 用户踩坑
- 新：运行时提取规范 → 完整工具 → 高层面工具兜底关键业务

---

## 当前状态分析

### 审计结论

| 维度 | api.json (管理后台) | relay.json (模型接口) |
|------|-------------------|---------------------|
| 端点覆盖 | 基本完整 | 优秀 (58个端点) |
| Schema 质量 | **极差** | 中等 |
| 阻断性问题 | 3个 | 2个 |
| 重要问题 | 25+ | 4个 |
| 响应 schema 引用 | 全部缺失 | 部分缺失 |

### 核心问题

1. **OpenAPI 规范是手写的，不是自动生成的** → 整个"自动生成"的前提不成立
2. **Channel 模型缺 18 个字段**（key、channel_info、setting 等）→ 渠道管理工具不可用
3. **Log 模型缺 16/21 个字段** → 日志查询工具几乎不可用
4. **所有响应缺少 schema 引用** → 工具不知道返回什么
5. **字段名错误**：groups → group
6. **高层面工具是事后补救**，不是从业务路径出发设计的

### 保留的（好的部分）

- MCP 服务器框架（go-sdk 集成 + 传输层）
- 可观测性（Prometheus + OTel + slog）
- Docker 构建 + 健康检查
- HTTP 客户端（client.go，认证注入）
- 中间件（限流、CORS、body 限制）
- 现有的 10 个高层面工具（需要增强，但设计模式可用）

---

## 建议变更

### 阶段 1：规范提取管道（新包 `internal/extractor/`）

**做什么：** 构建一个工具，从运行中的 New API 实例自动提取完整的 OpenAPI 规范。

**为什么：** 手写 JSON 才是万恶之源。从真实 API 提取才能保证完整性和准确性。

**怎么做：**

```
internal/extractor/
├── extractor.go        # 主入口：读取现有端点列表，逐个发请求
├── schema_infer.go     # 从真实响应推断 Schema（字段名、类型、是否 nullable）
├── merger.go           # 将推断的 Schema 合并回 OpenAPI 格式
├── auth.go             # 认证信息注入（Authorization + New-Api-User）
└── types.go            # 提取器内部类型定义
```

**提取流程：**
1. 读现有 `api.json` 获取端点骨架（路径、方法、operationId）
2. 对每个端点发送真实请求到 `localhost:4050`
3. 从响应反推 JSON Schema（字段名、类型、嵌套结构）
4. 合并回 OpenAPI 3.0.1 格式
5. 时序字段标记为 `nullable`，整型标记 `minimum/maximum`
6. 输出新 `api.json`

**relay.json 的处理不同：** 不提取，改为引用 OpenAI 官方 OpenAPI 规范 + 手动修复审计发现的 2 个必须修复 + 4 个建议修复。

### 阶段 2：解析器增强（`internal/openapi/parser.go`）

**做什么：** 增强解析器以处理更丰富的规范。

**为什么：** 新规范将包含更多元数据（响应 schema、$ref 解析、nullable 标记），需要解析器能处理。

**改动：**
- 支持 `$ref` 解析（现有代码不处理跨 schema 引用）
- 支持 `nullable` 字段标记
- 支持 `oneOf`/`anyOf` 组合类型
- 将响应 schema 信息附加到 ToolDef 中（供高层面工具使用）
- 保留现有的 camelCase 命名和英文描述生成逻辑

### 阶段 3：高层面工具体系重构（`internal/hightools/`）

**做什么：** 从业务路径出发重新设计高层面工具，而不是 API 端点出发。

**为什么：** 旧的高层面工具是"自动生成不够 → 补几个"，新的应该是"用户要做什么 → 设计工具"。

**核心业务路径及对应工具：**

| 业务路径 | 工具 | 状态 |
|---------|------|------|
| 添加渠道（含多 key 轮转） | `add_channel`（增强） | 已有，需加 mode/multi_key_mode 参数 |
| 往已有渠道追加 key | `add_channel_keys`（新增） | 新增 |
| 查看渠道健康状态 | `list_providers` | 已有，完善 |
| 测试渠道并切换 | `test_and_report_channels` | 已有，完善 |
| 管理用户和配额 | `list_users` / `toggle_user_status` / `set_user_quota` | 已有，增强 |
| 管理令牌生命周期 | `create_token` / `revoke_token` / `list_tokens`（新增） | 新增 |
| 渠道批量操作 | `batch_toggle_channels` / `batch_set_priority`（新增） | 新增 |
| 查看日志和统计 | `search_logs` / `get_log_stats`（新增） | 新增 |
| 系统配置管理 | `get_system_options` / `update_system_option`（新增） | 新增 |

### 阶段 4：测试体系增强

**做什么：** 基于真实 New API 的集成测试。

**为什么：** 现有 125 个测试都是单元测试，没有真正验证过工具能否调用成功。

**新增：**
- `internal/extractor/extractor_test.go` — 提取器测试
- `internal/hightools/integration_test.go` — 集成测试（需要真实 API）
- 更新 `scripts/test-e2e.py` — 增加关键业务流程的 E2E 验证
- 使用 `-tags=integration` 区分集成测试和单元测试

---

## 假设与决策

### 决策记录

| 决策 | 选择 | 理由 |
|------|------|------|
| 规范提取方式 | admin 运行时提取 + relay 引用/手动修 | admin 质量太差必须重来，relay 问题少 |
| 与上游关系 | 原地重构，v2 commit 断代 | 仓库已独立，基础设施是好的 |
| 工具设计哲学 | 由外向内（业务路径驱动） | 旧哲学是反的 |
| 提取器语言 | Go | 与项目一致，可直接复用 client 和类型 |
| 依赖管理 | 不引入新依赖 | 现有 go-sdk + kin-openapi 足够 |

### 假设

- New API 运行在 `localhost:4050`（可配置）
- 提取器需要 admin token 访问全部管理端点
- 提取器不需要修改 New API 本身

---

## 验证步骤

### 阶段性验证

1. **阶段 1 验证：** 提取器能生成完整的 api.json，包含 Channel 模型的所有字段
2. **阶段 1 验证：** 生成的规范能被 `kin-openapi` 解析器加载
3. **阶段 2 验证：** 新解析器能处理所有 225+ 端点，生成正确的 ToolDef
4. **阶段 2 验证：** `make test` 通过
5. **阶段 3 验证：** 每个高层面工具都能对真实 API 调用成功
6. **阶段 3 验证：** `add_channel` 能创建多 key 轮转渠道
7. **阶段 3 验证：** `add_channel_keys` 能向已有渠道追加 key
8. **阶段 4 验证：** E2E 测试覆盖关键业务流程

### 回归验证

- 所有现有单元测试通过（125+）
- 现有 10 个高层面工具功能不变
- HTTP 传输层正常
- Docker 构建正常

---

## 文件变更清单

### 新增文件

| 文件 | 说明 |
|------|------|
| `internal/extractor/extractor.go` | 提取器主入口 |
| `internal/extractor/schema_infer.go` | 从响应推断 Schema |
| `internal/extractor/merger.go` | Schema 合并回 OpenAPI |
| `internal/extractor/auth.go` | 认证信息处理 |
| `internal/extractor/types.go` | 类型定义 |
| `internal/extractor/extractor_test.go` | 提取器测试 |
| `internal/hightools/add_channel_keys.go` | 追加渠道 Key 工具 |
| `internal/hightools/create_token.go` | 创建令牌工具 |
| `internal/hightools/revoke_token.go` | 撤销令牌工具 |
| `internal/hightools/list_tokens.go` | 列出令牌工具 |
| `internal/hightools/batch_toggle_channels.go` | 批量开关渠道 |
| `internal/hightools/search_logs.go` | 搜索日志工具 |
| `internal/hightools/get_log_stats.go` | 日志统计工具 |
| `internal/hightools/integration_test.go` | 集成测试 |
| `internal/hightools/add_channel_keys_test.go` | 追加 Key 工具测试 |

### 修改文件

| 文件 | 改动 |
|------|------|
| `openapi/api.json` | 被提取器重新生成 |
| `openapi/relay.json` | 引用官方规范 + 手动修复 |
| `internal/openapi/parser.go` | 增强：支持 $ref、nullable、响应 schema |
| `internal/openapi/parser_test.go` | 更新以匹配新规范 |
| `internal/hightools/add_channel.go` | 增加 mode/multi_key_mode 参数 |
| `internal/hightools/register.go` | 注册新工具 |
| `cmd/server/main.go` | 集成提取器（可选，可离线使用） |
| `scripts/test-e2e.py` | 增加关键业务流程测试 |
| `Makefile` | 添加提取器相关目标 |

### 无需改动

- `internal/client/` — HTTP 客户端设计良好
- `internal/middleware/` — 限流中间件
- `internal/observability/` — 可观测性
- `internal/registry/` — 注册逻辑
- `internal/handler/` — MCP → HTTP 映射
- `Dockerfile`、`docker-compose.yml` — 不变
- `.github/workflows/test.yml` — 不变