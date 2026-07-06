# Tasks

## 阶段 1：规范提取管道

- [ ] Task 1: 创建 `internal/extractor/` 包并实现核心提取器
  - [ ] SubTask 1.1: 定义提取器类型和接口（types.go）
  - [ ] SubTask 1.2: 实现从 api.json 读取端点骨架（extractor.go）
  - [ ] SubTask 1.3: 实现向 New API 发送请求并收集响应（auth.go + extractor.go）
  - [ ] SubTask 1.4: 实现从 JSON 响应推断 Schema（schema_infer.go）
  - [ ] SubTask 1.5: 实现合并回 OpenAPI 3.0.1 格式（merger.go）
  - [ ] SubTask 1.6: 编写提取器单元测试

- [ ] Task 2: 运行提取器生成新的 api.json
  - [ ] SubTask 2.1: 确保 New API 正在运行（localhost:4050）
  - [ ] SubTask 2.2: 执行提取器并输出新 api.json
  - [ ] SubTask 2.3: 验证生成的规范能被 kin-openapi 解析
  - [ ] SubTask 2.4: 对比新旧规范，确认关键字段已补全

- [ ] Task 3: 修复 relay.json
  - [ ] SubTask 3.1: 修复重复 operationId（createImage → createImageEdit + createImageGeneration）
  - [ ] SubTask 3.2: 补充 /v1/models 响应中的 success 和 supported_endpoint_types
  - [ ] SubTask 3.3: 补充 ChatCompletionRequest 缺失的 6 个字段
  - [ ] SubTask 3.4: 清理未使用的 schema 和 tag

## 阶段 2：解析器增强

- [ ] Task 4: 增强 parser.go 支持 $ref 解析
  - [ ] SubTask 4.1: 实现跨 schema 引用解析逻辑
  - [ ] SubTask 4.2: 处理嵌套引用（$ref → $ref 链）
  - [ ] SubTask 4.3: 编写 $ref 解析测试

- [ ] Task 5: 增强 parser.go 支持 nullable 和组合类型
  - [ ] SubTask 5.1: 处理 nullable 字段标记
  - [ ] SubTask 5.2: 处理 oneOf/anyOf 组合类型
  - [ ] SubTask 5.3: 将响应 schema 信息附加到 ToolDef
  - [ ] SubTask 5.4: 编写组合类型解析测试

- [ ] Task 6: 全量回归测试
  - [ ] SubTask 6.1: 用新 api.json 运行全部解析测试
  - [ ] SubTask 6.2: 修复因规范变化导致的测试失败
  - [ ] SubTask 6.3: `make test` 全部通过

## 阶段 3：高层面工具体系

- [ ] Task 7: 增强 add_channel 工具
  - [ ] SubTask 7.1: 在 InputSchema 中增加 mode（single/multi_to_single）和 multi_key_mode（polling/random）
  - [ ] SubTask 7.2: 实现多 key 模式的处理逻辑（key 按换行拆分）
  - [ ] SubTask 7.3: 编写多 key 场景测试

- [ ] Task 8: 新增 add_channel_keys 工具
  - [ ] SubTask 8.1: 创建 add_channel_keys.go（InputSchema + Handler）
  - [ ] SubTask 8.2: 向 PUT /api/channel/ 发送 key_mode=append 请求
  - [ ] SubTask 8.3: 在 register.go 中注册新工具
  - [ ] SubTask 8.4: 编写 add_channel_keys 测试

- [ ] Task 9: 新增令牌管理工具集
  - [ ] SubTask 9.1: 实现 create_token 工具
  - [ ] SubTask 9.2: 实现 revoke_token 工具
  - [ ] SubTask 9.3: 实现 list_tokens 工具
  - [ ] SubTask 9.4: 在 register.go 中注册三件套
  - [ ] SubTask 9.5: 编写令牌管理测试

- [ ] Task 10: 新增日志查询工具（可选，如果时间允许）
  - [ ] SubTask 10.1: 实现 search_logs 工具
  - [ ] SubTask 10.2: 实现 get_log_stats 工具
  - [ ] SubTask 10.3: 在 register.go 中注册

## 阶段 4：集成测试

- [ ] Task 11: 编写集成测试
  - [ ] SubTask 11.1: 创建 `internal/hightools/integration_test.go`
  - [ ] SubTask 11.2: 测试创建多 key 渠道端到端
  - [ ] SubTask 11.3: 测试追加渠道 key 端到端
  - [ ] SubTask 11.4: 测试令牌管理端到端

- [ ] Task 12: 更新 E2E 测试脚本
  - [ ] SubTask 12.1: 更新 scripts/test-e2e.py 增加关键业务流程验证
  - [ ] SubTask 12.2: 验证 `make test-e2e` 通过

# Task Dependencies

- [Task 2] depends on [Task 1]
- [Task 4] depends on [Task 2]
- [Task 5] depends on [Task 4]
- [Task 6] depends on [Task 3, Task 5]
- [Task 7] depends on [Task 6]
- [Task 8] depends on [Task 6]
- [Task 9] depends on [Task 6]
- [Task 10] depends on [Task 6]
- [Task 11] depends on [Task 7, Task 8, Task 9]
- [Task 12] depends on [Task 11]