package textrev

import (
	"reflect"
	"strings"
	"testing"
)

func TestReadLines(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"lf single", "a\n", []string{"a"}},
		{"lf no trailing", "a\nb", []string{"a", "b"}},
		{"crlf", "a\r\nb\r\n", []string{"a", "b"}},
		{"blank line preserved", "a\n\nb\n", []string{"a", "", "b"}},
		{"single no newline", "abc", []string{"abc"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ReadLines([]byte(c.in))
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %#v want %#v", got, c.want)
			}
		})
	}
}

func TestRenderLines(t *testing.T) {
	if got := RenderLines(nil); len(got) != 0 {
		t.Fatalf("empty input should be empty bytes, got %q", got)
	}
	if got := RenderLines([]string{"a", "b"}); string(got) != "a\nb\n" {
		t.Fatalf("got %q", got)
	}
}

func TestCRLFInputProducesLFOutput(t *testing.T) {
	original := ReadLines([]byte("a\r\nb\r\nc"))
	if !reflect.DeepEqual(original, []string{"a", "b", "c"}) {
		t.Fatalf("CRLF parse = %#v", original)
	}
	revs := []Revision{
		{Type: "replace", Start: 1, End: 2, Lines: []LineChange{
			{Original: "a", Replacement: "A"},
			{Original: "b", Replacement: ""},
		}},
		{Type: "insert", Boundary: 2, Insert: []string{"after-range"}},
	}
	got, err := Apply(original, revs)
	if err != nil {
		t.Fatal(err)
	}
	if string(RenderLines(got)) != "A\nafter-range\nc\n" {
		t.Fatalf("rendered = %q", RenderLines(got))
	}
}

func TestApplyReplaceInsertDelete(t *testing.T) {
	original := []string{"l1", "l2", "l3", "l4"}
	revs := []Revision{
		{Type: "replace", Start: 2, End: 2, Lines: []LineChange{{Original: "l2", Replacement: "l2"}}},
		{Type: "replace", Start: 3, End: 3, Lines: []LineChange{{Original: "l3", Replacement: ""}}},
		{Type: "insert", Boundary: 0, Insert: []string{"top"}},
		{Type: "insert", Boundary: 4, Insert: []string{"tail1"}},
		{Type: "insert", Boundary: 4, Insert: []string{"tail2"}},
	}
	got, err := Apply(original, revs)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"top", "l1", "l2", "l4", "tail1", "tail2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestApplyInsertAtRangeEnd(t *testing.T) {
	original := []string{"a", "b", "c"}
	revs := []Revision{
		{Type: "replace", Start: 1, End: 2, Lines: []LineChange{
			{Original: "a", Replacement: "a"},
			{Original: "b", Replacement: "b"},
		}},
		{Type: "insert", Boundary: 2, Insert: []string{"after-b"}},
	}
	got, err := Apply(original, revs)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "b", "after-b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestApplyOriginalMismatch(t *testing.T) {
	original := []string{"a", "b"}
	revs := []Revision{
		{Type: "replace", Start: 1, End: 2, Lines: []LineChange{
			{Original: "a", Replacement: "a"},
			{Original: "WRONG", Replacement: "x"},
		}},
	}
	_, err := Apply(original, revs)
	if err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
	if !strings.Contains(err.Error(), "revision #0") {
		t.Fatalf("error must identify revision position: %v", err)
	}
}

func TestValidateOverlappingRanges(t *testing.T) {
	revs := []Revision{
		{Type: "replace", Start: 1, End: 3, Lines: []LineChange{{Original: "a"}, {Original: "b"}, {Original: "c"}}},
		{Type: "replace", Start: 2, End: 4, Lines: []LineChange{{Original: "x"}, {Original: "y"}, {Original: "z"}}},
	}
	err := ValidateRevisions(revs, 4)
	if err == nil || !strings.Contains(err.Error(), "intersects") {
		t.Fatalf("expected intersection error, got %v", err)
	}
}

func TestValidateInsertInsideRange(t *testing.T) {
	revs := []Revision{
		{Type: "replace", Start: 1, End: 4, Lines: []LineChange{{Original: "a"}, {Original: "b"}, {Original: "c"}, {Original: "d"}}},
		{Type: "insert", Boundary: 2, Insert: []string{"inside"}},
	}
	if err := ValidateRevisions(revs, 4); err == nil {
		t.Fatal("expected boundary-inside-range error")
	}
	// Only outer edges (start-1 and end) are allowed.
	for _, b := range []int{0, 4} {
		ok := []Revision{
			{Type: "replace", Start: 1, End: 4, Lines: []LineChange{{Original: "a"}, {Original: "b"}, {Original: "c"}, {Original: "d"}}},
			{Type: "insert", Boundary: b, Insert: []string{"edge"}},
		}
		if err := ValidateRevisions(ok, 4); err != nil {
			t.Fatalf("boundary %d should be allowed: %v", b, err)
		}
	}
}
