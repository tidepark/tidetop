package scenes

// 물결. 값 둘을 읽는다: 첫째는 물의 높이, 둘째는 물결의 거칠기다.
//
// 맺음
//   - 첫째 값의 채움(fill)이 물이 찬 높이다. 0 이어도 바닥에 얇게 깔려 있다.
//   - 둘째 값의 채움이 물결의 높이다. 0 이면 수면이 반듯하고 아무것도 움직이지 않는다.
//     거칠어지면 마루에 거품이 서고 물보라가 튄다. 범위를 넘으면(over) 물보라가 더 높이 튄다.
//   - 둘째 값이 없으면 잔잔한 물이다.
//   - 쉼(idle)과 끝남(done)은 어두운 밤의 물, 고장(fault)은 잿빛으로 탁한 물이다. 둘 다 멈춰 있다.
//
// 물결이 가는 빠르기는 물결마다 정해져 있다. 값은 높이만 바꾼다.

import (
	"math"

	"github.com/tidepark/tidetop/internal/render"
)

// 물결 셋을 겹친다. len: 길이(화면 높이의 몇 배), speed: 빠르기, part: 몫, dir: 가는 쪽
var waves = []struct{ len, speed, part, dir, phase float64 }{
	{1.9, 0.9, 0.55, 1, 0},
	{0.83, 1.6, 0.3, -1, 1.7},
	{0.37, 2.7, 0.15, 1, 0.4},
}

// 깊이(0 수면 ~ 1 깊은 곳) → 색
var water = map[string][]stop{
	"live":  {{0, rgb{104, 204, 212}}, {0.07, rgb{44, 150, 184}}, {0.4, rgb{18, 88, 148}}, {1, rgb{8, 38, 88}}},
	"rest":  {{0, rgb{72, 88, 114}}, {0.1, rgb{40, 54, 82}}, {1, rgb{16, 22, 42}}},
	"fault": {{0, rgb{116, 118, 112}}, {0.1, rgb{74, 76, 72}}, {1, rgb{32, 34, 34}}},
}
var foam = rgb{236, 248, 250}

type Tide struct{}

func (Tide) Name() string    { return "tide" }
func (Tide) Local() string   { return "mem-cpu" }
func (Tide) Skins() []string { return nil }

type tideCtx struct {
	noGlitch
	W, H                   float64
	t                      float64
	mood                   string
	rough, over, base, amp float64
	surf, lift             []float32 // JavaScript 의 Float32Array 와 같게
	box                    render.Box
}

func (c *tideCtx) wave(x float64) float64 {
	sum := 0.0
	for _, w := range waves {
		sum += w.part * math.Sin((x*2*math.Pi)/(w.len*c.H)-w.dir*c.t*w.speed+w.phase)
	}
	return sum
}

func (Tide) Frame(in render.Input) render.Ctx {
	a := first(in.Values)
	mood := moodOf(a.State)
	c := &tideCtx{W: float64(in.W), H: float64(in.H), t: in.Time, mood: mood}
	hasB := len(in.Values) > 1
	calm := mood != "live" || !hasB || in.Values[1].State != "live"
	if !calm {
		b := in.Values[1]
		c.rough = level(b)
		c.over = or0(b.Over)
	}
	c.base = math.Max(1, c.H*(0.05+0.8*level(a))) // 수면의 높이 (바닥에서부터)
	c.amp = c.rough * math.Min(c.H*0.1, c.base*0.8+1) * (1 + 0.4*c.over)
	c.surf = make([]float32, in.W)
	c.lift = make([]float32, in.W)
	crest := 0.0
	for x := 0; x < in.W; x++ {
		c.lift[x] = float32(c.wave(float64(x)))
		c.surf[x] = float32(c.base + c.amp*float64(c.lift[x]))
		crest = math.Max(crest, float64(c.surf[x]))
	}
	// 물이 있는 네모. 이 위는 비어 있다.
	c.box = render.Box{X0: 0, X1: c.W - 1, Y0: math.Floor(c.H-1-crest) - 2, Y1: c.H - 1}
	return c
}

func (c *tideCtx) Box() *render.Box { return &c.box }

func (c *tideCtx) Pixel(xi, yi int) (rgb, bool) {
	x := float64(xi)
	yb := c.H - 1 - float64(yi)
	depth := float64(c.surf[xi]) - yb
	if depth < 0 {
		return rgb{}, false
	}
	// 마루의 거품
	if c.rough > 0.4 && float64(c.lift[xi]) > 0.66-0.4*c.rough {
		// 두께가 들쭉날쭉한 얇은 띠
		if depth < 0.5+c.rough*2.2*nz(x*0.45+1.7, c.t*0.8) {
			return foam, true
		}
	}
	col := ramp(water[c.mood], clamp(depth/(c.H*0.55)))
	// 물속의 결. 잔잔하면 박혀 있고, 거칠수록 일렁인다.
	still := nz(x*0.11+3.3, yb*0.5+7.1) - 0.5
	moving := nz(x*(5/c.H)+c.t*0.4, yb*(11/c.H)-c.t*0.25) - 0.5
	k := 1 + still*0.1 + moving*0.55*c.rough
	return rgb{col[0] * k, col[1] * k, col[2] * k}, true
}

// 물보라. 낱개마다 튀는 자리와 주기가 정해져 있고, 거칠기는 보이는 개수와 높이만 바꾼다.
func (c *tideCtx) Dots(add render.Dot) {
	if c.rough <= 0.25 {
		return
	}
	n := int(clampTo(round(c.W/2), 10, 90))
	for i := 0; i < n; i++ {
		fi := float64(i)
		th := 0.25 + ((fi+0.5)/float64(n))*0.75
		show := smooth(th, th+0.15, c.rough+c.over)
		if show <= 0 {
			continue
		}
		s := mod(c.t/(1.2+1.6*rnd(fi, 1))+rnd(fi, 2), 1)
		home := math.Floor(rnd(fi, 3) * c.W)
		height := c.H * (0.05 + 0.13*rnd(fi, 4)) * (0.5 + c.rough*0.5 + c.over)
		y := float64(c.surf[int(home)]) + 1 + height*4*s*(1-s)
		x := home + (rnd(fi, 5)-0.5)*c.H*0.2*s
		add(x, c.H-1-y, rgb{196, 232, 238}, show*(1-s), false)
	}
}

var tideTone = map[string]string{"live": "blue", "rest": "dim", "fault": "grey"}

func (c *tideCtx) Glyph(cx, cy int) (rune, string, bool) {
	yb := c.H - 1 - float64(cy*2+1)
	depth := float64(c.surf[cx]) - yb
	if depth < -1 {
		return 0, "", false
	}
	if depth < 1 {
		crest := c.rough > 0.4 && float64(c.lift[cx]) > 0.62-0.45*c.rough
		if c.mood == "live" {
			if crest {
				return '^', "bright", true
			}
			return '~', "cyan", true
		}
		if crest {
			return '^', tideTone[c.mood], true
		}
		return '~', tideTone[c.mood], true
	}
	if depth < c.H*0.25 {
		return ':', tideTone[c.mood], true
	}
	return '.', tideTone[c.mood], true
}
