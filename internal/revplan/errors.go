package revplan

import "fmt"

// Error 指向计划中的具体修订位置（均从 1 开始计数）。
type Error struct {
	FileIndex     int    // 文件在 plan.files 中的序号（从 1 开始）
	RevisionIndex int    // 修订在文件 revisions 中的序号（从 1 开始；0 表示文件级错误）
	Path          string // 计划中的原始路径
	Reason        string // 人类可读原因
}

func (e *Error) Error() string {
	if e.RevisionIndex > 0 {
		return fmt.Sprintf("文件#%d(%q) 修订#%d: %s", e.FileIndex, e.Path, e.RevisionIndex, e.Reason)
	}
	return fmt.Sprintf("文件#%d(%q): %s", e.FileIndex, e.Path, e.Reason)
}

// ErrorList 是一次校验收集到的全部错误。
type ErrorList []*Error

func (el ErrorList) Error() string {
	var b []byte
	for i, e := range el {
		if i > 0 {
			b = append(b, '\n')
		}
		b = append(b, e.Error()...)
	}
	return string(b)
}
