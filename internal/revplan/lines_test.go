package revplan

import "testing"

func TestSplitLines(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"空文件零行", "", nil},
		{"仅换行视为一空行", "\n", []string{""}},
		{"LF 多行末尾换行", "a\nb\n", []string{"a", "b"}},
		{"LF 多行无末尾换行", "a\nb", []string{"a", "b"}},
		{"CRLF 多行", "a\r\nb\r\n", []string{"a", "b"}},
		{"CRLF 无末尾换行", "a\r\nb", []string{"a", "b"}},
		{"混合换行", "a\r\nb\nc\r\n", []string{"a", "b", "c"}},
		{"空行保留", "a\n\n\nb\n", []string{"a", "", "", "b"}},
		{"保留独立CR", "a\rb\n", []string{"a\rb"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SplitLines(c.in)
			if len(got) != len(c.want) {
				t.Fatalf("行数 = %d, 期望 %d (%q)", len(got), len(c.want), got)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("第 %d 行 = %q, 期望 %q", i+1, got[i], c.want[i])
				}
			}
		})
	}
}

func TestJoinLines(t *testing.T) {
	if got := JoinLines(nil); got != "" {
		t.Fatalf("空行应输出空文件, 得到 %q", got)
	}
	if got := JoinLines([]string{"a", "b"}); got != "a\nb\n" {
		t.Fatalf("得到 %q", got)
	}
	if got := JoinLines([]string{""}); got != "\n" {
		t.Fatalf("单行空行应输出一个换行, 得到 %q", got)
	}
}
