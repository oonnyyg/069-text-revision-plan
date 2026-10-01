# text-revision-plan

按 JSON 计划批量修订本地 UTF-8 文本的命令行工具，仅依赖 Go 标准库。

```
# 执行修订（-out 必须是尚不存在的全新目录）
go run ./cmd/textrev -plan plan.json -src ./source -out ./result

# 只校验，不创建任何输出
go run ./cmd/textrev -plan plan.json -src ./source -check

# 测试
go test ./...
```

计划格式、行语义、冲突与路径安全规则见 `docs/protocol.md`。
