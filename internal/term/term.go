// Package term 은 터미널 다루기다: 무엇을 그릴 수 있는지 알아보고, 대체 화면을 열고 닫고, 바뀐 칸만 다시 쓴다.
package term

import (
	"io"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/tidepark/tidetop/internal/render"
	xterm "golang.org/x/term"
)

const esc = "\x1b["

// Ability 는 이 터미널로 그릴 수 있는 것이다.
type Ability struct {
	Quality string // "high" 또는 "low"
	Color   bool
}

var utf8Locale = regexp.MustCompile(`(?i)utf-?8`)
var trueColorEnv = regexp.MustCompile(`(?i)truecolor|24bit`)

// IsTerminal 은 f 가 터미널인가이다.
func IsTerminal(f *os.File) bool { return xterm.IsTerminal(int(f.Fd())) }

// Detect: 24비트 색과 UTF-8 이 되면 high, 아니면 low. 색을 못 쓰거나 NO_COLOR 이면 단색이다.
func Detect(out *os.File) Ability {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" || !IsTerminal(out) {
		return Ability{"low", false}
	}
	enableVirtual(out)
	depth := 4
	termName := os.Getenv("TERM")
	switch {
	case trueColorEnv.MatchString(os.Getenv("COLORTERM")) || os.Getenv("WT_SESSION") != "" || runtime.GOOS == "windows":
		depth = 24
	case strings.Contains(termName, "256"):
		depth = 8
	}
	truecolor := depth >= 24
	// 로캘을 적지 않은 터미널은 24비트 색이 되면 UTF-8 도 된다고 본다
	locale := firstSet("LC_ALL", "LC_CTYPE", "LANG")
	utf8 := truecolor
	if locale != "" {
		utf8 = utf8Locale.MatchString(locale)
	}
	q := "low"
	if truecolor && utf8 {
		q = "high"
	}
	return Ability{q, depth >= 4}
}

func firstSet(names ...string) string {
	for _, n := range names {
		if v := os.Getenv(n); v != "" {
			return v
		}
	}
	return ""
}

// Size 는 칸 수와 줄 수다. 터미널이 아니면 COLUMNS, LINES 를 본다.
func Size(out *os.File) (int, int) {
	cols, rows, err := xterm.GetSize(int(out.Fd()))
	if err != nil || cols <= 0 || rows <= 0 {
		cols, rows = envInt("COLUMNS", 80), envInt("LINES", 24)
	}
	return max(8, cols), max(4, rows)
}

func envInt(name string, def int) int {
	if n, err := strconv.Atoi(os.Getenv(name)); err == nil && n > 0 {
		return n
	}
	return def
}

// Screen 은 대체 화면이다. 앞 장과 달라진 칸만 다시 쓴다.
type Screen struct {
	out      io.Writer
	previous *render.Grid
	open     bool
}

func NewScreen(out io.Writer) *Screen { return &Screen{out: out} }

// Open: 대체 화면, 커서 숨김, 줄 넘김 끔
func (s *Screen) Open() {
	io.WriteString(s.out, esc+"?1049h"+esc+"?25l"+esc+"?7l"+esc+"2J")
	s.open = true
}

func (s *Screen) Close() {
	if !s.open {
		return
	}
	s.open = false
	io.WriteString(s.out, esc+"0m"+esc+"?7h"+esc+"?25h"+esc+"?1049l")
}

func (s *Screen) Clear() {
	s.previous = nil
	io.WriteString(s.out, esc+"0m"+esc+"2J")
}

// Draw 는 앞 장과 달라진 칸만 다시 쓴다. 쓴 것이 있으면 true 다.
func (s *Screen) Draw(g *render.Grid) bool {
	old := s.previous
	if old != nil && (old.Cols != g.Cols || old.Rows != g.Rows) {
		old = nil
	}
	differs := func(i int) bool {
		return old == nil || g.Ch[i] != old.Ch[i] || g.Fg[i] != old.Fg[i] || g.Bg[i] != old.Bg[i]
	}
	var b strings.Builder
	nowFg, nowBg := render.None, render.None
	for y := 0; y < g.Rows; y++ {
		at := y * g.Cols
		for x := 0; x < g.Cols; x++ {
			if !differs(at + x) {
				continue
			}
			// 달라진 칸이 가까이 이어지면 한 번에 쓴다. 떨어져 있으면 커서를 옮긴다.
			end := x
			for k := x + 1; k < g.Cols && k-end <= 4; k++ {
				if differs(at + k) {
					end = k
				}
			}
			b.WriteString(esc + strconv.Itoa(y+1) + ";" + strconv.Itoa(x+1) + "H")
			for i := at + x; i <= at+end; i++ {
				if g.Fg[i] != render.None && g.Fg[i] != nowFg {
					nowFg = g.Fg[i]
					b.WriteString(render.FgCode(nowFg))
				}
				if g.Bg[i] != nowBg {
					nowBg = g.Bg[i]
					b.WriteString(render.BgCode(nowBg))
				}
				// 0 은 앞의 두 칸 글자가 덮은 칸이다. 커서는 이미 지나갔다.
				if g.Ch[i] != render.Covered {
					b.WriteRune(rune(g.Ch[i]))
				}
			}
			x = end
		}
	}
	s.previous = g
	if b.Len() == 0 {
		return false
	}
	// 한 장을 한꺼번에 보이게 한다(모르는 터미널은 이 부호를 무시한다).
	// 커서 숨김을 장마다 다시 보내고, 다 쓴 뒤에는 오른쪽 아래 구석에 세워 둔다.
	io.WriteString(s.out, esc+"?2026h"+esc+"?25l"+b.String()+esc+"0m"+esc+strconv.Itoa(g.Rows)+";"+strconv.Itoa(g.Cols)+"H"+esc+"?2026l")
	return true
}

// Raw 는 키를 한 글자씩 받도록 입력을 원시 모드로 둔다. 돌려받은 함수로 되돌린다.
func Raw(in *os.File) (restore func(), err error) {
	if !IsTerminal(in) {
		return func() {}, nil
	}
	st, err := xterm.MakeRaw(int(in.Fd()))
	if err != nil {
		return func() {}, err
	}
	return func() { xterm.Restore(int(in.Fd()), st) }, nil
}
