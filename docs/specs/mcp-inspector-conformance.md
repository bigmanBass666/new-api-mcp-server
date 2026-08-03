# Feat: 接入 MCP Inspector + Conformance 测试到开发/CI 流程

## Problem Statement

项目目前只有两类测试：单元测试（Go race detector）和 E2E 测试（Python + Docker Compose）。缺少对 MCP 协议规范本身的自动化验证，也缺少可视化的开发调试工具。这意味着：
- 工具注册、参数 schema、JSON-RPC 层面的协议偏差只能靠人工审查
- 调试新工具时需要写临时脚本或用 curl 拼原始 JSON-RPC 请求，效率低
- 无法直观验证客户端视角看到的工具列表、描述、schema 是否正确

## Solution

接入 MCP 官方生态中的两个工具：
1. **MCP Inspector** — 开发阶段的交互式调试工具，可视化浏览和调用工具
2. **MCP Conformance** — CI 中的协议规范合规测试，作为质量门控

## User Stories

1. As a developer, I want to visually inspect all registered tools and their schemas, so that I can verify tool definitions without writing temporary test scripts
2. As a developer, I want to call tools interactively with custom inputs and see raw JSON-RPC traffic, so that I can debug tool behavior and error responses
3. As a developer, I want the inspector to connect to both stdio and HTTP transports, so that I can debug whichever mode I'm working on
4. As a CI pipeline, I want to run MCP conformance tests against the server, so that protocol compliance is verified on every push
5. As a maintainer, I want conformance failures to block merges, so that spec-breaking changes are caught before reaching main
6. As a developer, I want conformance test results visible in CI artifacts, so that I can diagnose protocol-level failures without reproducing locally

## Implementation Decisions

- Inspector 通过 `npx @modelcontextprotocol/inspector` 运行，指向 server 的 HTTP transport（端口 4051/4050）。不需要改动任何 Go 代码
- Conformance 测试在 CI pipeline 中新增一个 stage，在 Docker 栈启动后、E2E 测试前运行。复用现有的 Docker Compose 环境
- Conformance 使用 TypeScript CLI（`npx @modelcontextprotocol/conformance`），CI runner 环境（ubuntu-latest）已具备 Node.js
- Conformance 测试场景：`server-initialize` + `core` suite，覆盖协议握手、工具发现、工具调用基础流程
- Conformance 测试超时设置为 60s，避免因上游 API 延迟导致误报
- Inspector 的接入方式记录在开发文档中，作为推荐的本地调试工作流
- 不升级 go-sdk 版本（保持 v1.4.1），conformance 测试通过黑盒方式验证协议合规，不依赖 SDK 版本

## Testing Decisions

- Conformance 测试是黑盒协议验证，与现有 E2E 测试（功能集成验证）互补而非重复
- 测试优先检查：initialize 握手、tools/list 响应完整性、tools/call 请求/响应格式
- 已知预期失败（如涉及 sampling/roots 的 deprecated 特性）通过 `--expected-failures` 文件基线化
- CI 中 conformance stage 失败会阻止 pipeline 继续（类似现有 E2E stage 的行为）
- Inspector 本身不做自动化测试，作为手动开发工具使用

## Out of Scope

- go-sdk 版本升级（当前 v1.4.1 已满足项目需求）
- MCP Registry 发布（目前处于 preview，API 未稳定）
- 将 conformance 测试翻译为 Go 原生测试（TypeScript CLI 已足够）
- Inspector 的 CI 集成（仅作为本地开发工具）

## Further Notes

- tasks 扩展声明（`io.modelcontextprotocol/tasks`）已在 `defaultCapabilities()` 中实现，无需额外改动
- 项目已有的 E2E 测试覆盖功能场景，conformance 覆盖的是协议层规范，两者不重叠
- Node.js 在 CI 环境（ubuntu-latest）中可用，无需额外安装步骤
