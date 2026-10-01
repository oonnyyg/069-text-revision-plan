package revplan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSrc(t *testing.T, src, rel, content string) {
	t.Helper()
	p := filepath.Join(src, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestApplyReplaceDeleteInsert(t *testing.T) {
	src := t.TempDir()
	writeSrc(t, src, "a.txt", "1\r\n2\r\n3\r\n4\r\n")

	plan := &Plan{Files: []FilePlan{{
		Path: "a.txt",
		Revisions: []Revision{
			{Op: OpInsert, At: 0, Lines: []string{"head"}},
			{Op: OpReplace, StartLine: 2, EndLine: 3,
				Original: []string{"2", "3"}, Replacement: []string{"two", "three"}},
			{Op: OpInsert, At: 4, Lines: []string{"tail"}},
		},
	}}}

	out := filepath.Join(t.TempDir(), "out")
	summary, err := plan.Execute(src, out)
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(out, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	want := "head\n1\ntwo\nthree\n4\ntail\n"
	if string(got) != want {
		t.Fatalf("结果 = %q, 期望 %q", got, want)
	}
	if summary.Files[0].OldLines != 4 || summary.Files[0].NewLines != 6 ||
		summary.Files[0].Revisions != 3 {
		t.Fatalf("摘要不正确: %+v", summary.Files[0])
	}
}

func TestDeleteAndEmptyFile(t *testing.T) {
	src := t.TempDir()
	writeSrc(t, src, "del.txt", "a\nb\nc\n")
	writeSrc(t, src, "empty.txt", "")

	plan := &Plan{Files: []FilePlan{
		{Path: "empty.txt", Revisions: []Revision{
			{Op: OpInsert, At: 0, Lines: []string{"x"}},
		}},
		{Path: "del.txt", Revisions: []Revision{
			{Op: OpReplace, StartLine: 1, EndLine: 3,
				Original: []string{"a", "b", "c"}, Replacement: nil},
		}},
	}}
	out := filepath.Join(t.TempDir(), "out")
	if _, err := plan.Execute(src, out); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "del.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Fatalf("全删应得到空文件, 实际 %q", data)
	}
	if data, _ = os.ReadFile(filepath.Join(out, "empty.txt")); string(data) != "x\n" {
		t.Fatalf("空文件插入错误: %q", data)
	}
}

func TestSameBoundaryInsertConcat(t *testing.T) {
	src := t.TempDir()
	writeSrc(t, src, "f.txt", "only\n")
	plan := &Plan{Files: []FilePlan{{
		Path: "f.txt",
		Revisions: []Revision{
			{Op: OpInsert, At: 0, Lines: []string{"a", "b"}},
			{Op: OpInsert, At: 0, Lines: []string{"c"}},
			{Op: OpInsert, At: 1, Lines: []string{"d"}},
			{Op: OpInsert, At: 1, Lines: []string{"e"}},
		},
	}}}
	out := filepath.Join(t.TempDir(), "out")
	if _, err := plan.Execute(src, out); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(out, "f.txt"))
	if want := "a\nb\nc\nonly\nd\ne\n"; string(data) != want {
		t.Fatalf("同边界拼接错误: %q, 期望 %q", data, want)
	}
}

func TestOrderIndependenceAndSameBoundaryConcat(t *testing.T) {
	src := t.TempDir()
	writeSrc(t, src, "f.txt", "a\nb\n")
	base := []Revision{
		{Op: OpReplace, StartLine: 1, EndLine: 1, Original: []string{"a"}, Replacement: []string{"A"}},
		{Op: OpInsert, At: 0, Lines: []string{"i1"}},
		{Op: OpInsert, At: 2, Lines: []string{"i2"}},
	}
	revs2 := []Revision{base[2], base[0], base[1]}

	out1 := filepath.Join(t.TempDir(), "o1")
	out2 := filepath.Join(t.TempDir(), "o2")
	p1 := &Plan{Files: []FilePlan{{Path: "f.txt", Revisions: base}}}
	p2 := &Plan{Files: []FilePlan{{Path: "f.txt", Revisions: revs2}}}
	if _, err := p1.Execute(src, out1); err != nil {
		t.Fatal(err)
	}
	if _, err := p2.Execute(src, out2); err != nil {
		t.Fatal(err)
	}
	d1, _ := os.ReadFile(filepath.Join(out1, "f.txt"))
	d2, _ := os.ReadFile(filepath.Join(out2, "f.txt"))
	// 不同位置修订重排不改变结果。
	if string(d1) != string(d2) || string(d1) != "i1\nA\nb\ni2\n" {
		t.Fatalf("重排结果不一致: %q vs %q", d1, d2)
	}
}

func TestValidationFailures(t *testing.T) {
	src := t.TempDir()
	writeSrc(t, src, "m.txt", "one\ntwo\nthree\n")

	cases := []struct {
		name string
		rev  Revision
		msg  string
	}{
		{"原文失配", Revision{Op: OpReplace, StartLine: 1, EndLine: 1,
			Original: []string{"ONE"}}, "原文不匹配"},
		{"区间相交", Revision{Op: OpReplace, StartLine: 2, EndLine: 2,
			Original: []string{"two"}, Replacement: []string{"2"}}, ""}, // 与下一条冲突
		{"插入位于区间内部", Revision{Op: OpInsert, At: 2, Lines: []string{"x"}}, "内部"},
		{"插入为空", Revision{Op: OpInsert, At: 1, Lines: nil}, "insert 内容为空"},
		{"区间越界", Revision{Op: OpReplace, StartLine: 3, EndLine: 5,
			Original: []string{"a", "b", "c"}}, "越界"},
		{"未知op", Revision{Op: "nope"}, "未知 op"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			revs := []Revision{c.rev}
			if c.name == "插入位于区间内部" {
				revs = append([]Revision{{Op: OpReplace, StartLine: 1, EndLine: 3,
					Original: []string{"one", "two", "three"}}}, c.rev)
			}
			if c.name == "区间相交" {
				revs = append(revs, Revision{Op: OpReplace, StartLine: 2, EndLine: 3,
					Original: []string{"two", "three"}})
			}
			plan := &Plan{Files: []FilePlan{{Path: "m.txt", Revisions: revs}}}
			out := filepath.Join(t.TempDir(), "should-not-exist")
			_, err := plan.Execute(src, out)
			if err == nil {
				t.Fatal("期望校验失败")
			}
			if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
				t.Fatal("失败时不应创建输出目录")
			}
			if c.msg != "" && !strings.Contains(err.Error(), c.msg) {
				t.Fatalf("错误 %q 未包含 %q", err, c.msg)
			}
		})
	}
}

func TestMultiFileFailuresAndDuplicates(t *testing.T) {
	src := t.TempDir()
	writeSrc(t, src, "a.txt", "1\n2\n")
	writeSrc(t, src, "sub/b.txt", "x\n")

	plan := &Plan{Files: []FilePlan{
		{Path: "a.txt", Revisions: []Revision{
			{Op: OpReplace, StartLine: 1, EndLine: 1, Original: []string{"9"}}},
		},
		{Path: "missing.txt"},
		{Path: "../escape.txt"},
		{Path: "sub/../a.txt"}, // 规范化后与第一个文件重复
	}}
	_, err := plan.ValidateOnly(src)
	errs, ok := err.(ErrorList)
	if !ok {
		t.Fatalf("期望 ErrorList, 得到 %T", err)
	}
	if len(errs) != 4 {
		t.Fatalf("应收集到 4 个错误（失配/缺失/越界/重复）; 得到 %d:\n%v", len(errs), errs)
	}
	for _, e := range errs {
		if e.FileIndex == 0 {
			t.Fatal("错误必须携带文件位置")
		}
	}
}
