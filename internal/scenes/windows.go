package scenes

// 불 켜진 창. 값 여럿을 읽는다. 창 하나가 값 하나다.
//
// 맺음
//   - 값의 순서대로 왼쪽 위부터 창을 놓는다. 창의 수와 화면의 모양을 보고 줄과 칸을 나눈다.
//   - 채움(fill)이 창의 밝기다. 0 이면 유리만 어둑하게 보이고, 찰수록 따뜻한 빛이 환해진다.
//   - 범위를 넘으면(over) 빛이 희어지고 창 둘레에 빛 점이 반짝인다.
//   - 쉼(idle)과 끝남(done)은 달빛만 비치는 창, 고장(fault)은 검붉은 창이다. 둘 다 멈춰 있다.
//
// 깜빡이는 빠르기는 창마다 정해져 있다. 값은 밝기와 깜빡임의 크기만 바꾼다.

import (
	"math"

	"github.com/tidepark/tidetop/internal/render"
)

var windowLight = []stop{
	{0, rgb{20, 24, 38}},
	{0.12, rgb{58, 42, 34}},
	{0.35, rgb{150, 96, 44}},
	{0.65, rgb{232, 164, 72}},
	{0.88, rgb{255, 212, 124}},
	{1, rgb{255, 234, 176}},
}
var moonGlass = rgb{34, 44, 70}
var brokenGlass = rgb{112, 34, 30}

type Windows struct{}

func (Windows) Name() string    { return "windows" }
func (Windows) Local() string   { return "cores" }
func (Windows) Skins() []string { return nil }

type window struct {
	i              float64
	mood           string
	x0, y0, x1, y1 float64
	lit, over      float64
}

type windowsCtx struct {
	noGlitch
	noBox
	t              float64
	windows        []window
	gc, gr, cw, ch float64
	n              int
}

func (Windows) Frame(in render.Input) render.Ctx {
	W, H := float64(in.W), float64(in.H)
	n := len(in.Values)
	// 창이 세로로 조금 긴 네모에 가깝고 빈자리가 적게 나오는 나눔을 고른다
	gc, gr, best := 0.0, 0.0, 0.0
	for c := 1; c <= n; c++ {
		r := math.Ceil(float64(n) / float64(c))
		score := math.Abs(math.Log(W/float64(c)/(H/r)/0.9)) + (float64(c)*r-float64(n))*0.12
		if gc == 0 || score < best {
			gc, gr, best = float64(c), r, score
		}
	}
	cw, ch := W/gc, H/gr
	ctx := &windowsCtx{t: in.Time, gc: gc, gr: gr, cw: cw, ch: ch, n: n}
	for i, v := range in.Values {
		row := math.Floor(float64(i) / gc)
		// 덜 찬 마지막 줄은 가운데로 모은다
		shift := 0.0
		if row == gr-1 {
			shift = (gc - (float64(n) - row*gc)) / 2
		}
		col := mod(float64(i), gc) + shift
		x0 := round(col*cw + cw*0.16)
		y0 := round(row*ch + ch*0.14)
		w := window{
			i: float64(i), mood: moodOf(v.State), x0: x0, y0: y0,
			x1: math.Max(x0, round((col+1)*cw-cw*0.16)-1),
			y1: math.Max(y0, round((row+1)*ch-ch*0.14)-1),
		}
		if w.mood == "live" {
			w.lit = level(v)
			w.over = or0(v.Over)
		}
		ctx.windows = append(ctx.windows, w)
	}
	return ctx
}

func (c *windowsCtx) find(x, y float64) *window {
	row := math.Floor(y / c.ch)
	if row < 0 || row >= c.gr {
		return nil
	}
	shift := 0.0
	if row == c.gr-1 {
		shift = (c.gc - (float64(c.n) - row*c.gc)) / 2
	}
	k := math.Floor(x/c.cw - shift)
	idx := row*c.gc + k
	if idx < 0 || idx >= float64(len(c.windows)) || idx != math.Floor(idx) {
		return nil
	}
	w := &c.windows[int(idx)]
	if x >= w.x0 && x <= w.x1 && y >= w.y0 && y <= w.y1 && k >= 0 {
		return w
	}
	return nil
}

func (c *windowsCtx) Pixel(xi, yi int) (rgb, bool) {
	x, y := float64(xi), float64(yi)
	w := c.find(x, y)
	if w == nil {
		return rgb{}, false
	}
	ww := w.x1 - w.x0 + 1
	wh := w.y1 - w.y0 + 1
	px := x - w.x0
	py := y - w.y0
	// 창살: 창이 넉넉히 클 때만
	bar := ww >= 7 && wh >= 7 && (px == math.Floor(ww/2) || py == math.Floor(wh*0.42))
	shade := 1.0
	if bar {
		shade = 0.4
	}
	shade *= 1.07 - 0.18*(py/wh)
	if w.mood != "live" {
		glass := moonGlass
		if w.mood == "fault" {
			glass = brokenGlass
		}
		return rgb{glass[0] * shade, glass[1] * shade, glass[2] * shade}, true
	}
	flicker := 1 + w.lit*0.05*math.Sin(c.t*(1.3+1.7*rnd(w.i, 1))+w.i*2.1) + w.over*0.1*math.Sin(c.t*7+w.i)
	col := ramp(windowLight, math.Min(1, w.lit*flicker))
	white := w.over * 0.5
	return rgb{
		(col[0] + (255-col[0])*white) * shade,
		(col[1] + (255-col[1])*white) * shade,
		(col[2] + (255-col[2])*white) * shade,
	}, true
}

// 범위를 넘은 창의 둘레에서 반짝이는 빛 점
func (c *windowsCtx) Dots(add render.Dot) {
	for _, w := range c.windows {
		if w.over <= 0.02 {
			continue
		}
		ww := w.x1 - w.x0 + 2
		wh := w.y1 - w.y0 + 2
		count := int(round((ww + wh) * (0.6 + w.over)))
		for j := 0; j < count; j++ {
			fj := float64(j)
			twinkle := math.Sin(c.t*(3+3*rnd(w.i+fj, 7)) + fj*1.7)
			if twinkle < 0.2 {
				continue
			}
			// 둘레를 따라 한 바퀴
			p := rnd(w.i*31+fj, 8) * 2 * (ww + wh)
			var sx, sy float64
			switch {
			case p < ww:
				sx, sy = p, -1
			case p < ww+wh:
				sx, sy = ww, p-ww
			case p < 2*ww+wh:
				sx, sy = p-ww-wh, wh
			default:
				sx, sy = -1, p-2*ww-wh
			}
			lift := 0.0
			if sy < 0 {
				lift = -0.3
			}
			add(w.x0-0.5+sx, w.y0-0.5+sy+lift, rgb{255, 226, 160}, twinkle, false)
		}
	}
}

const windowGlow = ".:=+#@"

func (c *windowsCtx) Glyph(cx, cy int) (rune, string, bool) {
	w := c.find(float64(cx), float64(cy*2))
	if w == nil {
		w = c.find(float64(cx), float64(cy*2+1))
	}
	if w == nil {
		return 0, "", false
	}
	switch w.mood {
	case "fault":
		return 'x', "red", true
	case "rest":
		return ':', "blue", true
	}
	if w.lit < 0.08 {
		// 꺼진 창은 네 귀만 찍는다
		x := float64(cx)
		yc := float64(cy)
		corner := (x == w.x0 || x == w.x1) && (yc == math.Floor(w.y0/2) || yc == math.Floor(w.y1/2))
		if corner {
			return '.', "dim", true
		}
		return 0, "", false
	}
	ch := windowGlow[int(math.Min(float64(len(windowGlow)-1), math.Floor(w.lit*float64(len(windowGlow)))))]
	tone := "yellow"
	if w.over > 0 || w.lit > 0.8 {
		tone = "bright"
	}
	return rune(ch), tone, true
}
