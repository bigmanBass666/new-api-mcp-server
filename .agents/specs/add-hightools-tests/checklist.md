# 检查清单

- [x] `add_channel_test.go` 存在，包含成功和所有错误路径测试
- [x] `toggle_channel_test.go` 存在，包含启用/禁用和所有错误路径测试
- [x] `toggle_user_status_test.go` 存在，包含启用/禁用和所有错误路径测试
- [x] `switch_group_test.go` 存在，包含成功和所有错误路径测试
- [x] 每个测试文件都使用 `httptest.NewServer` mock upstream
- [x] 每个测试文件都验证了 HTTP 方法、路径、请求体内容
- [x] `go test ./internal/hightools/ -v -count=1` 全部通过
- [x] `go build ./cmd/server` 编译无错误