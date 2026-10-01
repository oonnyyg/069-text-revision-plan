package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func TestCLIEndToEnd(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain unavailable")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("a\nb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := filepath.Join(dir, "plan.json")
	planJSON := `{"files":[
	  {"path":"a.txt","revisions":[
	    {"type":"replace","start":2,"end":2,"lines":[{"original":"b","replacement":"B"}]},
	    {"type":"insert","boundary":2,"insert":["c"]}
	  ]}
	]}`
	if err := os.WriteFile(plan, []byte(planJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	cmd := exec.Command("go", "run", "./cmd/textrev", "-plan", plan, "-src", src, "-out", out)
	cmd.Dir = findModuleRoot(t)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("go run failed: %v\n%s", err, output)
	}
	var summary struct {
		Files []struct {
			Path          string `json:"path"`
			OriginalLines int    `json:"original_lines"`
			NewLines      int    `json:"new_lines"`
			Revisions     int    `json:"revisions"`
		} `json:"files"`
	}
	if err := json.Unmarshal(output, &summary); err != nil {
		t.Fatalf("invalid summary JSON %q: %v", output, err)
	}
	got := summary.Files[0]
	if got.OriginalLines != 2 || got.NewLines != 3 || got.Revisions != 2 {
		t.Fatalf("unexpected summary: %+v", got)
	}
	data, err := os.ReadFile(filepath.Join(out, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "a\nB\nc\n" {
		t.Fatalf("a.txt = %q", data)
	}
}

func TestCLIFailureNonZeroAndNoOutput(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain unavailable")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := filepath.Join(dir, "plan.json")
	bad := `{"files":[
	  {"path":"a.txt","revisions":[
	    {"type":"replace","start":1,"end":1,"lines":[{"original":"NOPE","replacement":"x"}]}
	  ]}
	]}`
	if err := os.WriteFile(plan, []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	cmd := exec.Command("go", "run", "./cmd/textrev", "-plan", plan, "-src", src, "-out", out)
	cmd.Dir = findModuleRoot(t)
	if err := cmd.Run(); err == nil {
		t.Fatal("expected non-zero exit on mismatch")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("output dir must not exist after failure: %v", err)
	}
}
