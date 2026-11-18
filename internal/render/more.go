package render

import (
	"math"
	"strings"

	"github.com/tidepark/tidetop/internal/glitch"
	"github.com/tidepark/tidetop/internal/noise"
)

// IsWide 는 한글처럼 두 칸을 차지하는 글자인가이다.
func IsWide(r rune) bool {
	return (r >= 0x1100 && r <= 0x115f) || (r >= 0x2e80 && r <= 0xa4cf) || (r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) || (r >= 0xfe30 && r <= 0xfe4f) || (r >= 0xff00 && r <= 0xff60) || (r >= 0xffe0 && r <= 0xffe6)
}

// Width 는 글이 차지하는 칸 수다.
func Width(text string) int {
	n := 0
	for _, r := range text {
		n++
		if IsWide(r) {
			n++
		}
	}
	return n
}

// Stamp 는 알림글 몇 줄을 판에 얹는다. at 이 "bottom-right" 이면 오른쪽 아래, 아니면 왼쪽 위다.
// 한글처럼 두 칸을 차지하는 글자는 다음 칸을 Covered 로 두어 화면 쓰기가 건너뛰게 한다.
func Stamp(g *Grid, lines []string, high, color bool, at string) {
	fg, bg := None, None
	if high {
		fg, bg = 0xdcdcdc, 0x101014
	} else if color {
		fg = Basic + 97
	}
	inner := 0
	for _, l := range lines {
		inner = max(inner, Width(l))
	}
	corner := at == "bottom-right"
	left, top := 1, 1
	if corner {
		left = max(0, g.Cols-(inner+2)-1)
		top = max(0, g.Rows-len(lines)-1)
	}
	for row, line := range lines {
		if top+row >= g.Rows {
			return
		}
		var cells []uint16
		for _, r := range " " + line + strings.Repeat(" ", inner-Width(line)) + " " {
			cells = append(cells, uint16(r))
			if IsWide(r) {
				cells = append(cells, Covered)
			}
		}
		if limit := g.Cols - left - 1; len(cells) > limit {
			cells = cells[:max(0, limit)]
		}
		for k, code := range cells {
			// 줄 끝에서 두 칸 글자가 잘리면 빈칸으로 둔다
			last := k == g.Cols-left-2 && IsWide(rune(code))
			i := (top+row)*g.Cols + left + k
			if last {
				code = space
			}
			g.Ch[i], g.Fg[i], g.Bg[i] = code, fg, bg
		}
	}
}

// TileMin 은 조각의 가장 작은 크기(칸)다. 이보다 작아지면 나누지 않는다.
var TileMin = struct{ Cols, Rows int }{8, 3}

// Layout 은 조각 나누기의 배치다.
type Layout struct{ Across, Down, TW, TH int }

// TileLayout: 값 n 개를 조각 n 개에 놓는다. 조각이 점으로 거의 정사각(칸으로 2:1)에 가깝도록 고른다.
func TileLayout(n, cols, rows int) (Layout, bool) {
	var best Layout
	bestScore, found := 0, false
	for across := 1; across <= n; across++ {
		down := (n + across - 1) / across
		tw := int(math.Floor(float64(cols-(across-1)) / float64(across)))
		th := int(math.Floor(float64(rows-(down-1)) / float64(down)))
		if tw < TileMin.Cols || th < TileMin.Rows {
			continue
		}
		score := min(tw, th*2)
		if !found || score > bestScore {
			best, bestScore, found = Layout{across, down, tw, th}, score, true
		}
	}
	return best, found
}

// RenderTiles 는 값마다 조각 하나에 같은 장면을 그린다(코어마다 보기). 조각 사이는 한 칸씩 띄운다.
func RenderTiles(s Scene, in Input) *Grid {
	lay, ok := TileLayout(len(in.Values), in.Cols, in.Rows)
	if !ok {
		return Render(s, in)
	}
	g := NewGrid(in.Cols, in.Rows)
	g.High = in.Quality != "low"
	ox := int(math.Floor(float64(in.Cols-(lay.Across*(lay.TW+1)-1)) / 2))
	oy := int(math.Floor(float64(in.Rows-(lay.Down*(lay.TH+1)-1)) / 2))
	for k, v := range in.Values {
		one := in
		one.Values, one.Cols, one.Rows = in.Values[k:k+1], lay.TW, lay.TH
		_ = v
		tile := Render(s, one)
		x0 := ox + (k%lay.Across)*(lay.TW+1)
		y0 := oy + (k/lay.Across)*(lay.TH+1)
		for y := 0; y < lay.TH; y++ {
			from, to := y*lay.TW, (y0+y)*in.Cols+x0
			copy(g.Ch[to:to+lay.TW], tile.Ch[from:from+lay.TW])
			copy(g.Fg[to:to+lay.TW], tile.Fg[from:from+lay.TW])
			copy(g.Bg[to:to+lay.TW], tile.Bg[from:from+lay.TW])
		}
	}
	return g
}

// Crop 은 여러 창 이어 붙이기의 한 조각이다. 장면을 가로 Across × 세로 Down 의 큰 그림으로 셈하고,
// Col, Row 자리의 조각만 그린다. 큰 그림의 크기는 이 창의 크기 × 조각 수다.
// 주의 줄은 「이 컴퓨터가 빠듯하다」는 표시라서 큰 그림에 흩지 않고 줄의 자리를 줄여 이 조각 안에 모두 넣는다.
type Crop struct {
	Scene                  Scene
	Across, Down, Col, Row int
}

func (c Crop) Name() string    { return c.Scene.Name() }
func (c Crop) Local() string   { return c.Scene.Local() }
func (c Crop) Skins() []string { return c.Scene.Skins() }

type cropCtx struct {
	inner  Ctx
	ox, oy int
	box    *Box
	lines  []glitch.Line
}

func (c Crop) Frame(in Input) Ctx {
	big := in
	big.W, big.H = in.W*c.Across, in.H*c.Down
	big.Cols, big.Rows = in.Cols*c.Across, in.Rows*c.Down
	big.GlitchSeed = c.Row*c.Across + c.Col + 1
	inner := c.Scene.Frame(big)
	x := &cropCtx{inner: inner, ox: c.Col * in.W, oy: c.Row * in.H}
	if b := inner.Box(); b != nil {
		x.box = &Box{b.X0 - float64(x.ox), b.X1 - float64(x.ox), b.Y0 - float64(x.oy), b.Y1 - float64(x.oy)}
	}
	for _, l := range inner.Glitch() {
		if l.Across {
			l.Pos, l.Start, l.End = math.Floor(l.Pos/float64(c.Down)), l.Start/float64(c.Across), l.End/float64(c.Across)
		} else {
			l.Pos, l.Start, l.End = math.Floor(l.Pos/float64(c.Across)), l.Start/float64(c.Down), l.End/float64(c.Down)
		}
		x.lines = append(x.lines, l)
	}
	return x
}

func (c *cropCtx) Pixel(x, y int) (noise.RGB, bool) { return c.inner.Pixel(x+c.ox, y+c.oy) }
func (c *cropCtx) Glyph(cx, cy int) (rune, string, bool) {
	return c.inner.Glyph(cx+c.ox, cy+c.oy/2)
}
func (c *cropCtx) Dots(add Dot) {
	c.inner.Dots(func(x, y float64, col noise.RGB, w float64, onTop bool) {
		add(x-float64(c.ox), y-float64(c.oy), col, w, onTop)
	})
}
func (c *cropCtx) Box() *Box             { return c.box }
func (c *cropCtx) Glitch() []glitch.Line { return c.lines }
