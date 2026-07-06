# list_users Spec

## Why

当前 MCP 服务器提供 160+ 原子 API 工具，但缺少面向用户管理的场景封装。管理员若要查看所有用户的概况，需要自行调用 `/api/user/` 并解析 JSON 响应。`list_users` 是用户管理的核心入口工具，聚合上游接口输出，以结构化表格展示用户列表，让 AI 能一句话获得用户全貌。

## What Changes

- 新增 `internal/hightools/list_users.go`：实现 `list_users` 工具的构造函数和 handler 逻辑
- 修改 `internal/hightools/register.go`：将 `list_users` 加入 `RegisterAll()` 返回列表
- 新增 `internal/hightools/list_users_test.go`：使用 `httptest.Server` mock 上游用户列表 API，验证输出内容

## Impact

- Affected specs: 用户管理模块
- Affected code:
  - `internal/hightools/list_users.go` — 新工具实现
  - `internal/hightools/list_users_test.go` — 新工具测试
  - `internal/hightools/register.go` — 注册 list_users（一行新增）

## ADDED Requirements

### Requirement: list_users 工具

系统 SHALL 提供一个名为 `list_users` 的 MCP 工具，用于列出所有用户并以表格展示。

#### Scenario: 正常获取用户列表

- **WHEN** 用户调用 `list_users` 工具
- **THEN** 系统调用 `GET /api/user/` 获取所有用户（默认 page_size=100）
- **AND** 以结构化文本形式返回用户列表，包含每个用户的：ID、用户名、角色、状态、分组、配额

#### Scenario: 上游返回空列表

- **WHEN** 上游 `GET /api/user/` 返回空数组 `items: []`
- **THEN** 系统返回提示信息 "没有找到任何用户"

#### Scenario: 上游返回错误

- **WHEN** 上游 `GET /api/user/` 返回 `success=false` 或非 200 状态码
- **THEN** 系统返回 `IsError=true`，包含错误描述

### Requirement: 列表展示格式

系统 SHALL 按 Markdown 表格格式展示用户列表：

| ID | 用户名 | 角色 | 状态 | 分组 | 已用配额 | 总配额 |
|----|--------|------|------|------|----------|--------|
| 1 | admin | 管理员 | ✅ | default | 5,000 | 1,000,000 |
| 2 | user1 | 普通用户 | ❌ | vip | 100 | 500,000 |

- 角色映射：1=普通用户，10=管理员，100=超级管理员
- 状态：status=1 显示 ✅ 启用，其他显示 ❌ 禁用
- 配额数字以千分位逗号分隔

### Requirement: 工具的 InputSchema

工具无可选参数，直接返回所有用户。

### Requirement: 工具注册

`list_users` SHALL 以 `list_users` 名称注册到 MCP 服务器，注册在 `hightools/register.go` 的 `RegisterAll()` 中。工具注册在 API 工具注册之后进行。