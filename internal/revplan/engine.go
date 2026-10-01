package revplan

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// FileResult 是单个文件的修订结果摘要。
type FileResult struct {
	Path      string `json:"path"`
	OldLines  int    `json:"oldLines"`
	NewLines  int    `json:"newLines"`
	Revisions int    `json:"revisions"`
}

// Summary 是一次成功执行的 JSON 摘要。
type Summary struct {
	Files      []FileResult `json:"files"`
	TotalFiles int          `json:"totalFiles"`
	TotalRevs  int          `json:"totalRevisions"`
}

type preparedFile struct {
	result locateResult
	lines  []string
	revs   []Revision
	out    []string
}

// Check 校验整个计划：路径安全、文件存在、修订字段、原文匹配与区间冲突。
// 不写入任何文件。返回所有已定位文件的预处理结果与全部错误。
func (p *Plan) Check(srcDir string) ([]*preparedFile, ErrorList) {
	var errs ErrorList
	seen := make(map[string]int) // 规范化路径 -> 文件序号
	prepared := make([]*preparedFile, 0, len(p.Files))

	for fi := range p.Files {
		fp := &p.Files[fi]
		fileNo := fi + 1

		loc, err := safeLocate(srcDir, fp.Path)
		if err != nil {
			errs = append(errs, &Error{FileIndex: fileNo, Path: fp.Path, Reason: err.Error()})
			continue
		}
		if prev, ok := seen[loc.rel]; ok {
			errs = append(errs, &Error{FileIndex: fileNo, Path: fp.Path,
				Reason: fmt.Sprintf("规范化后与文件#%d重复", prev)})
			continue
		}
		seen[loc.rel] = fileNo

		data, err := os.ReadFile(loc.abs)
		if err != nil {
			errs = append(errs, &Error{FileIndex: fileNo, Path: fp.Path, Reason: "读取失败: " + err.Error()})
			continue
		}
		lines := SplitLines(string(data))
		pf := &preparedFile{result: loc, lines: lines, revs: fp.Revisions}
		checkFile(pf, fileNo, &errs)
		prepared = append(prepared, pf)
	}
	return prepared, errs
}

func checkFile(pf *preparedFile, fileNo int, errs *ErrorList) {
	n := len(pf.lines)
	// 替换区间列表（闭区间，行号从 1 开始），用于冲突检测。
	type span struct{ lo, hi int }
	var spans []span

	for ri := range pf.revs {
		rev := &pf.revs[ri]
		revNo := ri + 1
		addErr := func(format string, args ...any) {
			*errs = append(*errs, &Error{FileIndex: fileNo, RevisionIndex: revNo,
				Path: pf.result.rel, Reason: fmt.Sprintf(format, args...)})
		}

		switch rev.Op {
		case OpReplace:
			if rev.StartLine < 1 || rev.EndLine < rev.StartLine || rev.EndLine > n {
				addErr("replace 区间越界: [%d,%d]，文件共 %d 行", rev.StartLine, rev.EndLine, n)
			} else {
				if len(rev.Original) != rev.EndLine-rev.StartLine+1 {
					addErr("原文行数 %d 与区间长度 %d 不一致",
						len(rev.Original), rev.EndLine-rev.StartLine+1)
				} else {
					for k, want := range rev.Original {
						if strings.ContainsRune(want, '\n') || strings.ContainsRune(want, '\r') {
							addErr("原文第 %d 行含换行符", k+1)
							break
						}
						if got := pf.lines[rev.StartLine-1+k]; got != want {
							addErr("原文不匹配于第 %d 行: 期望 %q，实际 %q",
								rev.StartLine+k, want, got)
							break
						}
					}
				}
				for _, rl := range rev.Replacement {
					if strings.ContainsRune(rl, '\n') || strings.ContainsRune(rl, '\r') {
						addErr("替换文本含换行符")
						break
					}
				}
				for _, s := range spans {
					if rev.StartLine <= s.hi && s.lo <= rev.EndLine {
						addErr("replace 区间 [%d,%d] 与 [%d,%d] 相交",
							rev.StartLine, rev.EndLine, s.lo, s.hi)
						break
					}
				}
				spans = append(spans, span{rev.StartLine, rev.EndLine})
			}
		case OpInsert:
			if rev.At < 0 || rev.At > n {
				addErr("insert 边界 %d 越界，允许范围 [0,%d]", rev.At, n)
			} else if len(rev.Lines) == 0 {
				addErr("insert 内容为空")
			} else {
				for li, line := range rev.Lines {
					if strings.ContainsRune(line, '\n') || strings.ContainsRune(line, '\r') {
						addErr("插入内容第 %d 行含换行符", li+1)
						break
					}
				}
				for _, s := range spans {
					// 区间两端外侧（at==s.lo 在首行之前、at==s.hi 在末行之后）允许；
					// 严格内部 s.lo < at < s.hi 拒绝。
					if s.lo < rev.At && rev.At < s.hi {
						addErr("insert 边界 %d 位于 replace 区间 [%d,%d] 内部",
							rev.At, s.lo, s.hi)
						break
					}
				}
			}
		default:
			addErr("未知 op %q（应为 replace 或 insert）", rev.Op)
		}
	}
}

// Apply 在成功校验后为每个文件生成新行。结果仅依赖原始行号，
// 与各修订在计划中的排列顺序无关（同边界插入按计划顺序拼接）。
func apply(pf *preparedFile) {
	n := len(pf.lines)

	// 边界 -> 插入内容，按计划顺序拼接。
	inserts := make(map[int][]string)
	// 起始行 -> 替换行（nil 表示删除）。
	replStart := make(map[int][]string)
	replLen := make(map[int]int)
	for i := range pf.revs {
		rev := &pf.revs[i]
		switch rev.Op {
		case OpInsert:
			inserts[rev.At] = append(inserts[rev.At], rev.Lines...)
		case OpReplace:
			replStart[rev.StartLine] = rev.Replacement
			replLen[rev.StartLine] = rev.EndLine - rev.StartLine + 1
		}
	}

	var out []string
	out = append(out, inserts[0]...)
	for lineNo := 1; lineNo <= n; {
		var boundary int
		if rl, ok := replLen[lineNo]; ok {
			out = append(out, replStart[lineNo]...)
			boundary = lineNo + rl - 1
			lineNo += rl
		} else {
			out = append(out, pf.lines[lineNo-1])
			boundary = lineNo
			lineNo++
		}
		out = append(out, inserts[boundary]...)
	}
	pf.out = out
}

// orderedResults 按相对路径排序，保证输出与计划排列无关、稳定可复核。
func orderedResults(prepared []*preparedFile) []FileResult {
	sort.Slice(prepared, func(i, j int) bool { return prepared[i].result.rel < prepared[j].result.rel })
	results := make([]FileResult, 0, len(prepared))
	for _, pf := range prepared {
		results = append(results, FileResult{
			Path:      pf.result.rel,
			OldLines:  len(pf.lines),
			NewLines:  len(pf.out),
			Revisions: len(pf.revs),
		})
	}
	return results
}
