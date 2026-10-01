package revplan

import "strings"

// SplitLines 按 LF 拆分原始文本为逐行内容，接受 LF 或 CRLF 换行：
// CRLF 中紧跟 LF 的 CR 被视为行尾一部分并去除。
// 末尾换行不额外产生空行，因此空文件返回空切片（零行）。
// 返回的每个行内容均不含换行符。
func SplitLines(data string) []string {
	if len(data) == 0 {
		return nil
	}
	parts := strings.Split(data, "\n")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	for i, line := range parts {
		if strings.HasSuffix(line, "\r") {
			parts[i] = line[:len(line)-1]
		}
	}
	return parts
}

// JoinLines 将行内容以 LF 拼接。每个结果行末尾都有换行；
// 没有行时返回空字符串（输出空文件）。
func JoinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
