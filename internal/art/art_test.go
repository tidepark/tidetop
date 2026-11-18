package art

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/tidepark/tidetop/internal/feed"
	"github.com/tidepark/tidetop/internal/render"
)

var sgr = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func plain(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = sgr.ReplaceAllString(l, "")
	}
	return out
}

func TestCleanPickLoad(t *testing.T) {
	if got := Clean("a\x1b[2Jb\x1b]52;c;aGk=\x07c\u009b31md\x1b]0;x\x1b\\e\u0007f​g"); got != "abc31mdefg" {
		t.Errorf("Clean: %q", got)
	}
	pics := []Picture{{At: 0}, {At: 10}, {At: 50}}
	for _, c := range []struct{ v, want float64 }{{-3, 0}, {10, 10}, {49.9, 10}, {99, 50}} {
		if got := Pick(pics, c.v).At; got != c.want {
			t.Errorf("Pick(%v) = %v", c.v, got)
		}
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "0.txt"), []byte("낮음\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "1.txt"), []byte("on\n---\noff\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "note.md"), []byte("읽지 않는다"), 0o644)
	s, err := Load(dir, "ko")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Pictures) != 2 || s.Pictures[0].At != 0 || s.Pictures[1].At != 1 {
		t.Fatalf("pictures %+v", s.Pictures)
	}
	shot := func(value, time float64) []string {
		r := feed.Validate(map[string]any{"spec": 1.0, "values": []any{map[string]any{"value": value}}})
		g := render.Render(s, render.Input{Values: feed.Compute(r.Feed), State: "live", Cols: 10, Rows: 3, Time: time, Quality: "high", Color: true})
		return plain(render.Lines(g))
	}
	if got := shot(0, 0)[1]; got != "   낮음   " {
		t.Errorf("한글: %q", got)
	}
	if got := strings.TrimSpace(shot(1, 0)[1]); got != "on" {
		t.Errorf("첫 장: %q", got)
	}
	if got := strings.TrimSpace(shot(1, 0.5)[1]); got != "off" {
		t.Errorf("둘째 장: %q", got)
	}
}

func TestMakeSamples(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "art")
	if _, err := MakeSamples(dir, "ko"); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 12 {
		t.Fatalf("파일 %d 개", len(entries))
	}
	s, err := Load(dir, "ko")
	if err != nil {
		t.Fatal(err)
	}
	many := false
	for _, p := range s.Pictures {
		many = many || len(p.Frames) > 1
	}
	if len(s.Pictures) != 11 || !many {
		t.Errorf("그림 %d, 여러 장 %v", len(s.Pictures), many)
	}
	if _, err := MakeSamples(dir, "ko"); err == nil {
		t.Error("비어 있지 않은 폴더에 만들었다")
	}
	// JavaScript 판의 견본과 같다
	want, err := os.ReadFile("../../testdata/sample-60.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "60.txt")); string(got) != string(want) {
		t.Error("견본 60.txt 가 JavaScript 판과 다르다")
	}
	got, _ := os.ReadFile(filepath.Join(dir, "README.txt"))
	if !strings.Contains(string(got), "tidetop --art <this folder>") {
		t.Error("README.txt 에 영문 안내가 없다")
	}
}
