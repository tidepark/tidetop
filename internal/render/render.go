// Package render 는 장면을 터미널의 칸으로 바꾼다.
//
// 장면이 그리는 자리는 「점」이다. 점 하나는 글자 한 칸의 위쪽 반이나 아래쪽 반이다.
// 가로 W = 칸 수, 세로 H = 줄 수 × 2 이고 점은 거의 정사각이다. y 는 위에서 아래로 센다.
//
// high: 칸마다 반 칸 블록(▀ ▄)에 글자색과 바탕색을 주어 점 둘을 그린다(24비트 색).
//
//	그 위에 점자(칸마다 가로 2 × 세로 4)로 잔 점을 얹는다.
//
// low : 칸마다 ASCII 글자 하나. 색은 기본 여덟 가지이고, 색을 못 쓰면 단색이다.
//
// 화면은 칸마다 글자, 글자색, 바탕색을 담은 판(Grid)이다. 색은 정수 하나다:
// -1 정하지 않음, 0 ~ 0xffffff 24비트 색, Basic + 번호 는 기본 색.
package render

import (
	"fmt"
	"image"
	"math"
	"strconv"
	"strings"

	"github.com/tidepark/tidetop/internal/feed"
	"github.com/tidepark/tidetop/internal/glitch"
	"github.com/tidepark/tidetop/internal/noise"
)

const (
	esc   = "\x1b["
	reset = esc + "0m"
	// None 은 정하지 않은 색이다.
	None int32 = -1
	// Basic 에 기본 색 번호를 더한 것이 low 의 색이다.
	Basic int32 = 0x1000000
	space       = 32
	upper       = 0x2580 // ▀
	lower       = 0x2584 // ▄
	// Covered 는 앞의 두 칸 글자가 덮은 칸이다.
	Covered = 0
)

// Grid 는 화면 한 장이다.
type Grid struct {
	Cols, Rows int
	Ch         []uint16
	Fg, Bg     []int32
	High       bool
}

// spare 는 Recycle 로 돌려받은 판 하나다. 다음 NewGrid 가 크기가 같으면 비워서 다시 쓴다
// (장마다 판을 새로 잡지 않는다. 그리기는 한 고루틴에서만 한다).
var spare *Grid

// Recycle 은 다 쓴 판을 돌려준다. 돌려준 판은 더 읽거나 쓰지 않는다.
func Recycle(g *Grid) { spare = g }

// NewGrid 는 빈 판을 만든다.
func NewGrid(cols, rows int) *Grid {
	g := spare
	if g != nil && g.Cols == cols && g.Rows == rows {
		spare = nil
		g.High = false
	} else {
		n := cols * rows
		g = &Grid{Cols: cols, Rows: rows, Ch: make([]uint16, n), Fg: make([]int32, n), Bg: make([]int32, n)}
	}
	for i := range g.Ch {
		g.Ch[i] = space
		g.Fg[i] = None
		g.Bg[i] = None
	}
	return g
}

// Box 는 장면이 그리는 네모(점 단위)다. 이 밖은 비어 있다.
type Box struct{ X0, X1, Y0, Y1 float64 }

// Dot 은 점자 점 하나를 얹는다. w 는 밝기(한 칸에서 가장 밝은 점의 색을 쓴다), onTop 이면 칠한 칸 위에도 얹는다.
type Dot func(x, y float64, c noise.RGB, w float64, onTop bool)

// Input 은 장면에 주는 것이다.
type Input struct {
	Values     []feed.Computed
	State      string // "live" 또는 "stale"(소식이 끊김)
	Cols, Rows int
	Time       float64
	Quality    string
	Color      bool
	Skin       string
	W, H       int // 점 단위의 크기. 그리기가 채운다
	GlitchSeed int
	// View 는 이 창이 그리는 범위(점 단위)다. 이어 붙인 창의 조각일 때만 Crop 이 정하고, 비어 있으면 전체다.
	// 장면은 이 밖을 셈하지 않아도 된다(읽지 않는다).
	View image.Rectangle
}

// Ctx 는 장면이 한 장을 그리려고 셈해 둔 것이다.
type Ctx interface {
	Pixel(x, y int) (noise.RGB, bool)
	Glyph(cx, cy int) (ch rune, tone string, ok bool)
	Dots(add Dot)
	Box() *Box
	Glitch() []glitch.Line
}

// Scene 은 (값, 상태, 칸 수, 줄 수, 시각) → 화면인 순수한 함수다.
type Scene interface {
	Name() string
	Local() string
	Skins() []string
	Frame(in Input) Ctx
}

// 소식이 끊겼을 때 덮는 잿빛. 컴패니언의 rgba(90, 90, 90, 0.45) 와 같다.
func stale(c noise.RGB) noise.RGB {
	return noise.RGB{c[0]*0.55 + 40, c[1]*0.55 + 40, c[2]*0.55 + 40}
}

// 색은 4 단계씩 묶는다. 이웃한 칸이나 앞 장과 색이 같아지면 다시 적지 않아 쓰는 양이 준다.
func q(x float64) int32 {
	if x < 0 {
		return 0
	}
	// min(255, Round(x/4)*4) 와 같다. 점마다 세 번 부르므로 일반 반올림과 Min 을 거치지 않는다
	y := x / 4
	r := math.Floor(y)
	if y-r >= 0.5 {
		r++
	}
	if r >= 64 {
		return 255
	}
	return int32(r) * 4
}

// Pack 은 색을 정수 하나로 묶는다.
func Pack(c noise.RGB) int32 { return q(c[0])<<16 | q(c[1])<<8 | q(c[2]) }

// FgCode, BgCode 는 색을 터미널 부호로 바꾼다.
func FgCode(c int32) string {
	if c >= Basic {
		return fmt.Sprintf("%s%dm", esc, c-Basic)
	}
	return fmt.Sprintf("%s38;2;%d;%d;%dm", esc, c>>16, (c>>8)&255, c&255)
}

// AppendSGR 는 글자색(setFg 일 때)과 바탕색(setBg 일 때)을 부호 하나로 b 에 붙인다.
// 둘 다 바뀌면 ESC[38;2;r;g;b;48;2;r;g;bm 처럼 묶는다. 장마다 부르므로 문자열을 새로 만들지 않는다.
func AppendSGR(b []byte, fg, bg int32, setFg, setBg bool) []byte {
	b = append(b, esc...)
	if setFg {
		if fg >= Basic {
			b = strconv.AppendInt(b, int64(fg-Basic), 10)
		} else {
			b = appendRGB(append(b, "38;2;"...), fg)
		}
	}
	if setFg && setBg {
		b = append(b, ';')
	}
	if setBg {
		if bg == None {
			b = append(b, "49"...)
		} else {
			b = appendRGB(append(b, "48;2;"...), bg)
		}
	}
	return append(b, 'm')
}

func appendRGB(b []byte, c int32) []byte {
	b = strconv.AppendInt(b, int64(c>>16), 10)
	b = append(b, ';')
	b = strconv.AppendInt(b, int64((c>>8)&255), 10)
	b = append(b, ';')
	return strconv.AppendInt(b, int64(c&255), 10)
}

func BgCode(c int32) string {
	if c == None {
		return esc + "49m"
	}
	return fmt.Sprintf("%s48;2;%d;%d;%dm", esc, c>>16, (c>>8)&255, c&255)
}

type cellBox struct{ x0, x1, y0, y1 int }

func boxOf(ctx Ctx, cols, rows int) cellBox {
	b := ctx.Box()
	if b == nil {
		return cellBox{0, cols - 1, 0, rows - 1}
	}
	return cellBox{
		x0: int(math.Max(0, b.X0)),
		x1: int(math.Min(float64(cols-1), b.X1)),
		y0: int(math.Max(0, math.Floor(b.Y0/2))),
		y1: int(math.Min(float64(rows-1), math.Floor(b.Y1/2))),
	}
}

// 점자: 칸 안의 점 (가로 0~1, 세로 0~3) → 비트
var braille = [4][2]uint8{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}

type dotLayer struct {
	mask   []uint8
	color  []noise.RGB
	weight []float32 // JavaScript 의 Float32Array 와 같게 32비트로 둔다
	over   []uint8
}

// dotBuf 는 빛 점 판이다. 장마다 새로 잡지 않고 비워서 다시 쓴다(그리기는 한 고루틴에서만 한다).
var dotBuf dotLayer

func collectDots(ctx Ctx, cols, rows int) dotLayer {
	n := cols * rows
	d := &dotBuf
	if cap(d.mask) < n {
		*d = dotLayer{make([]uint8, n), make([]noise.RGB, n), make([]float32, n), make([]uint8, n)}
	} else {
		d.mask, d.color, d.weight, d.over = d.mask[:n], d.color[:n], d.weight[:n], d.over[:n]
		clear(d.mask)
		clear(d.color)
		clear(d.weight)
		clear(d.over)
	}
	ctx.Dots(func(x, y float64, c noise.RGB, w float64, onTop bool) {
		bx := int(math.Floor(x * 2))
		by := int(math.Floor(y * 2))
		if math.IsNaN(x*2) || math.IsNaN(y*2) || x*2 < 0 || y*2 < 0 || bx >= cols*2 || by >= rows*4 {
			return
		}
		i := (by>>2)*cols + (bx >> 1)
		d.mask[i] |= braille[by&3][bx&1]
		if onTop {
			d.over[i] = 1
		}
		// 한 칸에 색은 하나뿐이다. 가장 밝은 점의 색을 쓴다.
		if w >= float64(d.weight[i]) {
			d.weight[i] = float32(w)
			d.color[i] = c
		}
	})
	return *d
}

func gridHigh(ctx Ctx, in Input) *Grid {
	cols, rows := in.Cols, in.Rows
	dots := collectDots(ctx, cols, rows)
	tint := func(c noise.RGB) noise.RGB { return c }
	if in.State == "stale" {
		tint = stale
	}
	g := NewGrid(cols, rows)
	g.High = true
	box := boxOf(ctx, cols, rows)
	lines := ctx.Glitch()
	for cy := 0; cy < rows; cy++ {
		inRows := cy >= box.y0 && cy <= box.y1
		for cx := 0; cx < cols; cx++ {
			i := cy*cols + cx
			inside := inRows && cx >= box.x0 && cx <= box.x1
			var top, bot noise.RGB
			hasTop, hasBot := false, false
			if inside {
				top, hasTop = ctx.Pixel(cx, cy*2)
				bot, hasBot = ctx.Pixel(cx, cy*2+1)
			}
			// 주의의 줄은 장면 위에 얹는다
			if lines != nil {
				if l, ok := glitch.At(lines, float64(cx), float64(cy*2)); ok {
					top, hasTop = l.Color, true
				}
				if l, ok := glitch.At(lines, float64(cx), float64(cy*2+1)); ok {
					bot, hasBot = l.Color, true
				}
			}
			switch {
			case dots.mask[i] != 0 && ((!hasTop && !hasBot) || (hasTop && hasBot && dots.over[i] != 0)):
				g.Ch[i] = uint16(0x2800 + int(dots.mask[i]))
				g.Fg[i] = Pack(tint(dots.color[i]))
				if hasTop && hasBot {
					g.Bg[i] = Pack(tint(noise.RGB{(top[0] + bot[0]) / 2, (top[1] + bot[1]) / 2, (top[2] + bot[2]) / 2}))
				}
			case hasTop && hasBot:
				t, b := Pack(tint(top)), Pack(tint(bot))
				g.Bg[i] = b
				if t != b {
					g.Ch[i] = upper
					g.Fg[i] = t
				}
			case hasTop:
				g.Ch[i] = upper
				g.Fg[i] = Pack(tint(top))
			case hasBot:
				g.Ch[i] = lower
				g.Fg[i] = Pack(tint(bot))
			}
		}
	}
	return g
}

// Tones 는 low 의 빛깔 이름 → 기본 색 번호다.
var Tones = map[string]int32{"dim": 90, "red": 31, "green": 32, "magenta": 35, "yellow": 33, "bright": 97, "blue": 34, "cyan": 36, "grey": 37}

func tone(name string) int32 {
	if n, ok := Tones[name]; ok {
		return n
	}
	return 39
}

func gridLow(ctx Ctx, in Input) *Grid {
	cols, rows := in.Cols, in.Rows
	// 잔 점은 칸마다 하나의 글자로 줄인다
	specks := map[int]rune{}
	ctx.Dots(func(x, y float64, _ noise.RGB, w float64, _ bool) {
		cx := int(math.Floor(x))
		cy := int(math.Floor(y / 2))
		if math.Floor(x) < 0 || math.Floor(y/2) < 0 || cx >= cols || cy >= rows || w < 0.25 {
			return
		}
		ch := '.'
		if y/2-math.Floor(y/2) < 0.5 {
			ch = '\''
		}
		specks[cy*cols+cx] = ch
	})
	g := NewGrid(cols, rows)
	box := boxOf(ctx, cols, rows)
	lines := ctx.Glitch()
	for cy := 0; cy < rows; cy++ {
		inRows := cy >= box.y0 && cy <= box.y1
		for cx := 0; cx < cols; cx++ {
			i := cy*cols + cx
			var ch rune
			var tn string
			ok := false
			if inRows && cx >= box.x0 && cx <= box.x1 {
				ch, tn, ok = ctx.Glyph(cx, cy)
			}
			if lines != nil {
				l, hit := glitch.At(lines, float64(cx), float64(cy*2))
				if !hit {
					l, hit = glitch.At(lines, float64(cx), float64(cy*2+1))
				}
				if hit {
					ch, tn, ok = '|', l.Tone, true
					if l.Short {
						ch = '+'
					} else if l.Across {
						ch = '='
					}
				}
			}
			if !ok {
				if sp, has := specks[i]; has {
					ch, tn, ok = sp, "yellow", true
				}
			}
			if !ok {
				continue
			}
			g.Ch[i] = uint16(ch)
			if in.Color {
				if in.State == "stale" {
					tn = "dim"
				}
				g.Fg[i] = Basic + tone(tn)
			}
		}
	}
	return g
}

// Texter 는 글자를 칸에 바로 넣는 장면(사용자의 글자 그림)의 맥락이다.
type Texter interface {
	Cells(g *Grid, color bool)
}

// Render 는 장면 한 장을 판으로 그린다.
func Render(s Scene, in Input) *Grid {
	in.W, in.H = in.Cols, in.Rows*2
	ctx := s.Frame(in)
	if t, ok := ctx.(Texter); ok {
		return gridText(ctx, t, in)
	}
	if in.Quality == "low" {
		return gridLow(ctx, in)
	}
	return gridHigh(ctx, in)
}

// 글자를 칸에 바로 넣는다. 주의의 줄은 그 위에 얹는다.
func gridText(ctx Ctx, t Texter, in Input) *Grid {
	cols, rows := in.Cols, in.Rows
	high := in.Quality != "low"
	g := NewGrid(cols, rows)
	g.High = high
	t.Cells(g, in.Color)
	lines := ctx.Glitch()
	if lines == nil {
		return g
	}
	for cy := 0; cy < rows; cy++ {
		for cx := 0; cx < cols; cx++ {
			top, hasTop := glitch.At(lines, float64(cx), float64(cy*2))
			bot, hasBot := glitch.At(lines, float64(cx), float64(cy*2+1))
			if !hasTop && !hasBot {
				continue
			}
			i := cy*cols + cx
			line := top
			if !hasTop {
				line = bot
			}
			if !high {
				ch := '|'
				if line.Short {
					ch = '+'
				} else if line.Across {
					ch = '='
				}
				g.Ch[i] = uint16(ch)
				g.Fg[i] = None
				if in.Color {
					g.Fg[i] = Basic + tone(line.Tone)
				}
				continue
			}
			g.Bg[i] = None
			if hasTop && hasBot {
				g.Ch[i] = space
				g.Bg[i] = Pack(top.Color)
				if top.Color != bot.Color {
					g.Ch[i] = upper
					g.Fg[i] = Pack(top.Color)
					g.Bg[i] = Pack(bot.Color)
				}
			} else {
				g.Ch[i] = lower
				if hasTop {
					g.Ch[i] = upper
				}
				g.Fg[i] = Pack(line.Color)
			}
		}
	}
	// 줄이 두 칸 글자의 한쪽만 덮었으면 남은 쪽도 빈칸으로 둔다
	for i := range g.Ch {
		if g.Ch[i] == Covered && (i == 0 || !IsWide(rune(g.Ch[i-1]))) {
			g.Ch[i] = space
		} else if IsWide(rune(g.Ch[i])) && (i+1 >= len(g.Ch) || g.Ch[i+1] != Covered) {
			g.Ch[i] = space
		}
	}
	return g
}

// Lines 는 판을 줄마다의 글로 바꾼다. 바뀐 색만 적는다.
func Lines(g *Grid) []string {
	out := make([]string, 0, g.Rows)
	for cy := 0; cy < g.Rows; cy++ {
		var b strings.Builder
		fg, bg := None, None
		for i := cy * g.Cols; i < (cy+1)*g.Cols; i++ {
			if g.Fg[i] != None && g.Fg[i] != fg {
				fg = g.Fg[i]
				b.WriteString(FgCode(fg))
			}
			if g.Bg[i] != bg {
				bg = g.Bg[i]
				b.WriteString(BgCode(bg))
			}
			if g.Ch[i] != Covered {
				b.WriteRune(rune(g.Ch[i]))
			}
		}
		if g.High || fg != None {
			b.WriteString(reset)
		}
		out = append(out, b.String())
	}
	return out
}
