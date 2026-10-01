// Package revplan 读取并校验 JSON 文本修订计划，并将修订应用到来源文本。
package revplan

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Op 是修订动作类型。
type Op string

const (
	// OpReplace 用替换文本整体替换 [StartLine, EndLine] 闭区间内的原始行；
	// 替换文本为空表示删除这些行。
	OpReplace Op = "replace"
	// OpInsert 在原始行之间的边界插入非空文本行。
	OpInsert Op = "insert"
)

// Revision 是单条修订。两种动作复用同一结构：
// replace 使用 StartLine/EndLine/Original/Replacement；
// insert 使用 At/Lines。
type Revision struct {
	Op Op `json:"op"`

	// StartLine、EndLine 为 replace 的闭区间（原始文件从 1 开始的行号）。
	StartLine int `json:"startLine,omitempty"`
	EndLine   int `json:"EndLine,omitempty"`

	// At 为 insert 的边界：0 表示首行之前，原始行数表示末行之后，
	// n 表示第 n 行与第 n+1 行之间。
	At int `json:"at,omitempty"`

	// Original 为 replace 逐行原文，必须与源文件对应行完全一致。
	Original []string `json:"original,omitempty"`
	// Replacement 为 replace 的替换文本行，空表示删除整段区间。
	Replacement []string `json:"replacement,omitempty"`
	// Lines 为 insert 的插入行，必须非空且每行不含换行符。
	Lines []string `json:"lines,omitempty"`
}

// FilePlan 是单个文件的全部修订。
type FilePlan struct {
	// Path 为相对来源目录的 UTF-8 文本路径，使用正斜杠。
	Path      string     `json:"path"`
	Revisions []Revision `json:"revisions"`
}

// Plan 是完整的批量修订计划。
type Plan struct {
	Files []FilePlan `json:"files"`
}

// LoadFile 从文件加载计划。
func LoadFile(name string) (*Plan, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Load(f)
}

// Load 解析 JSON 计划。未知字段将被拒绝。
func Load(r io.Reader) (*Plan, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var p Plan
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("解析计划失败: %w", err)
	}
	var extra json.RawMessage
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("解析计划失败: 顶层存在多余 JSON 值")
	}
	return &p, nil
}
