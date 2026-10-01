package textrev

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveResult holds the verified source path and its normalized plan path.
type ResolveResult struct {
	// PlanPath is the slash-cleaned relative path used to detect duplicates
	// and to place the result under the output directory.
	PlanPath string
	// SrcPath is the absolute path of the file inside the source directory.
	SrcPath string
}

// ResolveFiles validates every planned path against srcDir and rejects
// absolute paths, escapes, symlinks and duplicates after normalization.
// Source files are opened (Lstat) but not read here.
func ResolveFiles(srcDir string, plan *Plan) ([]ResolveResult, error) {
	absSrc, err := filepath.Abs(srcDir)
	if err != nil {
		return nil, fmt.Errorf("resolve source dir: %w", err)
	}
	seen := make(map[string]string)
	results := make([]ResolveResult, 0, len(plan.Files))
	for _, fp := range plan.Files {
		res, err := resolveOne(absSrc, fp.Path)
		if err != nil {
			return nil, fmt.Errorf("file %s: %w", fp.Path, err)
		}
		if other, dup := seen[res.PlanPath]; dup {
			return nil, fmt.Errorf("file %s: duplicates normalized path of %s", fp.Path, other)
		}
		seen[res.PlanPath] = fp.Path
		results = append(results, res)
	}
	return results, nil
}

func resolveOne(absSrc, rel string) (ResolveResult, error) {
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, `/`) || strings.HasPrefix(rel, `\`) {
		return ResolveResult{}, fmt.Errorf("absolute paths are not allowed")
	}
	cleaned := filepath.Clean(rel)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return ResolveResult{}, fmt.Errorf("path escapes source directory")
	}
	target := filepath.Join(absSrc, cleaned)
	relToSrc, err := filepath.Rel(absSrc, target)
	if err != nil || relToSrc == ".." || strings.HasPrefix(relToSrc, ".."+string(filepath.Separator)) {
		return ResolveResult{}, fmt.Errorf("path escapes source directory")
	}
	if err := rejectSymlinkComponents(absSrc, cleaned); err != nil {
		return ResolveResult{}, err
	}
	info, err := os.Lstat(target)
	if err != nil {
		return ResolveResult{}, fmt.Errorf("stat source file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return ResolveResult{}, fmt.Errorf("symbolic links are not allowed")
	}
	if info.IsDir() {
		return ResolveResult{}, fmt.Errorf("path is a directory")
	}
	planPath := filepath.ToSlash(relToSrc)
	return ResolveResult{PlanPath: planPath, SrcPath: target}, nil
}

// rejectSymlinkComponents walks the path relative to root and rejects any
// intermediate component that is itself a symbolic link.
func rejectSymlinkComponents(root, cleaned string) error {
	parts := strings.Split(filepath.ToSlash(cleaned), "/")
	cur := root
	for i, part := range parts {
		if i == len(parts)-1 {
			break
		}
		cur = filepath.Join(cur, part)
		info, err := os.Lstat(cur)
		if err != nil {
			return fmt.Errorf("stat path component: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic links are not allowed in path")
		}
	}
	return nil
}
