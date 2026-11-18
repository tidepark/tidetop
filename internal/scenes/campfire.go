package scenes

// 모닥불. 값 하나를 읽는다.
//
// 맺음
//   - 채움(fill)이 불의 높이와 너비다. 0 이면 장작 사이의 불씨만 남고 아무것도 움직이지 않는다.
//   - 불티의 개수도 채움을 따른다. 범위를 넘으면(over) 더 높이, 더 넓게 날리는 불티가 더해진다.
//   - 쉼(idle)과 끝남(done)은 재워 둔 불이다. 불꽃이 없고 장작이 푸르게 식어 멈춰 있다.
//   - 고장(fault)은 꺼진 불이다. 잿빛 장작에서 연기만 오른다.
//
// 움직임은 모두 시각의 함수다. 흐르는 빠르기는 화면 크기로만 정하고 값은 높이, 너비, 개수, 진하기만 바꾼다.

import (
	"math"

	"github.com/tidepark/tidetop/internal/render"
)

// 불의 온도 → 색. 0 은 비어 있다.
var heatRamp = []stop{
	{0.0, rgb{58, 9, 6}},
	{0.07, rgb{132, 20, 6}},
	{0.18, rgb{208, 54, 9}},
	{0.36, rgb{243, 114, 18}},
	{0.56, rgb{255, 176, 46}},
	{0.76, rgb{255, 222, 112}},
	{0.92, rgb{255, 242, 184}},
	{1.0, rgb{255, 250, 226}},
}

const fireCut = 0.045 // 이보다 식은 자리는 비운다

// 불꽃은 혀 셋이 겹친 것이다. ox: 가운데에서 벗어난 정도, hs: 높이, ws: 너비
var lobes = []struct{ ox, hs, ws, seed float64 }{
	{0.0, 1.0, 1.0, 1},
	{-0.52, 0.6, 0.62, 2},
	{0.56, 0.5, 0.58, 3},
}

const scroll = 2.6 // 불꽃의 결이 오르는 빠르기 (잡음 칸/초). 값과 무관하다.

type Campfire struct{}

func (Campfire) Name() string    { return "campfire" }
func (Campfire) Local() string   { return "cpu" }
func (Campfire) Skins() []string { return nil }

type lobeRow struct {
	ok              bool
	v, sway, w, off float64
}

type campfireCtx struct {
	noGlitch
	W, H                                                   int
	t                                                      float64
	mood                                                   string
	burn, over                                             float64
	ground, hmax, logHalf, logThick, base, h, half, cx, fq float64
	swing                                                  float64
	rows                                                   [][]lobeRow
	box                                                    render.Box
}

func (Campfire) Frame(in render.Input) render.Ctx {
	v := first(in.Values)
	W, H := float64(in.W), float64(in.H)
	t := in.Time
	c := &campfireCtx{W: in.W, H: in.H, t: t, mood: moodOf(v.State)}
	if c.mood == "live" {
		c.burn = level(v)
		c.over = or0(v.Over)
	}
	c.ground = math.Max(1, round(H*0.08)) // 장작 아래의 빈 줄
	c.hmax = (H - c.ground) * 0.88
	c.logHalf = clampTo(c.hmax*0.36, 3, W*0.45)
	c.logThick = math.Max(1.2, c.hmax*0.05)
	c.base = c.ground + c.logThick // 불꽃이 시작하는 높이 (바닥에서부터)
	// 숨: 불이 클수록 크게 들썩인다. 0 에서는 멈춘다.
	breath := 1 + c.burn*(0.05*math.Sin(t*2.3)+0.035*math.Sin(t*3.9+1.3))
	c.h = c.hmax * math.Pow(c.burn, 0.75) * breath * (1 + 0.07*c.over)
	c.half = math.Max(c.logHalf*0.4, c.h*0.31)
	c.cx = (W - 1) / 2
	reach := math.Max(c.logHalf+1, c.half*2.6)
	c.fq = 5 / c.hmax // 잡음의 촘촘함 (칸/점)
	c.swing = (0.75 + 0.45*(1+0.6*c.over)) * c.half
	c.rows = make([][]lobeRow, in.H+2)
	// 장작과 불꽃이 들어가는 네모. 이 밖은 비어 있다.
	c.box = render.Box{
		X0: math.Floor(c.cx - reach), X1: math.Ceil(c.cx + reach),
		Y0: math.Floor(H-1-math.Max(c.base+c.h, c.ground+c.logThick*3.2)) - 1, Y1: H - 1,
	}
	return c
}

func (c *campfireCtx) Box() *render.Box { return &c.box }

// 불꽃의 윤곽: 아래가 조금 좁고, 곧 부풀었다가 끝으로 갈수록 가늘어진다
func profile(v float64) float64 { return (0.6 + 0.4*smooth(0, 0.2, v)) * math.Pow(1-v, 0.85) }

// 줄마다 한 번만 셈해 두는 것: 혀마다의 높이 비율, 흔들림, 너비, 가운데에서 벗어난 정도
func (c *campfireCtx) row(yb float64) []lobeRow {
	idx := int(yb)
	if idx >= 0 && idx < len(c.rows) && c.rows[idx] != nil {
		return c.rows[idx]
	}
	y := yb - c.base
	r := make([]lobeRow, len(lobes))
	for i, lobe := range lobes {
		s := lobe.seed
		v := (y + 1) / (c.h * lobe.hs)
		if v >= 1 {
			continue
		}
		sway := math.Sin(c.t*(1.05+s*0.31)+s*5+y*c.fq*0.9) * 0.1
		// 부푼 자리가 위로 흘러 올라간다
		bulge := 0.78 + 0.5*nz(s*13.7, y*c.fq*1.5-c.t*scroll*1.15)
		r[i] = lobeRow{true, v, sway, c.half * lobe.ws * profile(v) * bulge, lobe.ox * c.half * (1 - v*0.5)}
	}
	if idx >= 0 && idx < len(c.rows) {
		c.rows[idx] = r
	}
	return r
}

// 가운데에서 dx, 바닥에서 yb 만큼 떨어진 자리의 온도 (0~1)
func (c *campfireCtx) heat(dx, yb float64) float64 {
	y := yb - c.base
	if c.h < 0.6 || y < 0 || y > c.h || math.Abs(dx) > c.half*2.6 {
		return 0
	}
	sum := 0.0
	rows := c.row(yb)
	for i, r := range rows {
		if !r.ok {
			continue
		}
		v, w := r.v, r.w
		// 흔들려도 닿지 못하는 자리는 잡음을 셈하지 않는다
		if math.Abs(dx-r.off) >= w+c.swing*v {
			continue
		}
		s := lobes[i].seed
		// 흔들림: 위로 갈수록 크게 휜다
		bend := nz(dx*c.fq*0.5+s*37, y*c.fq*0.55-c.t*scroll*0.6+s*11) - 0.5
		// 혀: 자리마다 옆으로 밀려 가장자리가 갈라진다
		lick := (fbm(dx*c.fq*0.45+s*23, y*c.fq*1.7-c.t*scroll*1.3+s*19) - 0.5) * (1 + 0.6*c.over)
		u := (dx - r.off - (bend*1.3+r.sway+lick*0.9)*c.half*v) / w
		if u <= -1 || u >= 1 {
			continue
		}
		// 결: 위로 흐르며 불꽃의 가장자리를 뜯어낸다
		tear := fbm(dx*c.fq*1.7+s*7, y*c.fq*1.15-c.t*scroll+s*3)
		body := (1 - u*u) * (0.3 + 0.62*(1-v))
		sum += math.Max(0, body-tear*(0.08+0.5*v))
	}
	// 밑동은 장작에 가려 어둡다
	return clamp(math.Pow(sum, 1.1) * (0.86 + 0.22*c.over) * (0.55 + 0.45*smooth(0, c.logThick*2.5, y)))
}

// 엇갈려 놓인 장작 둘. 장작이면 0~1 의 「가운데에 가까운 정도」, 아니면 -1
func (c *campfireCtx) log(dx, yb float64) float64 {
	y := yb - c.ground
	if y < -0.5 || y > c.logThick*3.2 || math.Abs(dx) > c.logHalf+1 {
		return -1
	}
	rise := c.logThick * 1.9
	for _, dir := range []float64{1, -1} {
		// (-logHalf, 0) 에서 (logHalf × 0.8, rise) 로 가는 굵은 선
		ax := -c.logHalf * dir
		bx := c.logHalf * 0.8 * dir
		k := clamp((dx - ax) / (bx - ax))
		ly := c.logThick*0.5 + rise*k
		if math.Abs(y-ly) <= c.logThick*0.5 && k > 0 && k < 1 {
			return 1 - math.Abs(k-0.55)*1.6
		}
	}
	return -1
}

var logWood = map[string]rgb{"live": {46, 29, 21}, "rest": {30, 34, 48}, "fault": {64, 64, 68}}

// 불씨는 장작이 엇갈린 가운데에 모여 있고, 끝으로 갈수록 나무색이다
func (c *campfireCtx) ember(dx, yb, mid float64) float64 {
	return clamp(mid*1.25) * (0.45 + 0.55*nz(dx*0.27+40.5, yb*0.45+9.5))
}

func (c *campfireCtx) logColor(dx, yb, mid float64) rgb {
	wood := logWood[c.mood]
	if c.mood == "fault" {
		return wood
	}
	spot := c.ember(dx, yb, mid)
	if c.mood == "rest" {
		k := smooth(0.45, 0.9, spot) * 0.6
		return rgb{lerp(wood[0], 124, k), lerp(wood[1], 40, k), lerp(wood[2], 50, k)}
	}
	pulse := 1 + c.burn*0.22*math.Sin(c.t*1.7+dx*0.5+yb*0.8)
	e := smooth(0.3, 1, spot) * (0.42 + 0.58*c.burn) * pulse
	lit := 0.85 + 1.0*c.burn
	dark := rgb{wood[0] * lit, wood[1] * (0.85 + 0.7*c.burn), wood[2] * (0.85 + 0.3*c.burn)}
	if e < 0.03 {
		return dark
	}
	hot := ramp(heatRamp, 0.04+e*0.6)
	k := smooth(0.03, 0.25, e)
	return rgb{lerp(dark[0], hot[0], k), lerp(dark[1], hot[1], k), lerp(dark[2], hot[2], k)}
}

func (c *campfireCtx) Pixel(x, y int) (rgb, bool) {
	yb := float64(c.H - 1 - y)
	dx := float64(x) - c.cx
	if mid := c.log(dx, yb); mid >= 0 {
		return c.logColor(dx, yb, mid), true
	}
	if c.mood != "live" {
		return rgb{}, false
	}
	k := c.heat(dx, yb)
	if k < fireCut {
		return rgb{}, false
	}
	return ramp(heatRamp, k), true
}

// 불티와 연기. 낱개마다 주기와 길이 정해져 있고, 값은 보이는 개수만 바꾼다.
func (c *campfireCtx) Dots(add render.Dot) {
	W, H := float64(c.W), float64(c.H)
	top := H - 1
	if c.mood == "fault" {
		n := int(clampTo(round(W*H/50), 14, 220))
		for i := 0; i < n; i++ {
			fi := float64(i)
			s := mod(c.t/(7+6*rnd(fi, 1))+rnd(fi, 2), 1)
			y := c.base + s*(H-c.base)*(0.55+0.45*rnd(fi, 3))
			spread := c.logHalf * (0.12 + 0.75*s)
			x := c.cx + (rnd(fi, 4)-0.5)*2*spread + math.Sin(s*5+fi)*c.hmax*0.04 + s*s*c.hmax*0.12
			g := 176 - 80*s
			add(x, top-y, rgb{g, g, g + 6}, 1-s, false)
		}
		return
	}
	if c.mood != "live" || c.burn <= 0 {
		return
	}
	c.halo(add)
	n := int(clampTo(round(W*H/110), 10, 120))
	amount := math.Pow(c.burn, 1.4)
	for i := 0; i < n*3; i++ {
		fi := float64(i)
		wild := i >= n // 범위를 넘었을 때만 나오는 불티
		var th, show, period, rise float64
		if wild {
			th = ((fi - float64(n) + 0.5) / float64(n*2)) * 0.8
			show = smooth(th, th+0.12, c.over)
		} else {
			th = 0.06 + ((fi+0.5)/float64(n))*0.9
			show = smooth(th, th+0.12, amount)
		}
		if show <= 0 {
			continue
		}
		if wild {
			period = 1.2 + 1.4*rnd(fi, 1)
		} else {
			period = 2.6 + 2.8*rnd(fi, 1)
		}
		s := mod(c.t/period+rnd(fi, 2), 1)
		side := rnd(fi, 3) - 0.5
		if wild {
			rise = (H - c.base) * (0.6 + 0.5*rnd(fi, 4))
		} else {
			rise = c.hmax * (0.4 + 0.7*rnd(fi, 4))
		}
		y := c.base + c.h*(0.3+0.45*rnd(fi, 5)) + s*rise
		push := 0.6
		if wild {
			push = 2.2
		}
		x := c.cx + side*c.half*1.3 + math.Sin(s*(4+6*rnd(fi, 6))+fi)*c.hmax*0.035*(0.5+s) + side*s*c.hmax*push
		glow := show * math.Pow(1-s, 0.8) * (0.8 + 0.2*math.Sin(c.t*9+fi*2.4))
		if glow < 0.1 {
			continue
		}
		add(x, top-y, ramp(heatRamp, 0.4+glow*0.5), glow, false)
	}
}

// 불 둘레의 옅은 빛. 점자의 점을 성기게 찍어 가장자리가 어둠으로 풀리게 한다.
// 점의 자리는 화면에 박혀 있고, 불이 숨 쉬는 데 따라 켜지고 꺼진다.
func (c *campfireCtx) halo(add render.Dot) {
	W, H := float64(c.W), float64(c.H)
	rx := c.half * 1.7
	ry := c.h * 0.72
	x0 := math.Max(0, math.Floor((c.cx-rx)*2))
	x1 := math.Min(W*2-1, math.Ceil((c.cx+rx)*2))
	y1 := math.Min(H*2-1, math.Ceil((c.base+c.h*1.25)*2))
	for by := math.Floor(c.base * 2); by <= y1; by++ {
		for bx := x0; bx <= x1; bx++ {
			dx := bx/2 - c.cx
			y := by/2 - c.base
			d := (dx*dx)/(rx*rx) + ((y-c.h*0.4)*(y-c.h*0.4))/(ry*ry)
			if d >= 1 {
				continue
			}
			g := (1 - d) * (1 - d) * (0.25 + 0.55*c.burn)
			if rnd(bx*0.37+1.3, by*0.61+2.9) < g*0.16 {
				add(bx/2+0.25, H-1-by/2+0.25, rgb{96 + 70*g, 22 + 30*g, 8}, 0.05, false)
			}
		}
	}
}

// low: 칸 하나에 글자 하나
const flame = ".:;*%#@"

func (c *campfireCtx) Glyph(cx, cy int) (rune, string, bool) {
	yb := float64(c.H - 1 - (cy*2 + 1))
	dx := float64(cx) - c.cx
	for _, off := range []float64{0, 1} {
		mid := c.log(dx, yb+off)
		if mid < 0 {
			continue
		}
		if c.mood == "fault" {
			return '=', "dim", true
		}
		if c.mood == "rest" {
			return '=', "blue", true
		}
		if smooth(0.3, 1, c.ember(dx, yb+off, mid))*(0.42+0.58*c.burn) > 0.12 {
			return '*', "red", true
		}
		return '=', "dim", true
	}
	if c.mood != "live" {
		return 0, "", false
	}
	k := math.Max(c.heat(dx, yb), c.heat(dx, yb+1))
	if k < 0.08 {
		return 0, "", false
	}
	ch := flame[int(math.Min(float64(len(flame)-1), math.Floor(k*float64(len(flame)))))]
	tone := "bright"
	if k < 0.32 {
		tone = "red"
	} else if k < 0.7 {
		tone = "yellow"
	}
	return rune(ch), tone, true
}
