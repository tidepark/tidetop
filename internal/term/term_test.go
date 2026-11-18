package term

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tidepark/tidetop/internal/render"
)

// 바뀐 칸만 써도 통째로 쓴 것과 같은 화면이 되고, 같은 장을 다시 그리면 쓸 것이 없다
func TestDrawOnlyChanges(t *testing.T) {
	var out bytes.Buffer
	s := NewScreen(&out)
	g := render.NewGrid(10, 2)
	g.Ch[3], g.Fg[3] = 'a', 0x102030
	if !s.Draw(g) {
		t.Fatal("처음에는 써야 한다")
	}
	if !strings.Contains(out.String(), "\x1b[38;2;16;32;48m") {
		t.Errorf("색 부호가 없다: %q", out.String())
	}
	out.Reset()
	if s.Draw(g) {
		t.Errorf("같은 장인데 썼다: %q", out.String())
	}
	h := render.NewGrid(10, 2)
	copy(h.Ch, g.Ch)
	copy(h.Fg, g.Fg)
	copy(h.Bg, g.Bg)
	h.Ch[15] = '한'
	h.Ch[16] = render.Covered
	out.Reset()
	s.Draw(h)
	if !strings.Contains(out.String(), "\x1b[2;6H한") {
		t.Errorf("바뀐 칸만: %q", out.String())
	}
}

// 글자색과 바탕색이 함께 바뀌면 부호 하나로 쓰고, 묻는 글(Ask)은 장 끝에 붙여 한 번에 보낸다
func TestDrawOneWrite(t *testing.T) {
	var out bytes.Buffer
	s := NewScreen(&out)
	s.Asking = true
	g := render.NewGrid(4, 1)
	g.Ch[1], g.Fg[1], g.Bg[1] = 0x2580, 0x102030, 0x405060
	g.Ch[2], g.Fg[2], g.Bg[2] = 'x', render.Basic+31, render.None
	if !s.Draw(g) {
		t.Fatal("처음에는 써야 한다")
	}
	got := out.String()
	for _, want := range []string{"\x1b[38;2;16;32;48;48;2;64;80;96m▀", "\x1b[31;49mx"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q 가 없다: %q", want, got)
		}
	}
	if !strings.HasSuffix(got, "\x1b[?2026l"+Ask) {
		t.Errorf("묻는 글이 장 끝에 붙지 않았다: %q", got)
	}
}
