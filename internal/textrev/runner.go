package textrev

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Run validates every file and revision first; only if all checks pass does
// it create outDir and write result files. When dryRun is true no output
// directory is created.
func Run(planPath, srcDir, outDir string, dryRun bool) (*Summary, error) {
	plan, err := LoadPlan(planPath)
	if err != nil {
		return nil, err
	}
	resolved, err := ResolveFiles(srcDir, plan)
	if err != nil {
		return nil, err
	}
	absSrc, err := filepath.Abs(srcDir)
	if err != nil {
		return nil, err
	}
	var absOut string
	if !dryRun {
		absOut, err = filepath.Abs(outDir)
		if err != nil {
			return nil, fmt.Errorf("resolve output dir: %w", err)
		}
		if equalPath(absOut, absSrc) || isWithin(absOut, absSrc) || isWithin(absSrc, absOut) {
			return nil, fmt.Errorf("output directory must be separate from the source directory")
		}
	}

	type prepared struct {
		res   ResolveResult
		lines []string
		count int
	}
	preparedFiles := make([]prepared, 0, len(plan.Files))
	summary := &Summary{Files: make([]FileResult, 0, len(plan.Files))}

	for i, fp := range plan.Files {
		data, err := os.ReadFile(resolved[i].SrcPath)
		if err != nil {
			return nil, fmt.Errorf("file %s: read: %w", fp.Path, err)
		}
		original := ReadLines(data)
		newLines, err := Apply(original, fp.Revisions)
		if err != nil {
			return nil, fmt.Errorf("file %s: %w", fp.Path, err)
		}
		preparedFiles = append(preparedFiles, prepared{res: resolved[i], lines: newLines, count: len(fp.Revisions)})
		summary.Files = append(summary.Files, FileResult{
			Path:          resolved[i].PlanPath,
			OriginalLines: len(original),
			NewLines:      len(newLines),
			RevisionCount: len(fp.Revisions),
		})
	}

	if dryRun {
		return summary, nil
	}
	if _, err := os.Stat(absOut); err == nil {
		return nil, fmt.Errorf("output directory already exists: %s", absOut)
	}
	if err := os.MkdirAll(absOut, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}
	for _, pf := range preparedFiles {
		dst := filepath.Join(absOut, filepath.FromSlash(pf.res.PlanPath))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return nil, fmt.Errorf("file %s: %w", pf.res.PlanPath, err)
		}
		if err := os.WriteFile(dst, RenderLines(pf.lines), 0o644); err != nil {
			return nil, fmt.Errorf("file %s: write: %w", pf.res.PlanPath, err)
		}
	}
	return summary, nil
}

// MarshalSummary returns the canonical JSON summary with a trailing newline.
func MarshalSummary(s *Summary) ([]byte, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func equalPath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

func isWithin(target, base string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	if rel == "." || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." &&
		!(len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator))
}
