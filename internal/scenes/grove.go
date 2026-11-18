package scenes

// 숲. 값 하나를 읽는다. 값은 「여유」다: 높을수록 넉넉하다.
// 이 컴퓨터의 값을 쓸 때는 쓰지 않는 CPU 의 비율(100 − 사용률)을 준다.
//
// 맺음
//   - 채움(fill)이 우거진 정도다. 찰수록 나무와 숲에 잎이 들어차고, 땅이 푸르고, 연못이 차고,
//     폭포가 굵어지고, 해가 밝아지고, 구름이 떠간다.
//   - 0 이면 앙상한 가지와 마른 땅, 빈 연못 바닥만 남고 아무것도 움직이지 않는다.
//   - 잎은 덩이마다 남는 문턱이 정해져 있어서, 줄어들 때 군데군데 성겨진다.
//   - 계절의 단서(단풍, 눈)는 쓰지 않는다. 메마름과 우거짐만으로 읽는다.
//   - 범위 아래로 내려가면(under) 깨진 모니터처럼 가로세로 줄이 가고 깜빡인다.
//   - 쉼(idle)과 끝남(done)은 옅은 안개가 낀 숲이다. 고장(fault)은 잿빛 가지만 남는다. 둘 다 멈춰 있다.
//
// 스킨: night(기본) 은 하늘을 칠하지 않고 달과 멈춘 별을 둔다. day 는 하늘을 한 색으로 칠하고 해를 둔다.
// 나무, 숲, 땅, 바위는 화면 크기로만 정해지므로 크기마다 한 번 셈해 둔다.

import (
	"math"

	"github.com/tidepark/tidetop/internal/glitch"
	"github.com/tidepark/tidetop/internal/render"
)

var (
	leafRamp   = []stop{{0, rgb{26, 76, 38}}, {0.45, rgb{52, 124, 52}}, {0.8, rgb{92, 166, 70}}, {1, rgb{140, 196, 96}}}
	forestRamp = []stop{{0, rgb{22, 58, 46}}, {1, rgb{52, 104, 70}}}
	withered   = rgb{112, 116, 72} // 잎이 성길 때 섞는 시든 빛
	woodRamp   = []stop{{0, rgb{58, 40, 30}}, {1, rgb{116, 86, 60}}}
	bareWood   = rgb{86, 76, 70} // 뒤쪽 숲의 앙상한 가지
	grassRamp  = []stop{{0, rgb{36, 96, 44}}, {1, rgb{76, 150, 62}}}
	dryRamp    = []stop{{0, rgb{96, 78, 50}}, {1, rgb{138, 116, 76}}}
	pondBed    = rgb{74, 60, 44}
	rockRamp   = []stop{{0, rgb{58, 58, 62}}, {1, rgb{112, 108, 104}}}
	moss       = rgb{62, 110, 64}
	pondWater  = []stop{{0, rgb{92, 162, 196}}, {0.35, rgb{46, 112, 168}}, {1, rgb{26, 74, 136}}}
	fallRamp   = []stop{{0, rgb{64, 132, 186}}, {1, rgb{226, 242, 250}}}
	sunColor   = rgb{255, 226, 140}
	moonColor  = rgb{232, 234, 244}
	daySky     = rgb{104, 156, 212}
	haze       = rgb{150, 156, 160}
	groveCloud = []stop{{0, rgb{150, 160, 176}}, {1, rgb{236, 240, 246}}}
)

var groveSkins = []string{"night", "day"}

const (
	drift = 0.8 // 구름이 가는 빠르기 (점/초)
	pour  = 3.2 // 폭포의 결이 내려가는 빠르기 (잡음 칸/초)
)

var cloudSteps = []float64{0.3, 0.45, 0.6, 0.75} // 구름마다 나타나는 문턱

type Grove struct{}

func (Grove) Name() string    { return "grove" }
func (Grove) Local() string   { return "free" }
func (Grove) Skins() []string { return groveSkins }

// 점 하나를 두께 있는 선분에 그린다
func stroke(W, H int, put func(x, y int, s, side float64), x1, y1, x2, y2, th float64) {
	pad := th/2 + 1
	xa := int(math.Max(0, math.Floor(math.Min(x1, x2)-pad)))
	xb := int(math.Min(float64(W-1), math.Ceil(math.Max(x1, x2)+pad)))
	ya := int(math.Max(0, math.Floor(math.Min(y1, y2)-pad)))
	yb := int(math.Min(float64(H-1), math.Ceil(math.Max(y1, y2)+pad)))
	dx, dy := x2-x1, y2-y1
	len2 := dx*dx + dy*dy
	if len2 == 0 {
		len2 = 1
	}
	for y := ya; y <= yb; y++ {
		for x := xa; x <= xb; x++ {
			fx, fy := float64(x), float64(y)
			s := clamp(((fx+0.5-x1)*dx + (fy+0.5-y1)*dy) / len2)
			ex := fx + 0.5 - (x1 + dx*s)
			ey := fy + 0.5 - (y1 + dy*s)
			r := lerp(th, th*0.7, s)/2 + 0.3
			if ex*ex+ey*ey <= r*r {
				put(x, y, s, ex/r)
			}
		}
	}
}

type pond struct{ x, y, rx, ry float64 }

type groveLayout struct {
	W, H                     int
	gy                       int // 땅이 시작하는 줄
	wide                     bool
	wood                     []uint8 // 큰 나무의 줄기와 가지 (low 의 글자)
	woodShade                []float32
	leafNeed, leafShade      []float32 // 이 값보다 차면 잎이 있다
	forestNeed, forestShade  []float32
	forestBare               []uint8
	rock                     []float32 // 바위의 결, 바위가 아니면 -1
	groundNeed, groundShade  []float32
	tuft, tuftNeed           []float32
	fallX, fallTop, fallHalf float64
	pond                     pond
}

var groveCache *groveLayout

func filled(n int, v float32) []float32 {
	s := make([]float32, n)
	for i := range s {
		s[i] = v
	}
	return s
}

// 화면 크기로만 정해지는 것들. 크기가 바뀔 때만 다시 셈한다.
func groveLayoutOf(W, H int) *groveLayout {
	if c := groveCache; c != nil && c.W == W && c.H == H {
		return c
	}
	n := W * H
	fW, fH := float64(W), float64(H)
	L := &groveLayout{
		W: W, H: H, gy: int(round(fH * 0.78)), wide: W >= 80,
		wood: make([]uint8, n), woodShade: make([]float32, n),
		leafNeed: filled(n, 9), leafShade: make([]float32, n),
		forestNeed: filled(n, 9), forestShade: make([]float32, n), forestBare: make([]uint8, n),
		rock: filled(n, -1), groundNeed: make([]float32, n), groundShade: make([]float32, n),
	}
	gy := float64(L.gy)
	at := func(x, y int) int { return y*W + x }

	// 바위 벼랑과 폭포 (넓은 창에서만). 벼랑은 높게 세워 오른쪽 위 구석까지 닿게 한다.
	if L.wide {
		x0 := fW - round(fW*0.24)
		top := func(x float64) float64 {
			return gy - fH*(0.42+0.24*smooth(x0, fW, x)) + (nz(x*0.3, 1.7)-0.5)*fH*0.05
		}
		for y := 0; y < L.gy; y++ {
			fy := float64(y)
			edge := x0 + (gy-fy)*0.15 + (nz(fy*0.4, 4.2)-0.5)*3
			for x := int(math.Max(0, math.Floor(edge))); x < W; x++ {
				if fy >= top(float64(x)) {
					L.rock[at(x, y)] = float32(fbm(float64(x)*0.25, fy*0.18))
				}
			}
		}
		L.fallX = fW - fW*0.24*0.45
		L.fallTop = top(L.fallX) + 1
		L.fallHalf = math.Max(1.5, fW*0.022)
	}
	// 연못: 넓으면 폭포 아래, 좁으면 오른쪽
	if L.wide {
		L.pond = pond{L.fallX - fW*0.05, gy + (fH-gy)*0.5, fW * 0.15, math.Max(1.5, (fH-gy)*0.4)}
	} else {
		L.pond = pond{fW * 0.72, gy + (fH-gy)*0.5, fW * 0.17, math.Max(1.5, (fH-gy)*0.4)}
	}

	// 땅: 군데군데 마르는 문턱과 결
	for y := L.gy; y < H; y++ {
		for x := 0; x < W; x++ {
			fx, fy := float64(x), float64(y)
			L.groundNeed[at(x, y)] = float32(0.08 + 0.84*fbm(fx*0.07+3.1, fy*0.3))
			L.groundShade[at(x, y)] = float32(nz(fx*0.5, fy*0.9))
		}
	}
	L.tuft = make([]float32, W) // 풀포기의 높이(점)
	L.tuftNeed = make([]float32, W)
	for x := 0; x < W; x++ {
		fx := float64(x)
		L.tuft[x] = float32(math.Max(0, nz(fx*0.8, 3.1)*4-1))
		L.tuftNeed[x] = float32(0.25 + 0.6*nz(fx*0.21, 8.8))
	}

	// 뒤쪽 숲: 둥근 나무들
	sp := math.Max(4, fH*0.11)
	bare := func(x, y int, _, _ float64) {
		if x >= 0 && y >= 0 && x < W && y < H {
			L.forestBare[at(x, y)] = 1
		}
	}
	for i := 0.0; i*sp < fW+sp; i++ {
		fx := i*sp + (rnd(i, 1)-0.5)*sp*0.6
		r := math.Max(2, fH*(0.06+0.035*rnd(i, 2)))
		fy := gy - r*1.3 - rnd(i, 3)*fH*0.05
		need := 0.08 + 0.84*rnd(i, 4)
		stroke(W, H, bare, fx, gy, fx, fy, 1)
		stroke(W, H, bare, fx, fy+r*0.3, fx-r*0.6, fy-r*0.5, 0.8)
		stroke(W, H, bare, fx, fy+r*0.1, fx+r*0.55, fy-r*0.6, 0.8)
		for y := math.Max(0, math.Floor(fy-r)); y <= math.Min(gy-1, fy+r); y++ {
			for x := math.Max(0, math.Floor(fx-r)); x <= math.Min(fW-1, fx+r); x++ {
				d := math.Hypot(x+0.5-fx, (y+0.5-fy)*1.1) / r
				if d > 1 {
					continue
				}
				k := math.Max(0.03, need+(nz(x*0.7, y*0.7)-0.5)*0.2+0.25*smooth(0.7, 1, d))
				i := at(int(x), int(y))
				if k < float64(L.forestNeed[i]) {
					L.forestNeed[i] = float32(k)
					L.forestShade[i] = float32(clamp(0.55 + (x-fx-(y-fy))/(3*r) + (nz(x*0.9, y)-0.5)*0.3))
				}
			}
		}
	}

	// 큰 나무: 줄기에서 두 갈래씩 네 번 갈라진다
	tx := fW * 0.3
	if L.wide {
		tx = fW * 0.27
	}
	type branch struct{ x, y, x2, y2, th, ang float64 }
	type tip struct{ x, y float64 }
	var tips []tip
	var branches []branch
	var grow func(x, y, ang, length, th float64, depth int, id float64)
	grow = func(x, y, ang, length, th float64, depth int, id float64) {
		x2 := x + math.Sin(ang)*length
		y2 := y - math.Cos(ang)*length
		branches = append(branches, branch{x, y, x2, y2, th, ang})
		if depth <= 1 {
			tips = append(tips, tip{x2, y2})
		}
		if depth == 0 {
			return
		}
		spread := 0.62
		if depth == 4 {
			spread = 0.5
		}
		for k := 0.0; k < 2; k++ {
			a := ang + (k-0.5)*spread*2 + (rnd(id, k+7)-0.5)*0.35
			grow(x2, y2, a*0.92, length*(0.66+0.16*rnd(id, k+3)), th*0.66, depth-1, id*2+k+1)
		}
	}
	grow(tx, gy+1, 0.04, gy*0.27, math.Max(1.6, fH*0.04), 4, 1)
	for _, b := range branches {
		ch := uint8(124) // |
		if math.Abs(b.ang) >= 0.3 {
			if b.ang > 0 {
				ch = 47 // /
			} else {
				ch = 92 // \
			}
		}
		stroke(W, H, func(x, y int, _, side float64) {
			L.wood[at(x, y)] = ch
			L.woodShade[at(x, y)] = float32(clamp(0.45 + side*0.4 + (nz(float64(x)*0.8, float64(y)*0.4)-0.5)*0.3))
		}, b.x, b.y, b.x2, b.y2, b.th)
	}
	// 잎 덩이: 가지 끝마다 셋. 덩이마다 남는 문턱이 다르다.
	r0 := math.Max(2.2, fH*0.075)
	for ti, tp := range tips {
		i := float64(ti)
		for k := 0.0; k < 3; k++ {
			r := r0 * (0.75 + 0.5*rnd(i, k+11))
			cx := tp.x + (rnd(i, k+21)-0.5)*r0*1.4
			cy := tp.y + (rnd(i, k+31)-0.6)*r0*1.2
			need := 0.04 + 0.9*rnd(i*3+k, 41)
			for y := math.Max(0, math.Floor(cy-r)); y <= math.Min(fH-1, cy+r); y++ {
				for x := math.Max(0, math.Floor(cx-r)); x <= math.Min(fW-1, cx+r); x++ {
					d := math.Hypot(x+0.5-cx, y+0.5-cy) / r
					if d > 1 {
						continue
					}
					kk := math.Max(0.03, need+(nz(x*0.65, y*0.65)-0.5)*0.25+0.3*smooth(0.65, 1, d))
					idx := at(int(x), int(y))
					if kk < float64(L.leafNeed[idx]) {
						L.leafNeed[idx] = float32(kk)
						// 오른쪽 위에서 빛이 든다
						L.leafShade[idx] = float32(clamp(0.5 + (x-cx-(y-cy))/(2.6*r) + (nz(x*1.1, y*1.1)-0.5)*0.45))
					}
				}
			}
		}
	}
	groveCache = L
	return L
}

type groveCloudAt struct{ x, y, w, show float64 }

type groveCtx struct {
	noBox
	W, H                       int
	t, f                       float64
	mood, skin                 string
	L                          *groveLayout
	clouds                     []groveCloudAt
	sun                        float64
	sunX, sunY, sunR           float64
	fallHalf, pondFill, wither float64
	glitch                     []glitch.Line
}

func (Grove) Frame(in render.Input) render.Ctx {
	v := first(in.Values)
	mood := moodOf(v.State)
	f := 0.0
	if mood != "fault" {
		f = level(v)
	}
	L := groveLayoutOf(in.W, in.H)
	t := 0.0
	if mood == "live" {
		t = in.Time
	}
	W, H := float64(in.W), float64(in.H)
	c := &groveCtx{W: in.W, H: in.H, t: t, f: f, mood: mood, L: L, skin: groveSkins[0]}
	for _, s := range groveSkins {
		if s == in.Skin {
			c.skin = s
		}
	}
	// 구름: 문턱을 넘은 것만 보인다
	for ci, th := range cloudSteps {
		i := float64(ci)
		show := 0.0
		if mood == "live" {
			show = smooth(th, th+0.12, f)
		}
		if show <= 0 {
			continue
		}
		w := math.Max(6, H*(0.24+0.14*rnd(i, 1)))
		span := W + w*2
		x := mod(mod(rnd(i, 2)*span+t*drift*(0.7+0.5*rnd(i, 3)), span)+span, span) - w
		c.clouds = append(c.clouds, groveCloudAt{x, H * (0.15 + 0.12*rnd(i, 4)), w, show})
	}
	if mood == "live" {
		c.sun = smooth(0.5, 0.85, f)
	}
	// 넓은 창에서는 벼랑이 오른쪽 위를 차지하므로 해와 달을 조금 왼쪽에 둔다
	c.sunX = W * 0.8
	if L.wide {
		c.sunX = W * 0.6
	}
	c.sunY, c.sunR = H*0.14, math.Max(2, H*0.065)
	// 밤의 달은 해처럼 넉넉할수록 크다. 쉼에도 보인다(멈춘 채로).
	if c.skin == "night" && mood == "rest" {
		c.sun = smooth(0.5, 0.85, f)
	}
	if L.wide {
		c.fallHalf = L.fallHalf * math.Pow(f, 0.7)
	}
	c.pondFill = math.Pow(f, 0.8)
	c.wither = (1 - f) * 0.45 // 시든 정도: 잎이 성길수록 빛이 바랜다
	if mood == "live" {
		// 범위 아래(under)로 내려가면 깨진 모니터 같은 줄이 간다
		c.glitch = glitch.Make(smooth(0, 0.1, or0(v.Under)), in.W, in.H, t, in.GlitchSeed)
	}
	return c
}

func (c *groveCtx) Glitch() []glitch.Line { return c.glitch }

// 점 하나의 모습. glyph 가 0 이면 low 에서 비워 둔다(낮의 하늘). lit 이면 스스로 빛난다(해, 달).
type groveLook struct {
	col   rgb
	glyph rune
	tone  string
	lit   bool
}

func (c *groveCtx) look(xi, yi int) (groveLook, bool) {
	L, f, t := c.L, c.f, c.t
	x, y := float64(xi), float64(yi)
	i := yi*c.W + xi
	if yi >= L.gy {
		p := L.pond
		dx, dy := (x+0.5-p.x)/p.rx, (y+0.5-p.y)/p.ry
		d := dx*dx + dy*dy
		if d < c.pondFill {
			// 폭포가 떨어지는 자리의 물거품
			if c.fallHalf > 0.3 && math.Abs(x+0.5-L.fallX) < c.fallHalf+1.5 && yi < L.gy+2 {
				fm := nz(x*0.9, y*0.8-t*pour)
				return groveLook{mix(pondWater[0].Col, fallRamp[1].Col, 0.4+0.6*fm), '*', "bright", false}, true
			}
			if nz(x*0.45+t*0.5, y*1.4+2.1) > 0.8 {
				return groveLook{rgb{170, 214, 234}, '~', "cyan", false}, true
			}
			return groveLook{ramp(pondWater, clamp(1-d/c.pondFill+0.15*(y-p.y)/p.ry)), '~', "blue", false}, true
		}
		if d < 1 {
			// 빈 연못 바닥. 마르면 갈라진다.
			if math.Abs(nz(x*0.35, y*0.8)-0.5) < 0.05 {
				return groveLook{scale(pondBed, 0.7), ',', "dim", false}, true
			}
			return groveLook{pondBed, '_', "dim", false}, true
		}
		s := float64(L.groundShade[i])
		if f > float64(L.groundNeed[i]) {
			g := '.'
			if s > 0.6 {
				g = ','
			}
			return groveLook{ramp(grassRamp, s), g, "green", false}, true
		}
		return groveLook{ramp(dryRamp, s), '.', "yellow", false}, true
	}
	// 큰 나무: 잎이 가지를 덮는다
	if f > float64(L.leafNeed[i]) {
		sh := float64(L.leafShade[i])
		g := '%'
		if sh > 0.6 {
			g = '&'
		}
		return groveLook{mix(ramp(leafRamp, sh), withered, c.wither), g, "green", false}, true
	}
	if L.wood[i] != 0 {
		return groveLook{ramp(woodRamp, float64(L.woodShade[i])), rune(L.wood[i]), "yellow", false}, true
	}
	// 풀포기
	gy := float64(L.gy)
	if y >= gy-3 && f > float64(L.tuftNeed[xi]) && gy-y <= float64(L.tuft[xi])*f {
		return groveLook{ramp(grassRamp, 0.7), '"', "green", false}, true
	}
	// 폭포와 벼랑
	if c.fallHalf > 0.3 && math.Abs(x+0.5-L.fallX) < c.fallHalf && y >= L.fallTop {
		s := nz(x*0.9, y*0.16-t*pour)
		if s > 0.6 {
			return groveLook{ramp(fallRamp, s*s), '!', "bright", false}, true
		}
		return groveLook{ramp(fallRamp, s*s), '|', "cyan", false}, true
	}
	if r := float64(L.rock[i]); r >= 0 {
		col := ramp(rockRamp, r)
		// 이끼: 넉넉할수록 바위 위쪽에 낀다
		ms := f * smooth(0.55, 0.75, nz(x*0.4, y*0.5+9))
		if ms > 0.4 {
			return groveLook{mix(col, moss, 0.7), ':', "green", false}, true
		}
		return groveLook{col, '#', "dim", false}, true
	}
	// 뒤쪽 숲
	if f > float64(L.forestNeed[i]) {
		return groveLook{mix(ramp(forestRamp, float64(L.forestShade[i])), withered, c.wither*0.6), 'o', "green", false}, true
	}
	if L.forestBare[i] != 0 {
		return groveLook{bareWood, '|', "dim", false}, true
	}
	// 하늘: 구름과 해
	for _, cl := range c.clouds {
		u := (x + 0.5 - cl.x) / cl.w
		w := (y + 0.5 - cl.y) / cl.w
		if u < -0.6 || u > 0.6 || w > 0.12 {
			continue
		}
		if math.Hypot(u+0.28, w-0.02) < 0.22 || math.Hypot(u, w+0.08) < 0.3 || math.Hypot(u-0.3, w) < 0.2 {
			return groveLook{scale(ramp(groveCloud, clamp(0.4-w*2.5)), 0.35+0.65*cl.show), '=', "bright", false}, true
		}
	}
	if c.sun > 0 {
		r := c.sunR * (0.35 + 0.65*c.sun)
		// 해와 달은 어두워지지 않고 작아진다
		if math.Hypot(x+0.5-c.sunX, y+0.5-c.sunY) < r {
			if c.skin == "day" {
				return groveLook{scale(sunColor, 0.8+0.2*c.sun), 'O', "yellow", true}, true
			}
			// 초승달: 조금 비켜 겹친 원을 빼낸다
			if math.Hypot(x+0.5-c.sunX-r*0.45, y+0.5-c.sunY+r*0.2) >= r*0.85 {
				return groveLook{moonColor, 'C', "bright", true}, true
			}
		}
	}
	if c.skin == "day" {
		return groveLook{mix(daySky, haze, c.wither*1.4), 0, "", false}, true
	}
	return groveLook{}, false
}

// 밤빛, 안개, 잿빛
func nightTint(c rgb) rgb { return rgb{c[0]*0.55 + 4, c[1]*0.6 + 6, c[2]*0.72 + 18} }

func fade(c rgb, k float64) rgb {
	g := c[0]*0.3 + c[1]*0.55 + c[2]*0.15
	m := mix(c, rgb{g, g, g * 1.08}, k)
	return rgb{m[0] * (1 - 0.25*k), m[1] * (1 - 0.25*k), m[2] * (1 - 0.25*k)}
}

func grey(c rgb) rgb {
	g := (c[0]*0.3 + c[1]*0.55 + c[2]*0.15) * 0.7
	return rgb{g, g, g}
}

func (c *groveCtx) Pixel(x, y int) (rgb, bool) {
	got, ok := c.look(x, y)
	if !ok {
		return rgb{}, false
	}
	if c.mood == "fault" {
		return grey(got.col), true
	}
	col := got.col
	if c.skin == "night" && !got.lit {
		col = nightTint(col)
	}
	if c.mood == "rest" {
		return fade(col, 0.6), true
	}
	return col, true
}

func (c *groveCtx) Glyph(cx, cy int) (rune, string, bool) {
	got, ok := c.look(cx, cy*2)
	if !ok || got.glyph == 0 {
		got, ok = c.look(cx, cy*2+1)
	}
	if !ok || got.glyph == 0 {
		return 0, "", false
	}
	switch c.mood {
	case "live":
		return got.glyph, got.tone, true
	case "rest":
		return got.glyph, "dim", true
	}
	return got.glyph, "grey", true
}

// 밤하늘의 별: 자리가 정해져 있고 움직이지 않는다. 맑을수록(넉넉할수록) 많다.
func (c *groveCtx) Dots(add render.Dot) {
	if c.skin != "night" || c.mood == "fault" {
		return
	}
	W, H := float64(c.W), float64(c.H)
	n := int(round(W * H * 0.006))
	for i := 0; i < n; i++ {
		fi := float64(i)
		if c.f < rnd(fi, 71)*0.9 {
			continue
		}
		x := rnd(fi, 72) * W
		y := rnd(fi, 73) * float64(c.L.gy) * 0.75
		r := rnd(fi, 74)
		b := 0.4 + 0.6*r*r
		col := rgb{200*b + 40, 206*b + 40, 230*b + 25}
		if c.mood == "rest" {
			col = fade(col, 0.6)
		}
		add(x, y, col, b, false)
	}
}
