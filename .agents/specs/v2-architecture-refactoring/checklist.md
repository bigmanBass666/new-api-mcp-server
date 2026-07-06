# 验收清单

## 阶段 1：规范提取管道

- [ ] 提取器能连接到 localhost:4050 并成功获取数据
- [ ] 提取器生成的 Channel schema 包含 key、channel_info、setting 等全部 25+ 字段
- [ ] 提取器输出的 JSON 是有效的 OpenAPI 3.0.1 规范
- [ ] 生成的规范能被 kin-openapi 加载不报错
- [ ] 提取器不对 New API 做任何写操作
- [ ] relay.json 的重复 operationId 已修复
- [ ] relay.json 的 /v1/models 响应已补充 success 和 supported_endpoint_types

## 阶段 2：解析器增强

- [ ] 解析器能正确解析带 $ref 的嵌套 Schema
- [ ] 解析器能正确处理 nullable 字段标记
- [ ] 解析器能处理 oneOf/anyOf 组合类型
- [ ] 所有现有解析测试在新的规范下仍然通过
- [ ] `make test` 全部通过

## 阶段 3：高层面工具体系

- [ ] add_channel 支持 mode 参数创建多 key 轮转渠道
- [ ] add_channel 的旧用法（单 key）完全兼容
- [ ] add_channel_keys 能向已有渠道追加 key
- [ ] create_token/revoke_token/list_tokens 三件套功能正常
- [ ] 所有高层面工具在真实 New API 上调用成功

## 阶段 4：集成测试

- [ ] 集成测试覆盖创建多 key 渠道端到端流程
- [ ] 集成测试覆盖追加渠道 key 流程
- [ ] 集成测试覆盖令牌管理流程
- [ ] E2E 测试脚本包含关键业务流程验证
- [ ] `make test` 和 `make test-e2e` 全部通过