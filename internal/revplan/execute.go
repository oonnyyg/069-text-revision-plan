package revplan

import (
	"fmt"
	"os"
	"path/filepath"
)

// Execute 校验计划并将修订结果写入全新的输出目录。
// 只有全部文件与修订都通过校验后才会创建输出目录；
// 输出目录必须尚不存在。只输出计划涉及的文件，并保留相对目录结构。
func (p *Plan) Execute(srcDir, outDir string) (*Summary, error) {
	prepared, errs := p.Check(srcDir)
	if len(errs) > 0 {
		return nil, errs
	}

	for _, pf := range prepared {
		apply(pf)
	}

	absOut, err := filepath.Abs(outDir)
	if err != nil {
		return nil, err
	}
	switch _, err := os.Stat(absOut); {
	case err == nil:
		return nil, fmt.Errorf("输出目录已存在: %q", outDir)
	case !os.IsNotExist(err):
		return nil, err
	}

	// 先在目标父目录下创建根目录，再逐个建子目录、写文件。
	if err := os.Mkdir(absOut, 0o755); err != nil {
		return nil, err
	}
	for _, pf := range prepared {
		dst := filepath.Join(absOut, filepath.FromSlash(pf.result.rel))
		if dir := filepath.Dir(dst); dir != absOut {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, err
			}
		}
		if err := os.WriteFile(dst, []byte(JoinLines(pf.out)), 0o644); err != nil {
			return nil, err
		}
	}

	results := orderedResults(prepared)
	summary := &Summary{Files: results, TotalFiles: len(results)}
	for _, r := range results {
		summary.TotalRevs += r.Revisions
	}
	return summary, nil
}

// ValidateOnly 只校验计划、生成摘要而不创建或写入任何输出。
func (p *Plan) ValidateOnly(srcDir string) (*Summary, error) {
	prepared, errs := p.Check(srcDir)
	if len(errs) > 0 {
		return nil, errs
	}
	for _, pf := range prepared {
		apply(pf)
	}
	results := orderedResults(prepared)
	summary := &Summary{Files: results, TotalFiles: len(results)}
	for _, r := range results {
		summary.TotalRevs += r.Revisions
	}
	return summary, nil
}
