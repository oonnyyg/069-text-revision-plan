package revplan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// locateResult 是一个通过安全校验的文件定位结果。
type locateResult struct {
	// rel 是规范化后的相对路径（正斜杠），用于输出目录与去重。
	rel string
	// abs 是来源文件的绝对路径。
	abs string
}

// safeLocate 将计划中的相对路径解析为来源目录内的安全绝对路径。
// 拒绝绝对路径、空路径、越出来源的路径；路径各组成部分均不得为符号链接。
// 调用方还需保证最终文件存在且本身不是符号链接。
func safeLocate(srcDir, rel string) (locateResult, error) {
	if rel == "" {
		return locateResult{}, fmt.Errorf("路径为空")
	}
	if strings.ContainsRune(rel, 0) {
		return locateResult{}, fmt.Errorf("路径包含非法字符")
	}
	// 以正斜杠作为计划路径分隔符；拒绝反斜杠与 Windows 盘符等绝对形式。
	if filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" ||
		strings.HasPrefix(rel, `/`) || strings.HasPrefix(rel, `\`) {
		return locateResult{}, fmt.Errorf("路径必须为相对路径: %q", rel)
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return locateResult{}, fmt.Errorf("路径越出来源目录: %q", rel)
	}
	if clean == "." {
		return locateResult{}, fmt.Errorf("路径不是文件: %q", rel)
	}

	absSrc, err := filepath.Abs(srcDir)
	if err != nil {
		return locateResult{}, err
	}
	abs := filepath.Join(absSrc, clean)
	relToSrc, err := filepath.Rel(absSrc, abs)
	if err != nil || relToSrc == ".." ||
		strings.HasPrefix(relToSrc, ".."+string(filepath.Separator)) {
		return locateResult{}, fmt.Errorf("路径越出来源目录: %q", rel)
	}

	// 逐级检查来源根到目标之间是否存在符号链接。
	cur := absSrc
	segs := strings.Split(clean, string(filepath.Separator))
	for i, seg := range segs {
		cur = filepath.Join(cur, seg)
		info, err := os.Lstat(cur)
		if err != nil {
			if os.IsNotExist(err) {
				return locateResult{}, fmt.Errorf("文件不存在: %q", rel)
			}
			return locateResult{}, err
		}
		isLast := i == len(segs)-1
		if info.Mode()&os.ModeSymlink != 0 {
			return locateResult{}, fmt.Errorf("路径包含符号链接: %q", rel)
		}
		if !isLast && !info.IsDir() {
			return locateResult{}, fmt.Errorf("路径中间成分不是目录: %q", rel)
		}
		if isLast && info.IsDir() {
			return locateResult{}, fmt.Errorf("路径指向目录而非文件: %q", rel)
		}
	}

	return locateResult{rel: filepath.ToSlash(relToSrc), abs: abs}, nil
}
