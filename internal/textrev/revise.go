package textrev

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// FileResult summarizes a single processed file.
type FileResult struct {
	Path          string `json:"path"`
	OriginalLines int    `json:"original_lines"`
	NewLines      int    `json:"new_lines"`
	RevisionCount int    `json:"revisions"`
}

// Summary is the JSON report printed on success.
type Summary struct {
	Files []FileResult `json:"files"`
}

// op is one normalized revision positioned against the original file.
type op struct {
	index    int
	kind     string
	start    int // replace: first original line (1-based)
	end      int // replace: last original line (1-based)
	boundary int // insert: boundary (0..n)
	changes  []LineChange
	inserts  []string
}

// ReadLines splits content into logical lines.
// Both LF and CRLF are accepted; a trailing newline adds no extra line and
// an empty file yields zero lines.
func ReadLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	r := bufio.NewReader(bytes.NewReader(data))
	var lines []string
	for {
		line, err := r.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimSuffix(line, "\n")
			line = strings.TrimSuffix(line, "\r")
			lines = append(lines, line)
		}
		if err != nil {
			break
		}
	}
	return lines
}

// RenderLines joins lines with LF, terminating every line. No lines produces
// an empty byte slice.
func RenderLines(lines []string) []byte {
	if len(lines) == 0 {
		return []byte{}
	}
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

// ValidateRevisions checks range bounds, overlaps and boundary placement.
// It does not read file contents; check Apply for original-text matching.
func ValidateRevisions(revs []Revision, lineCount int) error {
	ops := make([]op, 0, len(revs))
	for i, rev := range revs {
		switch rev.Type {
		case "replace":
			if rev.Start > lineCount || rev.End > lineCount {
				return fmt.Errorf("revision #%d: replace range [%d,%d] exceeds %d original lines",
					i, rev.Start, rev.End, lineCount)
			}
			ops = append(ops, op{index: i, kind: "replace", start: rev.Start, end: rev.End, changes: rev.Lines})
		case "insert":
			if rev.Boundary > lineCount {
				return fmt.Errorf("revision #%d: insert boundary %d exceeds %d original lines",
					i, rev.Boundary, lineCount)
			}
			ops = append(ops, op{index: i, kind: "insert", boundary: rev.Boundary, inserts: rev.Insert})
		}
	}
	// Detect intersecting replace ranges.
	replaces := make([]op, 0)
	for _, o := range ops {
		if o.kind == "replace" {
			replaces = append(replaces, o)
		}
	}
	sort.Slice(replaces, func(a, b int) bool { return replaces[a].start < replaces[b].start })
	for i := 1; i < len(replaces); i++ {
		if replaces[i].start <= replaces[i-1].end {
			return fmt.Errorf("revision #%d: replace range [%d,%d] intersects revision #%d range [%d,%d]",
				replaces[i].index, replaces[i].start, replaces[i].end,
				replaces[i-1].index, replaces[i-1].start, replaces[i-1].end)
		}
	}
	// An insert boundary is forbidden strictly inside a replace range.
	for _, o := range ops {
		if o.kind != "insert" {
			continue
		}
		for _, r := range replaces {
			if o.boundary >= r.start && o.boundary < r.end {
				return fmt.Errorf("revision #%d: insert boundary %d lies inside replace range [%d,%d] of revision #%d",
					o.index, o.boundary, r.start, r.end, r.index)
			}
		}
	}
	return nil
}

// Apply validates original text for every replace, then builds the new lines.
// All positioning refers to the original file. Inserts at the same boundary
// are concatenated in plan order; otherwise revision order does not matter.
func Apply(original []string, revs []Revision) ([]string, error) {
	if err := ValidateRevisions(revs, len(original)); err != nil {
		return nil, err
	}
	replByStart := make(map[int]op)
	insertsByBoundary := make(map[int][]op)
	for i, rev := range revs {
		switch rev.Type {
		case "replace":
			replByStart[rev.Start] = op{index: i, kind: "replace", start: rev.Start, end: rev.End, changes: rev.Lines}
		case "insert":
			insertsByBoundary[rev.Boundary] = append(insertsByBoundary[rev.Boundary],
				op{index: i, kind: "insert", boundary: rev.Boundary, inserts: rev.Insert})
		}
	}
	// Verify original text for every replace before producing output.
	starts := make([]int, 0, len(replByStart))
	for s := range replByStart {
		starts = append(starts, s)
	}
	sort.Ints(starts)
	for _, s := range starts {
		o := replByStart[s]
		for ln := o.start; ln <= o.end; ln++ {
			orig := original[ln-1]
			want := o.changes[ln-o.start].Original
			if orig != want {
				return nil, fmt.Errorf("revision #%d: original text mismatch at line %d: got %q want %q",
					o.index, ln, orig, want)
			}
		}
	}
	// Emit: inserts before boundary 0, then for each original line either
	// skip/replace within a range or copy the line.
	var out []string
	out = append(out, collectInserts(insertsByBoundary, 0)...)
	i := 1
	for i <= len(original) {
		if o, ok := replByStart[i]; ok {
			for _, ch := range o.changes {
				if ch.Replacement != "" {
					out = append(out, ch.Replacement)
				}
			}
			i = o.end
			// Boundary at the range end is exactly after original line o.end.
			out = append(out, collectInserts(insertsByBoundary, i)...)
			i++
			continue
		}
		out = append(out, original[i-1])
		i++
		out = append(out, collectInserts(insertsByBoundary, i-1)...)
	}
	return out, nil
}

func collectInserts(byBoundary map[int][]op, boundary int) []string {
	ops := byBoundary[boundary]
	sort.SliceStable(ops, func(a, b int) bool { return ops[a].index < ops[b].index })
	var lines []string
	for _, o := range ops {
		lines = append(lines, o.inserts...)
	}
	return lines
}
