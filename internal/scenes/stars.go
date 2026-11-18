package scenes

// 별하늘. 값 하나를 읽는다. 값은 「여유」다: 높을수록 넉넉하다.
//
// 맺음
//   - 채움(fill)이 하늘이 맑은 정도다. 단계는 범위 전체에 고르게 나눈다.
//     하늘빛은 전 구간에 걸쳐 조금씩 밝아진다. 아래쪽 절반에서는 구름이 걷히고, 위쪽 절반에서는 은하수가 진해지고
//     유성이 떨어진다. 거의 다 차면(0.83 넘어) 위쪽에 오로라가 일렁인다. 별의 수는 채움을 따른다.
//   - 0.4 를 넘으면 군데군데 별이 잠깐 십자로 반짝인다.
//   - 은하수가 비껴가는 두 구석에도 볼 것을 둔다: 오른쪽 아래의 고리 있는 행성, 왼쪽 위의 작은 은하.
//   - 범위 아래로 내려가면(under) 깨진 모니터처럼 가로세로 줄이 가고 깜빡인다.
//   - 쉼(idle)과 끝남(done)은 옅은 안개가 낀 하늘, 고장(fault)은 잿빛 별 몇 개다. 둘 다 멈춰 있다.
//
// 스킨: milkyway(기본), galaxy. 은하수, 구름, 별자리는 화면 크기로만 정하므로 크기마다 한 번 셈해 둔다.

import (
	"math"

	"github.com/tidepark/tidetop/internal/glitch"
	"github.com/tidepark/tidetop/internal/render"
)

var (
	bandRamp    = []stop{{0, rgb{18, 16, 34}}, {0.3, rgb{46, 40, 88}}, {0.65, rgb{106, 100, 152}}, {1, rgb{212, 206, 232}}}
	starsCloud  = []stop{{0, rgb{20, 22, 30}}, {1, rgb{46, 50, 64}}}
	skyColor    = rgb{16, 24, 58}
	auroraLow   = rgb{70, 230, 150}
	auroraHigh  = rgb{136, 92, 214}
	starRamp    = []stop{{0, rgb{120, 130, 170}}, {0.6, rgb{210, 214, 236}}, {1, rgb{255, 246, 220}}}
	meteorColor = rgb{255, 244, 214}
	galaxyRamp  = []stop{{0, rgb{18, 18, 40}}, {0.3, rgb{48, 54, 112}}, {0.6, rgb{112, 132, 204}}, {1, rgb{226, 230, 252}}}
	coreColor   = rgb{255, 234, 196}
	planetRamp  = []stop{{0, rgb{70, 44, 40}}, {0.45, rgb{150, 96, 70}}, {0.8, rgb{214, 160, 112}}, {1, rgb{244, 214, 170}}}
	ringColor   = rgb{200, 184, 150}
	starsSkins  = []string{"milkyway", "galaxy"}
	meteorSteps = []float64{0.55, 0.7, 0.82, 0.92} // 유성 자리마다 떨어지기 시작하는 문턱
)

const (
	auroraCut  = 0.1 // 이보다 옅은 오로라는 그리지 않는다
	auroraRay  = 0.5 // 글자 그림: 이보다 짙으면 | 로 긋는다
	auroraGlow = 0.3 // 글자 그림: 이보다 짙으면 ! 로 찍는다. 더 옅으면 비운다
	bandCut    = 0.2 // 은하수도 이보다 옅으면 그리지 않는다
	fly        = 70  // 유성의 빠르기 (점/초)
	sparkLife  = 0.9 // 별 하나가 반짝 빛나는 동안 (초)
	gTilt      = -0.42
	gFlat      = 0.48
	gWind      = 2.4
)

type Stars struct{}

func (Stars) Name() string    { return "stars" }
func (Stars) Local() string   { return "free" }
func (Stars) Skins() []string { return starsSkins }

type star struct {
	x, y, b, need, phase, rate float64
	twinkle                    bool
}

type starsLayout struct {
	W, H                  int
	skin                  string
	band, bandNeed, cloud []float32
	core, arm, radius     []float32 // 은하 스킨
	mini, miniCore        []float32 // 은하수 스킨의 작은 은하
	stars                 []star
	gcx, gcy, gR0         float64
}

func (L *starsLayout) toScreen(r, th float64) (float64, float64) {
	cs, sn := math.Cos(gTilt), math.Sin(gTilt)
	u := r * L.gR0 * math.Cos(th)
	v := r * L.gR0 * math.Sin(th) * gFlat
	return L.gcx + u*cs - v*sn, L.gcy + u*sn + v*cs
}

// 은하: 기울어진 원반에 두 팔이 감긴 모습. 가운데는 따뜻하고 팔은 푸르다.
func (L *starsLayout) galaxy() {
	W, H := float64(L.W), float64(L.H)
	L.gcx, L.gcy = W*0.55, H*0.5
	L.gR0 = math.Min(W*0.5, H*0.95)
	cs, sn := math.Cos(gTilt), math.Sin(gTilt)
	n := L.W * L.H
	L.core, L.arm, L.radius = make([]float32, n), make([]float32, n), make([]float32, n)
	for y := 0; y < L.H; y++ {
		for x := 0; x < L.W; x++ {
			i := y*L.W + x
			fx, fy := float64(x), float64(y)
			dx, dy := fx+0.5-L.gcx, fy+0.5-L.gcy
			u := dx*cs + dy*sn
			v := (-dx*sn + dy*cs) / gFlat
			r := math.Hypot(u, v) / L.gR0
			th := math.Atan2(v, u)
			wave := (1 + math.Cos(2*(th-gWind*math.Log(r+0.05)))) / 2
			dust := 0.55 + 0.45*fbm(fx*0.12, fy*0.16)
			q := r / 0.17
			L.radius[i] = float32(r)
			L.core[i] = float32(math.Exp(-(q * q)) * 1.1)
			// 팔 사이에도 옅은 빛이 있다
			L.arm[i] = float32(math.Exp(-r*2.4) * (0.3 + 0.7*math.Pow(wave, 2.2)) * dust * smooth(1.05, 0.7, r) * 1.45)
		}
	}
}

// 왼쪽 위의 멀리 있는 작은 은하(은하수 스킨). 은하수가 비껴가는 구석을 채운다.
func (L *starsLayout) miniGalaxy() {
	W, H := float64(L.W), float64(L.H)
	cx, cy := W*0.17, H*0.2
	r0 := math.Max(3, math.Min(W*0.15, H*0.24))
	cs, sn := math.Cos(0.5), math.Sin(0.5)
	n := L.W * L.H
	L.mini, L.miniCore = make([]float32, n), make([]float32, n)
	for y := math.Max(0, math.Floor(cy-r0)); y <= math.Min(H-1, cy+r0); y++ {
		for x := math.Max(0, math.Floor(cx-r0*1.2)); x <= math.Min(W-1, cx+r0*1.2); x++ {
			dx, dy := x+0.5-cx, y+0.5-cy
			u := dx*cs + dy*sn
			v := (-dx*sn + dy*cs) / 0.55
			r := math.Hypot(u, v) / r0
			if r > 1.1 {
				continue
			}
			wave := (1 + math.Cos(2*(math.Atan2(v, u)-2.2*math.Log(r+0.08)))) / 2
			i := int(y)*L.W + int(x)
			q := r / 0.18
			L.miniCore[i] = float32(math.Exp(-(q * q)))
			L.mini[i] = float32(float64(L.miniCore[i]) + math.Exp(-r*2.6)*(0.3+0.7*wave*wave)*smooth(1.05, 0.6, r)*1.2)
		}
	}
}

var starsCache *starsLayout

func starsLayoutOf(W, H int, skin string) *starsLayout {
	if c := starsCache; c != nil && c.W == W && c.H == H && c.skin == skin {
		return c
	}
	n := W * H
	fW, fH := float64(W), float64(H)
	L := &starsLayout{W: W, H: H, skin: skin, band: make([]float32, n), bandNeed: make([]float32, n), cloud: make([]float32, n)}
	// 은하수: 왼쪽 아래에서 오른쪽 위로 건너는 띠
	ax, ay := -0.1*fW, 0.95*fH
	bx, by := 1.1*fW, 0.02*fH
	length := math.Hypot(bx-ax, by-ay)
	nx := -(by - ay) / length
	ny := (bx - ax) / length
	half := math.Max(3, fH*0.17)
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			i := y*W + x
			fx, fy := float64(x), float64(y)
			d := ((fx+0.5-ax)*nx + (fy+0.5-ay)*ny) / half
			glow := math.Exp(-d * d * 1.9)
			// 빛은 군데군데 뭉쳐 있다
			dust := 0.42 + 0.58*smooth(0.2, 0.85, fbm(fx*0.07, fy*0.13))
			// 가운데를 따라 어두운 틈이 난다
			q := d / 0.35
			rift := smooth(0.5, 0.72, fbm(fx*0.05+11, fy*0.11)) * math.Exp(-(q * q))
			L.band[i] = float32(glow * dust * (1 - 0.8*rift))
			// 은하수는 가장자리부터 군데군데 줄어든다. 밝은 가운데가 가장 오래 남는다.
			L.bandNeed[i] = float32(clamp(1.05 - float64(L.band[i])*1.2 + (nz(fx*0.2, fy*0.3)-0.5)*0.35))
			// 구름은 창 크기에 맞춰 흩어 놓는다. 작은 창이나 일부만 보여도 낀 정도가 보인다.
			L.cloud[i] = float32(fbm((fx*5)/fW+4.4, (fy*4.5)/fH+1.9))
		}
	}
	if skin == "galaxy" {
		L.galaxy()
	} else {
		L.miniGalaxy()
	}
	// 별: 하늘에 고루, 은하수에는 더 촘촘히. 별마다 남는 문턱이 있다.
	plain := round(fW * fH * 0.018)
	for i := 0.0; i < plain; i++ {
		need := 0.03 + 0.95*rnd(i, 4)
		if i < plain*0.04 {
			need = 0
		}
		b := rnd(i, 3)
		L.stars = append(L.stars, star{x: rnd(i, 1) * fW, y: rnd(i, 2) * fH, b: b * b, need: need})
	}
	inBand := round(fW * fH * 0.014)
	if skin != "galaxy" {
		for i := 0.0; i < inBand; i++ {
			u := rnd(i, 11)
			off := (rnd(i, 12) + rnd(i, 13) + rnd(i, 14) - 1.5) * half * 0.7
			L.stars = append(L.stars, star{
				x: lerp(ax, bx, u) + nx*off, y: lerp(ay, by, u) + ny*off,
				b: math.Pow(rnd(i, 15), 3), need: 0.1 + 0.88*rnd(i, 16),
			})
		}
	} else {
		// 팔을 따라 촘촘한 별
		for i := 0.0; i < inBand*0.7; i++ {
			r := 0.12 + 0.88*math.Pow(rnd(i, 11), 0.8)
			th := mod(i, 2)*math.Pi + gWind*math.Log(r+0.05) + (rnd(i, 12)+rnd(i, 13)-1)*0.5
			x, y := L.toScreen(r, th)
			L.stars = append(L.stars, star{x: x, y: y, b: math.Pow(rnd(i, 15), 3), need: 0.1 + 0.88*rnd(i, 16)})
		}
	}
	for i := range L.stars {
		fi := float64(i)
		L.stars[i].twinkle = i%11 == 0 // 반짝이는 것은 열에 하나쯤
		L.stars[i].phase = rnd(fi, 21) * 6.28
		L.stars[i].rate = 0.8 + 1.2*rnd(fi, 22)
	}
	starsCache = L
	return L
}

// 멈춰 있는 빛(하늘, 은하수, 구름)은 값이 그대로인 동안 한 번만 셈해 둔다.
// 코어마다 나눠 볼 때는 조각마다 값이 달라서 값마다 따로 둔다.
type stillLight struct {
	done, ok bool
	col      rgb
	kind     string
}

var (
	memoOf *starsLayout
	memos  = map[float64][]stillLight{}
)

type planetAtT struct{ x, y, r float64 }

type starsCtx struct {
	noBox
	W, H                                                    int
	t, f                                                    float64
	mood                                                    string
	L                                                       *starsLayout
	memo                                                    []stillLight
	core, reach, band, aurora, overcast, sky, mist, twinkle float64
	planet                                                  *planetAtT
	glitch                                                  []glitch.Line
	sparks                                                  map[int]float64
}

func (Stars) Frame(in render.Input) render.Ctx {
	v := first(in.Values)
	mood := moodOf(v.State)
	f := 0.0
	if mood != "fault" {
		f = level(v)
	}
	skin := starsSkins[0]
	for _, s := range starsSkins {
		if s == in.Skin {
			skin = s
		}
	}
	L := starsLayoutOf(in.W, in.H, skin)
	t := 0.0
	if mood == "live" {
		t = in.Time
	}
	W, H := float64(in.W), float64(in.H)
	// 멈춘 빛은 값을 1/50 단위로 묶어 셈한다. 값이 조금씩 움직여도 셈해 둔 것을 다시 쓴다.
	fq := round(f*50) / 50
	if memoOf != L || len(memos) > 128 {
		memos = map[float64][]stillLight{}
		memoOf = L
	}
	memo, ok := memos[fq]
	if !ok {
		memo = make([]stillLight, in.W*in.H)
		memos[fq] = memo
	}
	c := &starsCtx{
		W: in.W, H: in.H, t: t, f: f, mood: mood, L: L, memo: memo,
		core:     smooth(0.05, 0.5, fq), // 은하의 가운데
		reach:    0.25 + 0.85*fq,        // 은하의 팔이 보이는 반지름
		band:     smooth(0.3, 0.95, fq), // 은하수가 남은 정도
		overcast: 1 - smooth(0.05, 0.6, fq),
		sky:      smooth(0, 0.25, fq) * (0.35 + 0.65*fq),
	}
	if mood == "live" {
		c.aurora = smooth(0.83, 0.97, f)
		c.twinkle = smooth(0.2, 0.5, f)
		c.glitch = glitch.Make(smooth(0, 0.1, or0(v.Under)), in.W, in.H, t, in.GlitchSeed)
	}
	if mood == "rest" {
		c.mist = 0.6
	}
	// 오른쪽 아래의 행성: 넉넉할수록 크다. 어두워지지 않고 작아진다.
	if fq > 0.15 {
		c.planet = &planetAtT{W * 0.8, H * 0.74, math.Max(2.2, H*0.12) * (0.45 + 0.55*smooth(0.15, 0.9, fq))}
	}
	c.sparks = c.makeSparks()
	return c
}

func (c *starsCtx) Glitch() []glitch.Line { return c.glitch }

// 반짝임: 자리마다 정해진 주기로 별 하나를 골라 잠깐 십자로 빛낸다. 점 번호 → 밝기(0~1).
func (c *starsCtx) makeSparks() map[int]float64 {
	out := map[int]float64{}
	if c.mood != "live" {
		return out
	}
	W, H := float64(c.W), float64(c.H)
	spots := int(math.Max(3, round(W*H/450)))
	var seen []star
	for _, s := range c.L.stars {
		if s.need <= c.f && s.x >= 0 && s.x < W && s.y >= 0 && s.y < H {
			seen = append(seen, s)
		}
	}
	put := func(x, y int, k float64) {
		if x < 0 || y < 0 || x >= c.W || y >= c.H {
			return
		}
		i := y*c.W + x
		out[i] = math.Max(out[i], k)
	}
	for j := 0; j < spots && len(seen) > 0; j++ {
		fj := float64(j)
		if c.f < 0.4+0.6*((fj+0.5)/float64(spots)) {
			continue
		}
		period := 2.5 + 3.5*rnd(fj, 31)
		shift := rnd(fj, 32) * period
		k := math.Floor((c.t + shift) / period)
		age := mod(c.t+shift, period)
		if age > sparkLife {
			continue
		}
		s := seen[int(math.Floor(rnd(k*13+fj, 33)*float64(len(seen))))]
		x, y := int(math.Floor(s.x)), int(math.Floor(s.y))
		if c.cloudAt(y*c.W+x) > 0.3 {
			continue
		}
		glow := math.Sin((math.Pi * age) / sparkLife)
		put(x, y, glow)
		arms := 1
		if glow > 0.55 {
			arms = 2
		}
		for r := 1; r <= arms; r++ {
			arm := glow * 0.35
			if r == 1 {
				arm = glow * 0.7
			}
			put(x+r, y, arm)
			put(x-r, y, arm)
			put(x, y+r, arm)
			put(x, y-r, arm)
		}
	}
	return out
}

func starsFade(col rgb, k float64) rgb {
	g := col[0]*0.3 + col[1]*0.55 + col[2]*0.15
	m := mix(col, rgb{g, g, g * 1.08}, k)
	return rgb{m[0] * (1 - 0.25*k), m[1] * (1 - 0.25*k), m[2] * (1 - 0.25*k)}
}

func starsGrey(col rgb) rgb {
	g := (col[0]*0.3 + col[1]*0.55 + col[2]*0.15) * 0.75
	return rgb{g, g, g}
}

// 구름의 짙기 (0~1)
func (c *starsCtx) cloudAt(i int) float64 {
	if c.overcast <= 0.02 {
		return 0
	}
	return clamp((float64(c.L.cloud[i]) - (0.82 - 0.3*c.overcast)) / 0.06)
}

// 오로라의 빛 (0~1)과 위로 올라간 정도
func (c *starsCtx) auroraAt(x, y float64) (a, up float64, ok bool) {
	H := float64(c.H)
	if c.aurora <= 0 || y > H*0.55 {
		return 0, 0, false
	}
	base := H * (0.34 + 0.07*math.Sin(x*0.045+c.t*0.15) + 0.03*math.Sin(x*0.11-c.t*0.23) + 0.06*(nz(x*0.02+c.t*0.05, 3.3)-0.5))
	up = (base - y) / (H * 0.24)
	var fall float64
	if up >= 0 {
		fall = math.Exp(-up * 1.6)
	} else {
		q := (up * H * 0.24) / 1.2
		fall = math.Exp(-(q * q))
	}
	rays := 0.35 + 0.65*nz(x*0.4+c.t*0.22, 7.7)
	// 하늘 전체가 아니라 몇 덩이로 나뉘어 걸린다
	patch := smooth(0.22, 0.55, nz(x*0.018+c.t*0.03, 1.1))
	a = fall * rays * patch * c.aurora
	return a, clamp(up), a > 0.02
}

// 점 하나의 빛과 그 종류
func (c *starsCtx) light(x, y int) (rgb, string, bool) {
	i := y*c.W + x
	m := &c.memo[i]
	if !m.done {
		m.col, m.kind, m.ok = c.still(x, y)
		m.done = true
	}
	if m.ok && m.kind == "cloud" {
		return m.col, m.kind, true
	}
	if spark := c.sparks[i]; spark > 0.08 {
		base := rgb{}
		if m.ok {
			base = m.col
		}
		return mix(base, rgb{255, 252, 240}, spark), "spark", true
	}
	if a, up, ok := c.auroraAt(float64(x), float64(y)); ok && a > auroraCut {
		tone := mix(auroraLow, auroraHigh, up)
		k := clamp(a)
		lit := rgb{tone[0] * k, tone[1] * k, tone[2] * k}
		if m.ok {
			lit = rgb{m.col[0] + tone[0]*k, m.col[1] + tone[1]*k, m.col[2] + tone[2]*k}
		}
		return rgb{math.Min(255, lit[0]), math.Min(255, lit[1]), math.Min(255, lit[2])}, "aurora", true
	}
	return m.col, m.kind, m.ok
}

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// 행성과 고리. 고리는 기울어진 납작한 띠이고, 행성 앞(아래쪽)을 지날 때만 몸을 가린다.
func (c *starsCtx) planetAt(x, y float64) (rgb, string, bool) {
	p := c.planet
	if p == nil {
		return rgb{}, "", false
	}
	dx, dy := x+0.5-p.x, y+0.5-p.y
	if math.Abs(dx) > p.r*2.3 || math.Abs(dy) > p.r*1.3 {
		return rgb{}, "", false
	}
	u := (dx*0.96 + dy*0.28) / p.r
	v := (-dx*0.28 + dy*0.96) / (p.r * 0.3)
	rr := math.Hypot(u, v)
	ring := rr > 1.35 && rr < 2.1 && !(rr > 1.66 && rr < 1.76)
	d := math.Hypot(dx, dy) / p.r
	if d < 1 {
		if ring && v > 0 {
			return scale(ringColor, 0.75+0.25*b2f(rr < 1.66)), "ring", true
		}
		// 왼쪽 위에서 빛이 든다. 가로 줄무늬가 옅게 있다.
		lit := clamp(0.55 - (dx+dy)/(p.r*2.4) + 0.08*math.Sin(dy*1.3+0.7))
		return ramp(planetRamp, lit*(1-0.35*math.Pow(d, 4))), "planet", true
	}
	if ring {
		return scale(ringColor, 0.6+0.25*b2f(rr < 1.66)), "ring", true
	}
	return rgb{}, "", false
}

// 멈춰 있는 빛: 구름, 행성, 은하수, 하늘빛
func (c *starsCtx) still(xi, yi int) (rgb, string, bool) {
	i := yi*c.W + xi
	x, y := float64(xi), float64(yi)
	if cl := c.cloudAt(i); cl > 0 {
		return ramp(starsCloud, cl*(0.4+0.6*nz(x*0.3, y*0.5))), "cloud", true
	}
	var col rgb
	has := false
	var b float64
	var sky rgb
	hasSky := c.sky > 0.12
	if hasSky {
		sky = scale(skyColor, c.sky)
	}
	if p, kind, ok := c.planetAt(x, y); ok {
		return p, kind, true
	}
	L := c.L
	if L.skin == "galaxy" {
		// 여유가 줄면 바깥 팔부터 꺼지고 가운데가 마지막까지 남는다
		core := float64(L.core[i]) * c.core
		b = core + float64(L.arm[i])*c.band*smooth(c.reach, c.reach-0.2, float64(L.radius[i]))
		k := clamp((b - bandCut) / (1 - bandCut))
		// 밝은 가운데만 따뜻한 빛이고, 옅은 자리는 푸르다
		if b > bandCut {
			col, has = mix(ramp(galaxyRamp, k), coreColor, clamp((core/b)*k*1.3)), true
		}
	} else {
		need := float64(L.bandNeed[i])
		b = float64(L.band[i]) * (0.55 + 0.45*c.band) * smooth(need, need+0.08, c.band)
		if b > bandCut {
			col, has = ramp(bandRamp, (b-bandCut)/(1-bandCut)), true
		}
		// 작은 은하도 은하수와 같이 옅어진다
		m := float64(L.mini[i]) * c.band
		if m > bandCut && m > b {
			b = m
			k := clamp((m - bandCut) / (1 - bandCut))
			col, has = mix(ramp(galaxyRamp, k), coreColor, clamp(float64(L.miniCore[i])*k)), true
		}
	}
	kind := "sky"
	if has {
		kind = "band"
		if b > 0.5 {
			kind = "bright"
		}
	}
	// 하늘빛 위에 은하수를 얹는다. 은하수의 어두운 가장자리가 하늘보다 어두워지지 않게 한다.
	if hasSky {
		if has {
			col = rgb{math.Max(col[0], sky[0]), math.Max(col[1], sky[1]), math.Max(col[2], sky[2])}
		} else {
			col, has = sky, true
		}
	}
	return col, kind, has
}

func (c *starsCtx) Pixel(x, y int) (rgb, bool) {
	col, _, ok := c.light(x, y)
	if !ok {
		return rgb{}, false
	}
	if c.mood == "fault" {
		return starsGrey(col), true
	}
	if c.mist != 0 {
		col = starsFade(col, c.mist)
	}
	return col, true
}

func (c *starsCtx) Dots(add render.Dot) {
	tint := func(col rgb) rgb {
		if c.mood == "fault" {
			return starsGrey(col)
		}
		if c.mist != 0 {
			return starsFade(col, c.mist)
		}
		return col
	}
	W, H := float64(c.W), float64(c.H)
	for _, s := range c.L.stars {
		if c.f < s.need && s.need > 0 {
			continue
		}
		if s.x < 0 || s.x >= W || s.y < 0 || s.y >= H {
			continue
		}
		if c.cloudAt(int(math.Floor(s.y))*c.W+int(math.Floor(s.x))) > 0.3 {
			continue
		}
		b := s.b
		// 반짝임도 여유를 따른다. 0 에서는 아무것도 움직이지 않는다.
		if s.twinkle && c.twinkle > 0 {
			b *= 1 - c.twinkle*(0.45-0.45*math.Sin(c.t*s.rate+s.phase))
		}
		add(s.x, s.y, tint(ramp(starRamp, b)), 0.3+b, true)
	}
	if c.mood != "live" {
		return
	}
	// 유성: 자리마다 정해진 주기로 한 번씩 긋는다. 여유는 몇 자리가 쓰이는지만 정한다.
	for jj, th := range meteorSteps {
		j := float64(jj)
		if c.f < th {
			continue
		}
		period := 5 + 6*rnd(j, 1)
		k := math.Floor((c.t + rnd(j, 2)*period) / period)
		age := mod(c.t+rnd(j, 2)*period, period)
		life := 0.9
		if age > life {
			continue
		}
		x0 := (0.2 + 0.75*rnd(k*4+j, 3)) * W
		y0 := (0.05 + 0.3*rnd(k*4+j, 4)) * H
		ang := 2.3 + 0.5*rnd(k*4+j, 5) // 왼쪽 아래로
		head := age * fly
		fade := 1 - age/life
		for s := 0.0; s < 14; s++ {
			d := head - s*0.8
			if d < 0 {
				break
			}
			x := x0 + math.Cos(ang)*d
			y := y0 + math.Sin(ang)*d
			add(x, y, scale(meteorColor, 1-s/16), fade*(1.4-s/14), true)
		}
	}
}

func (c *starsCtx) Glyph(cx, cy int) (rune, string, bool) {
	y := cy * 2
	_, kind, ok := c.light(cx, y)
	if !ok {
		y++
		_, kind, ok = c.light(cx, y)
	}
	if !ok {
		return 0, "", false
	}
	// 글자 그림의 오로라는 짙은 빛줄기만 긋는다. 옅은 곳은 밑의 하늘을 보인다
	ray := ' '
	if kind == "aurora" {
		a, _, _ := c.auroraAt(float64(cx), float64(y))
		switch {
		case a >= auroraRay:
			ray = '|'
		case a >= auroraGlow:
			ray = '!'
		default:
			m := c.memo[y*c.W+cx]
			if !m.ok {
				return 0, "", false
			}
			kind = m.kind
		}
	}
	tone := ""
	if c.mood == "fault" {
		tone = "grey"
	} else if c.mood == "rest" {
		tone = "dim"
	}
	or := func(def string) string {
		if tone != "" {
			return tone
		}
		return def
	}
	switch kind {
	case "cloud":
		return '~', "dim", true
	case "spark":
		return '*', "bright", true
	case "planet":
		return '@', or("yellow"), true
	case "ring":
		return '-', or("yellow"), true
	case "sky":
		return 0, "", false
	case "aurora":
		return ray, or("green"), true
	case "bright":
		return ':', or("bright"), true
	}
	return '.', or("blue"), true
}
