package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"example.com/text-revision-plan/internal/revplan"
)

func TestCLISuccessAndCheck(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	out := filepath.Join(root, "out")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "f.txt"), []byte("a\r\nb\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := revplan.Plan{Files: []revplan.FilePlan{{
		Path:      "f.txt",
		Revisions: []revplan.Revision{{Op: revplan.OpReplace, StartLine: 1, EndLine: 1, Original: []string{"a"}, Replacement: []string{"A"}}},
	}}}
	planPath := filepath.Join(root, "plan.json")
	writeJSON(t, planPath, plan)

	// check 模式不产生输出目录。
	code := run([]string{"-plan", planPath, "-src", src, "-check"})
	if code != 0 {
		t.Fatal("check 应成功")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("check 模式不应创建 out（此处从未指定）")
	}

	if code := run([]string{"-plan", planPath, "-src", src, "-out", out}); code != 0 {
		t.Fatal("执行应成功")
	}
	data, err := os.ReadFile(filepath.Join(out, "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "A\nb\n" {
		t.Fatalf("输出错误: %q", data)
	}

	// 输出目录已存在时必须失败。
	if code := run([]string{"-plan", planPath, "-src", src, "-out", out}); code != 1 {
		t.Fatalf("已存在输出目录应失败, code=%d", code)
	}
}

func TestCLIMismatchFailure(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "f.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(root, "plan.json")
	writeJSON(t, planPath, revplan.Plan{Files: []revplan.FilePlan{{
		Path:      "f.txt",
		Revisions: []revplan.Revision{{Op: revplan.OpReplace, StartLine: 1, EndLine: 1, Original: []string{"y"}}},
	}}})

	out := filepath.Join(root, "out")
	if code := run([]string{"-plan", planPath, "-src", src, "-out", out}); code != 1 {
		t.Fatalf("原文失配应非零退出, code=%d", code)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("失败时不得创建输出目录")
	}
}

func TestSymlinkRejected(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "outside.txt")
	if err := os.WriteFile(target, []byte("secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(src, "link.txt")); err != nil {
		if os.IsPermission(err) && !canMakeSymlink() {
			t.Skip("当前环境不允许创建符号链接")
		}
		t.Fatal(err)
	}
	planPath := filepath.Join(root, "plan.json")
	writeJSON(t, planPath, revplan.Plan{Files: []revplan.FilePlan{{Path: "link.txt"}}})
	plan, perr := revplan.LoadFile(planPath)
	if perr != nil {
		t.Fatal(perr)
	}
	if _, err := plan.ValidateOnly(src); err == nil {
		t.Fatal("符号链接必须被拒绝")
	}
}

func canMakeSymlink() bool {
	dir, err := os.MkdirTemp("", "symtest")
	if err != nil {
		return false
	}
	defer os.RemoveAll(dir)
	return os.Symlink(filepath.Join(dir, "a"), filepath.Join(dir, "b")) == nil
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

var _ = exec.Command
