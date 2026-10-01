# 文本批量修订协议 (textrev)

`textrev` 读取一个 JSON 修订计划，对来源目录中的 UTF-8 文本文件做
行级替换、删除与插入，把结果写入一个全新的输出目录。来源目录永不修改。

仅使用 Go 标准库，入口为：

```
go run ./cmd/textrev -plan plan.json -src ./source -out ./result
```

只校验、不写文件：

```
go run ./cmd/textrev -plan plan.json -src ./source -check
```

成功时向标准输出打印 JSON 摘要；任何校验失败时向标准错误报告路径与
修订位置，进程以非零状态退出，并且不创建输出目录。

## 命令行参数

| 参数 | 说明 |
| --- | --- |
| `-plan` | JSON 修订计划文件路径（必填） |
| `-src`  | 来源目录（必填），计划中的相对路径都相对它解析 |
| `-out`  | 全新输出目录（非 `-check` 时必填），该目录不得已存在 |
| `-check` | 只执行全部校验并输出摘要，不创建输出目录 |

## 计划 JSON

顶层是 `files` 数组，每个元素对应一个文件：

```json
{
  "files": [
    {
      "path": "notes/chapter1.txt",
      "revisions": [
        {
          "type": "replace",
          "start": 3,
          "end": 4,
          "lines": [
            {"original": "old line 3", "replacement": "new line 3"},
            {"original": "old line 4", "replacement": ""}
          ]
        },
        {"type": "insert", "boundary": 0, "insert": ["版权声明行"]}
      ]
    }
  ]
}
```

### replace（替换 / 删除）

- `start`、`end` 是原始文件从 1 开始的闭区间行号。
- `lines` 长度必须等于 `end - start + 1`，按顺序逐行对应。
- 每个元素携带 `original`（原文）与 `replacement`（替换文本）。
- `original` 必须与对应原始行完全一致，否则该文件修订失败。
- `replacement` 为空字符串表示删除该行；非空则替换为该行。

### insert（插入）

- `boundary` 指定原始行之间的边界：`0` 为首行之前，等于原文件行数时
  为末行之后；`k` 表示在第 `k` 行之后、第 `k+1` 行之前。
- `insert` 为非空字符串数组，元素不允许为空字符串。
- 同一边界上的多个插入按它们在计划中出现的顺序拼接。

## 定位与冲突规则

- 所有行号与边界都基于**原始文件**，与计划中修订的排列顺序无关。
- 同一文件的两个 replace 区间不得相交（端点相接允许）。
- insert 的边界不得落在某个 replace 区间“内部”：对区间
  `[start,end]`，边界 `b` 满足 `start <= b < end` 时被拒绝；
  只有外侧边界 `b == start-1`（整段之前）与 `b == end`（整段之后）允许。
- 校验顺序：先做计划形状、路径安全校验，再逐文件读取并核对原文与
  冲突；任一文件失败则整体失败、零输出。

## 路径安全

- 计划中的 `path` 必须是相对路径；绝对路径、盘符路径、以 `/` 或 `\`
  开头的路径一律拒绝。
- 规范化后越出来源目录的路径（如 `../x`）拒绝。
- 文件本身及其任何中间目录组件为符号链接时拒绝。
- 规范化（清理 `.`、`..`、分隔符）后指向同一文件的重复计划拒绝。
- 输出目录不得与来源目录相同或互为父/子目录。

## 行格式

- 输入接受 LF 或 CRLF；行内容本身不含换行符；CR 在行尾被剥离。
- 末尾换行不额外产生一行；空文件为零行。
- 输出统一使用 LF，每个结果行末尾都有 `\n`；零行结果输出空文件。

## 成功摘要

标准输出为带缩进的 JSON，例如：

```json
{
  "files": [
    {"path": "notes/chapter1.txt", "original_lines": 12, "new_lines": 11, "revisions": 3}
  ]
}
```

- 仅包含计划涉及的文件，相对目录结构原样保留到输出目录。
- `original_lines` / `new_lines` 为修订前、后的行数，
  `revisions` 为该文件的修订条目数。
