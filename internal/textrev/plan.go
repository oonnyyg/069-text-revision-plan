package textrev

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Revision struct {
	Type     string       `json:"type"`
	Start    int          `json:"start,omitempty"`
	End      int          `json:"end,omitempty"`
	Boundary int          `json:"boundary,omitempty"`
	Lines    []LineChange `json:"lines,omitempty"`
	Insert   []string     `json:"insert,omitempty"`
}

// LineChange pairs one original line with its replacement. An empty
// Replacement deletes the original line.
type LineChange struct {
	Original    string `json:"original"`
	Replacement string `json:"replacement"`
}

type FilePlan struct {
	Path      string     `json:"path"`
	Revisions []Revision `json:"revisions"`
}

type Plan struct {
	Files []FilePlan `json:"files"`
}

func LoadPlan(path string) (*Plan, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParsePlan(f)
}

func ParsePlan(r io.Reader) (*Plan, error) {
	var p Plan
	dec := json.NewDecoder(r)
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("parse plan: %w", err)
	}
	if err := p.validateShape(); err != nil {
		return nil, err
	}
	return &p, nil
}

func (p *Plan) validateShape() error {
	if len(p.Files) == 0 {
		return fmt.Errorf("plan contains no files")
	}
	for fi := range p.Files {
		fp := &p.Files[fi]
		if fp.Path == "" {
			return fmt.Errorf("file #%d: empty path", fi)
		}
		if len(fp.Revisions) == 0 {
			return fmt.Errorf("file %s: no revisions", fp.Path)
		}
		for ri, rev := range fp.Revisions {
			switch rev.Type {
			case "replace":
				if rev.Start <= 0 || rev.End < rev.Start {
					return fmt.Errorf("file %s revision #%d: invalid replace range [%d,%d]",
						fp.Path, ri, rev.Start, rev.End)
				}
				if len(rev.Lines) != rev.End-rev.Start+1 {
					return fmt.Errorf("file %s revision #%d: %d line entries for range of %d",
						fp.Path, ri, len(rev.Lines), rev.End-rev.Start+1)
				}
			case "insert":
				if rev.Boundary < 0 {
					return fmt.Errorf("file %s revision #%d: negative boundary %d",
						fp.Path, ri, rev.Boundary)
				}
				if len(rev.Insert) == 0 {
					return fmt.Errorf("file %s revision #%d: insert requires non-empty lines",
						fp.Path, ri)
				}
				for _, line := range rev.Insert {
					if line == "" {
						return fmt.Errorf("file %s revision #%d: inserted line is empty",
							fp.Path, ri)
					}
				}
			default:
				return fmt.Errorf("file %s revision #%d: unknown revision type %q",
					fp.Path, ri, rev.Type)
			}
		}
	}
	return nil
}
