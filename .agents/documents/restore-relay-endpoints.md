# 恢复 relay.json 丢失的 20 个端点

## Phase 1: Explore — 完成

已从 git 历史查明丢失范围和内容。

### 当前状态

relay.json 从 Phase 0-2 提交（9c0018d）后丢失了 20 个端点：

**A. Midjourney 16 个端点** — 完整丢失
- POST /mj/submit/{imagine,describe,blend,change,simple-change,action,shorten,modal,edits,video,upload-discord-images}
- GET /mj/task/{id}/{fetch,image-seed}
- POST /mj/task/list-by-condition
- GET /mj/image/{id}
- POST /mj/insight-face/swap

**B. Gemini 扩展 3 个端点** — 需要改为新标签
- POST /v1beta/models/{model}:streamGenerateContent
- POST /v1beta/models/{model}:embedContent  
- POST /v1beta/models/{model}:countTokens
- 旧标签 `"Gemini"` → 当前用 `"Gemini格式"`

**C. 视频混剪 1 个端点** — 需要改为新标签  
- POST /v1/videos/{video_id}/remix
- 旧标签 `"OpenAI"` → 应改为 `"视频生成/Sora兼容格式"`

> 所有端点均使用内联 schema（无 $ref 引用），无需修改 components/schemas。

### 恢复策略

1. 用 Python 脚本从 git 历史（commit 105e30a）提取精确 JSON
2. 通过编程方式合并到当前 relay.json（避免手工编辑 7000+ 行 JSON 出错）
3. 创建合并脚本 `scripts/restore-relay-endpoints.py`，完成：
   - 添加 `Midjourney` tag
   - 插入 16 个 Midjourney 端点（tag→Midjourney）
   - 插入 3 个 Gemini 端点（tag→Gemini格式）
   - 插入 1 个视频混剪端点（tag→视频生成/Sora兼容格式）
4. 验证：`go build ./cmd/server` 编译通过
5. 运行集成测试验证工具注册正确

### 不做的
- 不改动 relay.json 现有内容（38 个端点保持原样）
- 不改动 relay.json 以外任何文件
- 不改动 OpenAPI 解析代码