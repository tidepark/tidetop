package scenes

// 도서관. 값 하나를 읽는다. 값은 「여유」다: 높을수록 넉넉하다.
// 이 컴퓨터의 값을 쓸 때는 쓰지 않는 CPU 의 비율(100 − 사용률)을 준다.
// tidepic 의 도서관과 맺음과 색을 같이 쓰고, 그림은 터미널의 점에 맞게 따로 그린다.
//
// 맺음
//   - 2층 서가의 단면이다. 아래층과 위층에 책장이 늘어서 있고, 위층 마루 앞에 난간이 있다. 아래층에서
//     위층으로 사다리와 계단(오른쪽, 오른쪽 위로 오름)이 있고, 천장 가운데에 촛불 샹들리에가 걸려 있다.
//     아래층 왼쪽에 책상과 등, 사다리와 계단 사이에 안락의자가 있다.
//   - 채움(fill)이 책의 수와 빛이다. 100 은 책이 가득하고 금빛, 50 은 책이 절반이고 호박빛,
//     0 은 빈 서가에 남청빛이다. 0 에서도 기둥과 난간의 윤곽은 남는다.
//   - 도는 서가: 책 자리는 고정이고, 자리마다 빌려 나갔다 돌아오는 주기(한 시간)가 있다. 주기의 앞쪽
//     fill 만큼만 꽂혀 있으므로 꽂힌 수는 값을 따르고, 값이 그대로여도 이따금 한 권이 나가고 다른 자리에
//     한 권이 들어온다. 값이 내려가면 가장 오래 꽂힌 책부터 나간다. 가득 찼을 때도 한 권이 25초쯤 비었다가
//     돌아온다. 책의 색과 높이는 몇 번째로 돌아온 책인지로 정하므로, 빈 동안 바뀌어 다른 책으로 돌아온다.
//     값이 잠깐 내렸다 오르면 같은 책이 돌아온다.
//     같은 칸에서 이웃한 자리는 주기가 비슷해서 빈 곳 옆의 책이 다음에 나간다(무리 지어 빈다).
//     책은 기댈 이웃이나 옆판이 닿는 거리에 있을 때만 기운다.
//   - 빛: 샹들리에의 초 열 자루가 채움이 0.02 + 순위 × 0.098 을 넘을 때 한 자루씩 켜진다. 빛은 샹들리에에서
//     퍼지고 띠로 나눈다. 띠의 가장자리만 바둑판 점으로 섞는다. 어둠은 남청, 밝음은 호박에서 금빛이다.
//     방 전체의 빛도 켜진 초의 수를 따른다(tidepic 은 채움을 그대로 따른다). 값이 흔들려도 초가 켜지거나
//     꺼질 때만 빛이 바뀌어, 띠의 경계가 기어다니며 화면을 다시 쓰는 일이 준다.
//   - 사람: 걷는 한 사람이 책상 → 책 꺼내기 → 사다리 오르내리기 → 계단 → 위층에서 읽기 → 계단 → 책상을
//     정해진 길로 돈다. 읽는 사람 셋이 채움 0.2, 0.45, 0.7 을 넘을 때 한 명씩 자리에 나타난다
//     (위층 왼쪽, 안락의자, 위층 오른쪽). 0.04 아래는 아무도 없다.
//   - 움직이는 것은 촛불의 깜빡임, 빛 알갱이(채움이 0.5 를 넘을 때, 점자), 사람뿐이고 빠르기는 정해져 있다.
//     값은 개수만 바꾼다.
//   - 범위를 넘으면(over) 갓 위와 서가 밑 바닥에 책이 더 쌓이고, 빛과 빛 알갱이가 조금 는다.
//   - 범위 아래로 내려가면(under) 깨진 모니터처럼 가로세로 줄이 가고 깜빡인다.
//   - 쉼(idle)과 끝남(done)은 문 닫은 밤의 도서관이다. 책은 값대로 남고, 샹들리에는 꺼지고, 책상 등만 켜져
//     둘레가 따뜻하다. 책이 남아 있어 0(빈 서가)과 갈린다.
//   - 고장(fault)은 잿빛이고 책 몇 무더기가 바닥에 쏟아져 있다. 서가의 책은 tidepic 처럼 값대로 둔다.
//   - 쉼, 끝남, 고장은 사람이 없고 멈춰 있다. 도는 서가도 멈춘다(시각 0 의 서가를 그린다).
//   - 끊김은 그리기가 잿빛을 덮는다.
//
// 터미널에 맞춘 것: 점 하나가 반 칸이다. 책은 폭 1 점, 높이 3~5 점이고, 칸 하나는 책 5 점에 선반 1 점,
// 책장 한 칸 폭은 기둥 1 점에 안쪽 10 점, 사람은 2 × 4 점이다.
// 줄이 넉넉하면 위층을 더 쌓고(둘까지), 그래도 천장이 넓으면 위층 서가에 칸을 더한다(맨 위층부터).
// 줄이 모자라면 층마다 칸을 하나로 줄이고, 더 모자라면(코어마다 나눠 볼 때의 작은 조각) 서가와 빛만 남긴다.
// 너비가 40 점보다 작으면 책상, 사다리, 계단, 안락의자, 걷는 사람, 샹들리에를 뺀다. 읽는 사람의 수는 남긴다.
//
// 그리는 법: 서가, 기둥, 난간, 사다리, 계단, 가구, 샹들리에는 크기마다 한 번 그려 두고 점마다 앞뒤 겹을 매긴다.
// 책과 빛은 바뀔 때만(값이 그대로면 몇 초에 한 번) 다시 입힌다. 장마다 촛불, 사람, 빛 알갱이만 얹는다.
// JavaScript 판과 셈의 차례를 맞췄다. 위치와 반올림에 닿는 곱셈은 float64() 로 감싸
// 곱셈과 덧셈이 한 번에 묶이지(FMA) 않게 한다.

import (
	"math"

	"github.com/tidepark/tidetop/internal/glitch"
	"github.com/tidepark/tidetop/internal/render"
)

// 낮의 바탕색(tidepic 의 도서관과 같다). 빛은 나중에 한꺼번에 입힌다.
var (
	libCeiling      = rgb{62, 38, 24}
	libFrieze       = rgb{90, 56, 38}
	libBack         = rgb{46, 27, 17}
	libBackShade    = rgb{36, 21, 13}
	libPost         = rgb{110, 66, 38}
	libPostDark     = rgb{78, 46, 26}
	libBoard        = rgb{122, 74, 42}
	libBoardLight   = rgb{160, 106, 60}
	libCornice      = rgb{140, 88, 50}
	libCorniceLight = rgb{176, 122, 69}
	libPlinth       = rgb{90, 54, 31}
	libFloor        = rgb{107, 68, 40}
	libFloorSeam    = rgb{85, 52, 32}
	libBeamTrim     = rgb{192, 136, 72}
	libRail         = rgb{140, 88, 50}
	libRailLight    = rgb{176, 122, 69}
	libLadder       = rgb{160, 106, 60}
	libLadderDark   = rgb{122, 74, 42}
	libTread        = rgb{168, 112, 63}
	libStringer     = rgb{90, 54, 31}
	libLeather      = rgb{122, 46, 42}
	libLeatherDark  = rgb{98, 37, 34}
	libLeatherLight = rgb{146, 55, 50}
	libLampShade    = rgb{62, 122, 74}
	libLampWarm     = rgb{255, 230, 160}
	libGilt         = rgb{226, 192, 112}
	libPages        = rgb{232, 220, 192}
	libDust         = rgb{255, 214, 140}
	libOutline      = rgb{30, 18, 11}
	libFlame        = [3]rgb{{255, 244, 192}, {255, 208, 96}, {255, 154, 48}}
	libBrass        = map[byte]rgb{'g': {200, 154, 74}, 'G': {138, 99, 40}, 'y': {240, 208, 128}, 'c': {243, 234, 208}, 'r': {207, 224, 240}}
)

// 글자 그림의 빛깔 번호
const (
	tNone uint8 = iota
	tDim
	tRed
	tGreen
	tYellow
	tBlue
	tMagenta
	tCyan
	tBright
	tGrey
)

var libToneNames = []string{"", "dim", "red", "green", "yellow", "blue", "magenta", "cyan", "bright", "grey"}

// 책의 색과 글자 그림의 빛깔
type libBook struct {
	col  rgb
	tone uint8
}

var libBooks = []libBook{
	{rgb{181, 64, 47}, tRed}, {rgb{126, 42, 46}, tRed}, {rgb{62, 138, 74}, tGreen}, {rgb{44, 94, 58}, tGreen},
	{rgb{58, 95, 158}, tBlue}, {rgb{44, 63, 110}, tBlue}, {rgb{47, 138, 134}, tCyan}, {rgb{209, 163, 58}, tYellow},
	{rgb{200, 105, 46}, tYellow}, {rgb{138, 90, 52}, tYellow}, {rgb{181, 64, 47}, tRed}, {rgb{62, 138, 74}, tGreen},
	{rgb{58, 95, 158}, tBlue}, {rgb{227, 211, 168}, tBright}, {rgb{94, 74, 126}, tMagenta}, {rgb{47, 138, 134}, tCyan},
}

// 사람. 걷는 사람과 읽는 사람 셋은 머리색과 옷색만 다르다.
type libPalette map[byte]rgb

var libWalker = libPalette{'h': {91, 54, 36}, 's': {242, 200, 162}, 'c': {233, 220, 192}, 'C': {199, 182, 148}, 'p': {61, 79, 115}, 'w': {244, 236, 216}, 'W': {255, 252, 240}}

func (p libPalette) with(h, c, C, pp rgb) libPalette {
	out := libPalette{}
	for k, v := range p {
		out[k] = v
	}
	out['h'], out['c'], out['C'], out['p'] = h, c, C, pp
	return out
}

type libReader struct {
	at      float64
	where   string
	palette libPalette
}

var libReaders = []libReader{
	{0.2, "left", libWalker.with(rgb{42, 36, 34}, rgb{79, 122, 90}, rgb{61, 96, 74}, rgb{90, 70, 54})},
	{0.45, "chair", libWalker.with(rgb{31, 26, 24}, rgb{201, 166, 74}, rgb{168, 135, 58}, rgb{58, 58, 72})},
	{0.7, "right", libWalker.with(rgb{176, 136, 80}, rgb{58, 74, 106}, rgb{44, 58, 85}, rgb{106, 90, 72})},
}

// 그림의 몸은 왼쪽 두 열이다. 셋째 열은 바라보는 쪽으로 나간다(뒤집으면 왼쪽으로).
var libSprites = map[string][]string{
	"walkA":    {"hs", "cc", "cC", "pp"},
	"walkB":    {"hs", "cc", "cC", "p.p"},
	"reach":    {"hss", "cc", "cC", "pp"},
	"climbA":   {"hh", "cc", "Cc", "pp"},
	"climbB":   {"hh", "cc", "cC", "pp"},
	"read":     {"hs", "cw", "cC", "pp"},
	"read2":    {"hs", "cW", "cC", "pp"},
	"sit":      {"hs", "cw", "cC"},
	"armchair": {"hs", "cw"},
}

// 샹들리에: 위 단에 초 넷, 아래 단에 초 여섯. 불꽃은 초의 바로 위 점이다(켜질 때만).
var libChandelier = []string{
	".....G.....",
	"..c.cGc.c..",
	"..ggggggg..",
	".....G.....",
	"c.c.cGc.c.c",
	"ggggggggggg",
	"..yyyyyyy..",
	"....r.r....",
}

var libCandles = [][2]int{{2, 1}, {4, 1}, {6, 1}, {8, 1}, {0, 4}, {2, 4}, {4, 4}, {6, 4}, {8, 4}, {10, 4}}

// 켜지는 순서. tidepic 처럼 rnd(초 번호, 41) 의 차례로 정해 두었다.
var libCandleOrder = []int{8, 5, 7, 9, 1, 4, 2, 0, 6, 3}

const (
	libChW      = 11
	libChH      = 8
	libFloorH   = 3  // 아래층 마루
	libPlinthH  = 1  // 서가 밑동
	libRow      = 6  // 칸 하나: 책 5 + 선반 1
	libBeamH    = 3  // 위층 마루의 들보
	libRailH    = 2  // 난간의 높이(마루 위). 사람(4 점)의 머리와 가슴이 난간 위로 보인다
	libCorniceH = 2  // 맨 위 서가의 갓
	libBay      = 11 // 책장 한 칸: 기둥 1 + 안쪽 10
	libInner    = libBay - 1
	libTreadW   = 2    // 계단 디딤판의 폭
	libCycle    = 3600 // 한 자리의 책이 빌려 나갔다 돌아오는 한 바퀴(초)
	libBands    = 7    // 빛의 띠
	libWalkV    = 3    // 걷는 빠르기 (점/초)
	libClimbV   = 1.6
	libStairsV  = 2
	libFlicker  = 4 // 촛불이 깜빡이는 빠르기 (번/초)
)

const libAway = 25.0 / libCycle // 가득 찼을 때도 이만큼은 비었다가 다른 책으로 돌아온다

var libBayer = func() [16]float64 {
	var out [16]float64
	for i, v := range []float64{0, 8, 2, 10, 12, 4, 14, 6, 3, 11, 1, 9, 15, 7, 13, 5} {
		out[i] = (v + 0.5) / 16
	}
	return out
}()

// 앞뒤 겹. 큰 쪽이 앞이다. 사람은 홀수 겹이라 구조와 겹치지 않는다.
const (
	zRoom       uint8 = 0
	zBook       uint8 = 1
	zPile       uint8 = 2
	zUpper      uint8 = 3
	zRail       uint8 = 4
	zLadder     uint8 = 6
	zSeated     uint8 = 7
	zChairFront uint8 = 8
	zWalker     uint8 = 9
	zDesk       uint8 = 10
	zChandelier uint8 = 12
)

// 글자 그림에서 한 칸의 두 점 가운데 무엇을 보일지: 큰 쪽이 이긴다
func libRank(ch uint8) int {
	switch ch {
	case 0:
		return 0
	case 'o':
		return 9
	case '*':
		return 8
	case 'i':
		return 7
	case '-', '+', 'H', '/', '\\':
		return 6
	case '_':
		return 5
	case '|', '=':
		return 4
	case '#':
		return 3
	case ':':
		return 2
	}
	return 1
}

type Library struct{}

func (Library) Name() string    { return "library" }
func (Library) Local() string   { return "free" }
func (Library) Skins() []string { return nil }

// 한 장의 판: 점마다 색, 앞뒤 겹, 글자, 빛깔
type libSheet struct {
	W, H        int
	col         []float64
	z, ch, tone []uint8
}

func newLibSheet(W, H int) *libSheet {
	n := W * H
	return &libSheet{W: W, H: H, col: make([]float64, n*3), z: make([]uint8, n), ch: make([]uint8, n), tone: make([]uint8, n)}
}

// 점 하나를 칠한다. 앞에 있는 것(겹이 더 큰 것)은 덮지 않는다.
func (g *libSheet) put(x, y int, col rgb, z uint8, ch byte, tone uint8) {
	if x < 0 || y < 0 || x >= g.W || y >= g.H {
		return
	}
	i := y*g.W + x
	if g.z[i] > z {
		return
	}
	g.z[i] = z
	g.col[i*3], g.col[i*3+1], g.col[i*3+2] = col[0], col[1], col[2]
	g.ch[i] = ch
	g.tone[i] = tone
}

// ── 책 ───────────────────────────────────────────────

type libSlot struct {
	stack bool
	x, w  int
	u, id float64
}

// 칸 하나의 자리를 왼쪽부터 늘어놓는다. 자리와 폭은 언제나 같고, 꽂히는 책의 모습은 바퀴마다 바뀐다.
// 주기의 시작점(u)은 칸을 따라 비스듬히 오르내리고(1.2~2.5 바퀴) ±0.06 쯤 흩어 두어,
// 이웃한 자리가 비슷한 때에 나가고 들어온다. 눕혀 쌓은 책(폭 3)도 한 자리다.
func libRowSlots(seed float64) []libSlot {
	var slots []libSlot
	base := rnd(seed, 1)
	turns := 1.2 + float64(rnd(seed, 2)*1.3)
	if rnd(seed, 3) < 0.5 {
		turns = -turns
	}
	x, k := 0, 10.0
	for x < libInner {
		r := rnd(seed, k)
		k++
		if r < 0.07 && x > 0 {
			x++
			continue
		}
		stack := r < 0.15 && libInner-x >= 4
		w := 1
		if stack {
			w = 3
		}
		u := base + turns*(float64(x)+float64(w)/2)/libInner + float64(0.12*(rnd(seed, k)-0.5))
		k++
		slots = append(slots, libSlot{stack, x, w, u - math.Floor(u), seed*16 + float64(len(slots))})
		x += w
		if stack {
			x++
		}
	}
	return slots
}

// 그 자리에 몇 번째로 돌아온 책인지(gen)에 따라 모습이 정해진다
func libLookSeed(id float64, gen int32) float64 { return id + float64(gen)*7919 }

func libBookHeight(id float64, gen int32) int {
	return 3 + int(math.Floor(rnd(libLookSeed(id, gen), 1)*3))
}

func libLeans(id float64, gen int32) bool { return rnd(libLookSeed(id, gen), 5) < 0.7 }

func libPaintBook(g *libSheet, x, base int, id float64, gen int32, lean int) {
	s := libLookSeed(id, gen)
	h := 3 + int(math.Floor(rnd(s, 1)*3))
	b := libBooks[int(math.Floor(rnd(s, 2)*float64(len(libBooks))))]
	band := rnd(s, 3) < 0.45
	for j := 0; j < h; j++ {
		dx := 0
		if lean != 0 {
			dx = int(round(float64(lean*j) / float64(h-1)))
		}
		col := b.col
		if band && j == h-2 {
			col = libGilt
		} else if j == h-1 {
			col = scale(b.col, 0.82)
		}
		g.put(x+dx, base-j, col, zBook, '|', b.tone)
	}
}

// 눕혀 쌓은 책: 한 권이 한 점 높이다
func libPaintStack(g *libSheet, x, base, w int, id float64, gen int32) {
	s := libLookSeed(id, gen)
	count := 1 + int(math.Floor(rnd(s, 6)*3))
	for j := 0; j < count; j++ {
		fj := float64(j)
		length := w
		if rnd(s, 7+fj) < 0.4 {
			length--
		}
		b := libBooks[int(math.Floor(rnd(s, 11+fj)*float64(len(libBooks))))]
		off := (j % 2) * (w - length)
		for k := 0; k < length; k++ {
			col := b.col
			if k == length-1 {
				col = libPages
			}
			g.put(x+off+k, base-j, col, zBook, '=', b.tone)
		}
	}
}

// 갓 위나 바닥에 눕혀 쌓은 책 한 권
func libLying(g *libSheet, x, y, length int, col rgb, tone, z uint8, pagesAt int) {
	for k := 0; k < length; k++ {
		c := col
		if k == pagesAt {
			c = libPages
		}
		g.put(x+k, y, c, z, '=', tone)
	}
}

// ── 구조 ─────────────────────────────────────────────

type libShelfRow struct {
	x, base int
	slots   []libSlot
}

type libUpperFloor struct{ walk, top, rows int }

type libLadderAt struct{ top, bottom, x, lean int }

type libLamp struct{ x, y, r float64 }

type libSeg struct {
	kind       string
	x0, x1, y  int
	y0, y1     int
	up         bool
	dur, start float64
}

type libLayout struct {
	W, H                                 int
	g                                    *libSheet
	rows                                 []libShelfRow
	n, x0                                int
	room, furnished                      bool
	floorTop, lowerTop, walk, corniceTop int
	uppers                               []libUpperFloor
	chX, chY                             int
	lightX, lightY, rx, ry               float64
	steps, run, stairX                   int
	ladderX, deskX, chairX               int
	ladder                               libLadderAt
	lamp                                 *libLamp
	shape, lampShape                     []float64
	slots                                int
	base                                 *libBase
	route                                []libSeg
	routeTotal                           float64
	// 장마다 얹는 점(촛불, 사람). 같은 장 안에서만 쓰는 표(stamp)로 지운다.
	ovStamp []int32
	ovZ     []uint8
	ovCol   []float64
	ovCh    []uint8
	ovTone  []uint8
	stamps  int32
}

// 책장: 기둥, 칸의 뒤판, 선반, 밑동. 칸마다 책 자리를 적어 둔다.
func (L *libLayout) shelves(top, rows, floor int) {
	g := L.g
	for b := 0; b < L.n; b++ {
		ix := L.x0 + b*libBay + 1
		for r := 0; r < rows; r++ {
			y := top + r*libRow
			for x := ix; x < ix+libInner; x++ {
				g.put(x, y, libBackShade, zRoom, 0, tNone)
				for yy := y + 1; yy < y+libRow-1; yy++ {
					g.put(x, yy, libBack, zRoom, 0, tNone)
				}
				g.put(x, y+libRow-1, libBoardLight, zRoom, '_', tYellow)
			}
			seed := b*97 + floor*31 + r*7 + 3
			if floor >= 2 {
				seed += 5000 * floor
			}
			L.rows = append(L.rows, libShelfRow{ix, y + libRow - 2, libRowSlots(float64(seed))})
		}
		for x := ix; x < ix+libInner; x++ {
			for y := top + rows*libRow; y < top+rows*libRow+libPlinthH; y++ {
				g.put(x, y, libPlinth, zRoom, 0, tNone)
			}
		}
	}
	for b := 0; b <= L.n; b++ {
		px := L.x0 + b*libBay
		for y := top; y < top+rows*libRow+libPlinthH; y++ {
			g.put(px, y, libPost, zRoom, '#', tYellow)
		}
	}
}

func (L *libLayout) beam(y int) {
	for x := 0; x < L.W; x++ {
		L.g.put(x, y, libRailLight, zRoom, 0, tNone)
		L.g.put(x, y+1, libBeamTrim, zRoom, '=', tYellow)
		L.g.put(x, y+2, libPostDark, zRoom, 0, tNone)
	}
}

// 난간: 위에 가로대, 두 점마다 동자, 서가 기둥과 맞춘 기둥
func (L *libLayout) railing(walk, end int) {
	g := L.g
	top := walk - libRailH
	for x := 0; x < end; x++ {
		spindle := x%2 == 0
		ch := byte('-')
		if spindle {
			ch = '+'
		}
		g.put(x, top, libRailLight, zRail, ch, tYellow)
		if spindle {
			for y := top + 1; y < walk; y++ {
				g.put(x, y, libRail, zRail, '|', tYellow)
			}
		}
	}
	for b := 0; b <= L.n; b++ {
		px := L.x0 + b*libBay
		if px < 0 || px >= end {
			continue
		}
		for y := top - 1; y < walk; y++ {
			col, ch := libPost, byte('|')
			if y == top-1 {
				col = libRailLight
			}
			if y < top+1 {
				ch = '+'
			}
			g.put(px, y, col, zRail, ch, tYellow)
		}
	}
}

func (L *libLayout) paintLadder(l libLadderAt, z uint8) {
	length := l.bottom - l.top
	for y := l.top; y < l.bottom; y++ {
		dx := int(round(float64(l.lean*(l.bottom-1-y)) / float64(length)))
		L.g.put(l.x+dx, y, libLadder, z, 'H', tYellow)
		L.g.put(l.x+dx+3, y, libLadderDark, z, 0, tNone)
		if (l.bottom-y)%3 == 2 {
			L.g.put(l.x+dx+1, y, libRailLight, z, 0, tNone)
			L.g.put(l.x+dx+2, y, libRailLight, z, 0, tNone)
		}
	}
}

// 계단의 디딤판 높이와, 디딤판 끝을 잇는 비스듬한 선
func (L *libLayout) stairTop(x int) int {
	i := int(math.Floor(float64(x-L.stairX) / libTreadW))
	return int(round(float64(L.floorTop) - float64((i+1)*(L.floorTop-L.walk))/float64(L.steps)))
}

func (L *libLayout) stairLine(x int) int {
	return int(round(float64(L.floorTop) - float64(float64(x-L.stairX+1)/libTreadW*(float64(L.floorTop-L.walk)/float64(L.steps)))))
}

// 그 열에서 발이 닿는 높이
func (L *libLayout) standY(x int) int {
	if x < L.stairX {
		return L.floorTop
	}
	if x >= L.stairX+L.run {
		return L.walk
	}
	return L.stairTop(x)
}

// 계단은 디딤판과 옆판만 그리고 아래는 비워 서가가 보이게 한다
func (L *libLayout) stairs() {
	for x := L.stairX; x < L.stairX+L.run && x < L.W; x++ {
		y := L.stairTop(x)
		line := max(y+1, L.stairLine(x))
		L.g.put(x, y, libTread, zLadder, '/', tYellow)
		for yy := y + 1; yy <= line; yy++ {
			col := libLadderDark
			if (x-L.stairX)%libTreadW == 0 {
				col = libPostDark
			}
			L.g.put(x, yy, col, zLadder, 0, tNone)
		}
		L.g.put(x, line+1, libStringer, zLadder, 0, tNone)
	}
}

// 계단 손잡이: 디딤판 선을 따라가는 가로대와 디딤판마다 동자, 아래 기둥
func (L *libLayout) banister() {
	g := L.g
	for x := L.stairX - 1; x < L.stairX+L.run && x < L.W; x++ {
		y := L.stairLine(x) - libRailH - 1
		g.put(x, y, libRailLight, zDesk, '/', tYellow)
		if x >= L.stairX && (x-L.stairX)%libTreadW == 1 {
			for yy := y + 1; yy < L.stairTop(x); yy++ {
				g.put(x, yy, libRail, zDesk, 0, tNone)
			}
		}
	}
	nx := L.stairX - 1
	for y := L.stairLine(nx) - libRailH - 2; y < L.floorTop; y++ {
		g.put(nx, y, libPost, zDesk, '|', tYellow)
	}
}

func (L *libLayout) desk() {
	g := L.g
	x := L.deskX
	y := L.floorTop - 3
	for k := 0; k < 7; k++ {
		g.put(x+k, y, libBoardLight, zDesk, '=', tYellow)
		g.put(x+k, y+1, libBoard, zDesk, 0, tNone)
	}
	g.put(x+1, y+2, libPostDark, zDesk, '|', tYellow)
	g.put(x+5, y+2, libPostDark, zDesk, '|', tYellow)
	// 등: 녹색 갓과 놋쇠 대
	g.put(x+4, y-2, libLampShade, zDesk, '^', tGreen)
	g.put(x+5, y-2, libLampShade, zDesk, '^', tGreen)
	g.put(x+5, y-1, libBrass['g'], zDesk, '|', tYellow)
	// 책상 의자의 등받이(사람이 자리를 비워도 남는다)
	for yy := L.floorTop - 5; yy < L.floorTop; yy++ {
		g.put(x, yy, libPostDark, zLadder, '|', tYellow)
	}
}

// 안락의자: 등받이는 앉은 사람 뒤, 앉는 자리와 팔걸이는 앞
func (L *libLayout) armchair() {
	g := L.g
	cx := L.chairX
	y := L.floorTop
	for yy := y - 5; yy < y-1; yy++ {
		col := libLeatherDark
		if yy == y-5 {
			col = libLeatherLight
		}
		g.put(cx+1, yy, col, zLadder, '#', tRed)
	}
	for x := cx - 2; x <= cx+1; x++ {
		col := libLeather
		if x == cx-2 {
			col = libLeatherLight
		}
		g.put(x, y-2, col, zChairFront, '=', tRed)
	}
	g.put(cx-2, y-3, libLeatherLight, zChairFront, '=', tRed)
	g.put(cx-2, y-1, libPostDark, zChairFront, 0, tNone)
	g.put(cx+1, y-1, libPostDark, zChairFront, 0, tNone)
}

func (L *libLayout) chandelier() {
	cx := L.chX + 5
	for y := 0; y < L.chY; y++ {
		col := libBrass['G']
		if y%2 != 0 {
			col = libBrass['g']
		}
		L.g.put(cx, y, col, zChandelier, '|', tYellow)
	}
	for j, row := range libChandelier {
		for k := 0; k < libChW; k++ {
			key := row[k]
			if key == '.' {
				continue
			}
			ch := byte('-')
			switch key {
			case 'c':
				ch = 'i'
			case 'G':
				ch = '|'
			case 'r':
				ch = ':'
			}
			tone := tYellow
			if key == 'c' || key == 'r' {
				tone = tBright
			}
			L.g.put(L.chX+k, L.chY+j, libBrass[key], zChandelier, ch, tone)
		}
	}
}

var libCache *libLayout

// 화면 크기로만 정해지는 것들. 크기가 바뀔 때만 다시 셈한다.
func libLayoutOf(W, H int) *libLayout {
	if c := libCache; c != nil && c.W == W && c.H == H {
		return c
	}
	fW, fH := float64(W), float64(H)
	L := &libLayout{W: W, H: H, g: newLibSheet(W, H), n: int(math.Ceil(float64(W-1)/libBay)) + 1}
	L.x0 = int(math.Floor(float64(W-(L.n*libBay+1)) / 2))
	g := L.g
	// 줄이 넉넉하면 층마다 칸 둘, 모자라면 하나, 더 모자라면 방 없이 서가만
	rpf := 0
	if H >= 40 {
		rpf = 2
	} else if H >= 22 {
		rpf = 1
	}
	L.room = rpf > 0
	L.furnished = L.room && W >= 40
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			g.put(x, y, libCeiling, zRoom, 0, tNone)
		}
	}

	if !L.room {
		rows := int(math.Ceil(float64(H-libPlinthH) / libRow))
		L.shelves(H-libPlinthH-rows*libRow, rows, 0)
		L.lightX, L.lightY = fW/2, fH*0.2
		L.rx = math.Max(fH, fW*0.5) * 1.2
		L.ry = L.rx * 0.9375
	} else {
		block := func(rows int) int { return libPlinthH + rows*libRow }
		extra := 0
		if rpf == 2 {
			extra = int(math.Max(0, math.Min(2, math.Floor(float64(H-48)/24))))
		}
		used := libFloorH + block(rpf) + (extra+1)*(libBeamH+block(rpf)) + libCorniceH
		add := 0
		if rpf == 2 {
			add = int(math.Max(0, math.Floor(float64(H-used-18)/libRow)))
		}
		L.floorTop = H - libFloorH
		L.lowerTop = L.floorTop - block(rpf)
		for k, w := 0, L.lowerTop-libBeamH; k <= extra; k++ {
			rows := rpf + add/(extra+1)
			if extra-k < add%(extra+1) {
				rows++
			}
			h := block(rows)
			L.uppers = append(L.uppers, libUpperFloor{w, w - h, rows})
			w -= h + libBeamH
		}
		L.walk = L.uppers[0].walk
		L.corniceTop = L.uppers[extra].top - libCorniceH
		L.chX = int(math.Floor(fW/2 - float64(libChW)/2))
		L.chY = max(0, L.corniceTop-libChH+libCorniceH)
		L.lightX, L.lightY = fW/2, float64(L.chY+4)
		L.rx, L.ry = fH*0.8, fH*0.75
		// 계단(오른쪽), 사다리, 책상, 안락의자의 자리
		L.steps = max(2, int(round(float64(L.floorTop-L.walk)/2.5)))
		L.run = L.steps * libTreadW
		L.stairX = W - L.run - 3
		L.ladderX = int(math.Floor(fW * 0.28))
		L.deskX = max(2, int(math.Floor(fW*0.04)))
		L.chairX = int(round(float64(L.ladderX+5+L.stairX-2) / 2))
		L.ladder = libLadderAt{L.walk - libRailH - 2, L.floorTop, L.ladderX, 1}

		// 천장 아래 띠, 서가, 갓, 마루, 들보
		for x := 0; x < W; x++ {
			g.put(x, L.corniceTop-1, libFrieze, zRoom, 0, tNone)
		}
		for k, u := range L.uppers {
			L.shelves(u.top, u.rows, k+1)
		}
		L.shelves(L.lowerTop, rpf, 0)
		for x := 0; x < W; x++ {
			g.put(x, L.corniceTop, libCorniceLight, zRoom, '_', tYellow)
			g.put(x, L.corniceTop+1, libCornice, zRoom, 0, tNone)
			g.put(x, L.floorTop, libBoardLight, zRoom, '_', tYellow)
			for y := L.floorTop + 1; y < H; y++ {
				col := libFloor
				if (x+y*5)%9 == 0 {
					col = libFloorSeam
				}
				g.put(x, y, col, zRoom, '.', tYellow)
			}
		}
		for _, u := range L.uppers {
			L.beam(u.walk)
		}
		// 난간. 계단이 올라오는 곳은 끊는다. 더 쌓은 층은 아래층 마루에서 사다리로 오른다.
		for k, u := range L.uppers {
			end := W
			if k == 0 && L.furnished {
				end = L.stairX + L.run - libTreadW*2
			}
			L.railing(u.walk, end)
		}
		for k := 1; k < len(L.uppers); k++ {
			f := 0.34
			if k%2 != 0 {
				f = 0.66
			}
			L.paintLadder(libLadderAt{L.uppers[k].walk - libRailH - 2, L.uppers[k-1].walk, int(round(fW * f)), 1}, zRail)
		}
		if L.furnished {
			L.paintLadder(L.ladder, zLadder)
			L.stairs()
			L.banister()
			L.desk()
			L.armchair()
			// 쉼: 책상 등의 갓 아래가 밝다
			L.lamp = &libLamp{float64(L.deskX) + 4.5, float64(L.floorTop - 4), math.Max(5, fH*0.15)}
			// 좁은 창에서는 샹들리에가 위층 사람을 가리므로 빼고, 빛만 위 가운데에서 퍼진다
			L.chandelier()
		}
	}

	// 빛이 닿는 정도: 샹들리에(또는 서가만 있을 때 위쪽 가운데)에서 퍼진다
	n := W * H
	L.shape = make([]float64, n)
	if L.lamp != nil {
		L.lampShape = make([]float64, n)
	}
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			dx := (float64(x) - L.lightX) / L.rx
			dy := (float64(y) - L.lightY) / L.ry
			L.shape[y*W+x] = math.Exp(-(float64(dx*dx) + float64(dy*dy)) * 1.4)
			if L.lamp != nil {
				lx := (float64(x) - L.lamp.x) / L.lamp.r
				ly := (float64(y) - L.lamp.y) / L.lamp.r
				L.lampShape[y*W+x] = math.Exp(-(float64(lx*lx) + float64(ly*ly)))
			}
		}
	}
	for _, row := range L.rows {
		L.slots += len(row.slots)
	}
	L.ovStamp = make([]int32, n)
	L.ovZ = make([]uint8, n)
	L.ovCol = make([]float64, n*3)
	L.ovCh = make([]uint8, n)
	L.ovTone = make([]uint8, n)
	libCache = L
	return L
}

// ── 빛 ───────────────────────────────────────────────

// 어둠은 남청빛, 빛은 호박빛에서 금빛으로(tidepic 과 같은 식). v 는 띠로 나눈 빛이다.
func libLightUp(col rgb, v float64, gray bool) rgb {
	r, g, b := col[0], col[1], col[2]
	lum := (float64(0.3*r) + float64(0.59*g) + float64(0.11*b)) / 255
	mr := float64(r*0.1) + float64(lum*54) + 7
	mg := float64(g*0.1) + float64(lum*74) + 11
	mb := float64(b*0.1) + float64(lum*114) + 24
	wr := r * (0.3 + float64(0.8*v))
	wg := g * (0.24 + float64(0.72*v))
	wb := b * (0.2 + float64(0.55*v))
	k := math.Min(1, v/0.45)
	m := k * k * (3 - 2*k)
	out := rgb{mr + float64((wr-mr)*m), mg + float64((wg-mg)*m), mb + float64((wb-mb)*m)}
	if !gray {
		return out
	}
	l := float64(0.3*out[0]) + float64(0.59*out[1]) + float64(0.11*out[2])
	return rgb{float64(out[0]*0.15) + float64(l*0.85), float64(out[1]*0.15) + float64(l*0.85), (float64(out[2]*0.15) + float64(l*0.85)) * 1.04}
}

// 책 자리마다 지금의 모습: 꽂혀 있으면 몇 번째로 돌아온 책인지, 비어 있으면 -1
func (L *libLayout) shelfState(f, t float64) []int32 {
	open := math.Min(f, 1-libAway)
	out := make([]int32, L.slots)
	i := 0
	for _, row := range L.rows {
		for _, s := range row.slots {
			p := s.u + t/libCycle
			gen := math.Floor(p)
			out[i] = -1
			if p-gen < open {
				out[i] = int32(gen)
			}
			i++
		}
	}
	return out
}

func libSameState(a, b []int32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// 책과 빛을 입힌 판
type libBase struct {
	mood        string
	fq, overq   float64
	lit         int
	state       []int32
	col, vb     []float64
	z, ch, tone []uint8
}

// 책, 쌓인 책, 빛을 입힌 판. 열쇠와 책 자리가 같으면 앞의 것을 다시 쓴다.
func (L *libLayout) baseOf(c *libCtx) *libBase {
	if b := L.base; b != nil && b.mood == c.mood && b.fq == c.fq && b.overq == c.overq && b.lit == c.lit && libSameState(b.state, c.state) {
		return b
	}
	n := L.W * L.H
	g := &libSheet{W: L.W, H: L.H, col: append([]float64(nil), L.g.col...), z: append([]uint8(nil), L.g.z...),
		ch: append([]uint8(nil), L.g.ch...), tone: append([]uint8(nil), L.g.tone...)}

	// 책: 오른쪽 빈틈 너머의 책(없으면 칸의 옆판)에 윗머리가 닿을 때만 그쪽으로 기운다
	i := 0
	for _, row := range L.rows {
		here := c.state[i : i+len(row.slots)]
		i += len(row.slots)
		for k, s := range row.slots {
			gen := here[k]
			if gen < 0 {
				continue
			}
			if s.stack {
				libPaintStack(g, row.x+s.x, row.base, s.w, s.id, gen)
				continue
			}
			next := k + 1
			for next < len(row.slots) && here[next] < 0 {
				next++
			}
			edge := libInner
			if next < len(row.slots) {
				edge = row.slots[next].x
			}
			free := edge - (s.x + s.w)
			reach := min(int(math.Floor(float64(libBookHeight(s.id, gen))*0.45)), 2)
			lean := 0
			if libLeans(s.id, gen) && free >= 1 && free <= reach {
				lean = free
			}
			libPaintBook(g, row.x+s.x, row.base, s.id, gen, lean)
		}
	}
	if L.room {
		// 갓 위에 쌓인 책(거의 다 찼을 때). 범위를 넘으면 칸마다 더 높이 쌓인다.
		for b := 0; b < L.n; b++ {
			fb := float64(b)
			if c.overq == 0 && rnd(fb, 51) < 0.35 {
				continue
			}
			t0 := 0.0
			if c.overq == 0 {
				t0 = 0.82 + float64(0.17*rnd(fb, 52))
			}
			count := 2 + int(round(c.overq*2*(0.5+rnd(fb, 56))))
			x := L.x0 + b*libBay + 2 + int(math.Floor(rnd(fb, 55)*5))
			for j := 0; j < count && t0+float64(float64(j)*0.02) < c.fq; j++ {
				s := fb*7 + float64(j)
				bk := libBooks[int(math.Floor(rnd(s, 54)*float64(len(libBooks))))]
				length := 3
				if rnd(s, 53) < 0.4 {
					length = 2
				}
				libLying(g, x+j%2, L.corniceTop-1-j, length, bk.col, bk.tone, zBook, length-1)
			}
		}
		// 바닥에 쌓인 책. 범위를 넘으면 서가 밑에 가지런히 쌓이고, 고장이면 쏟아져 흩어진다.
		spilled := c.mood == "fault"
		share := math.Min(1, c.overq*1.6)
		if spilled {
			share = 0.55
		}
		for b := 0; b < L.n; b++ {
			fb := float64(b)
			if rnd(fb, 81) >= share {
				continue
			}
			x := L.x0 + b*libBay + 2 + int(math.Floor(rnd(fb, 82)*4))
			count := 1 + int(round(c.overq*3*(0.4+rnd(fb, 83))))
			if spilled {
				count = 1 + int(math.Floor(rnd(fb, 83)*2))
			}
			for j := 0; j < count; j++ {
				s := fb*11 + float64(j)
				length := 3 + int(math.Floor(rnd(s, 84)*2))
				dx := int(math.Floor(rnd(s, 85) * 2))
				if spilled {
					dx = int(math.Floor((rnd(s, 85) - 0.5) * 5))
				}
				bk := libBooks[int(math.Floor(rnd(s, 86)*float64(len(libBooks))))]
				pages := length - 1
				if j%2 != 0 {
					pages = 0
				}
				libLying(g, x+dx, L.floorTop-1-j, length, bk.col, bk.tone, zPile, pages)
			}
			if spilled {
				// 펼쳐진 채 떨어진 책
				ox := x + 5 + int(math.Floor(rnd(fb, 87)*3))
				bk := libBooks[int(math.Floor(rnd(fb, 88)*float64(len(libBooks))))]
				libLying(g, ox, L.floorTop-1, 4, bk.col, bk.tone, zPile, -1)
				libLying(g, ox+1, L.floorTop-2, 2, libPages, tBright, zPile, -1)
			}
		}
	}

	// 빛: 방 전체의 빛(amb) + 샹들리에(켜진 초의 몫) + 책상 등(쉼). 띠로 나누고 가장자리만 바둑판 점으로 섞는다.
	glow := 0.0
	if c.mood == "live" {
		glow = float64(c.lit) / float64(len(libCandles))
	}
	amb := 0.0
	switch c.mood {
	case "fault":
		amb = 0.55
	case "live":
		amb = float64(0.36*glow) + float64(0.1*c.overq)
	}
	lamp := 0.0
	if c.mood == "rest" && L.lampShape != nil {
		lamp = 0.8
	}
	gray := c.mood == "fault"
	vb := make([]float64, n)
	col := make([]float64, n*3)
	for y := 0; y < L.H; y++ {
		for x := 0; x < L.W; x++ {
			p := y*L.W + x
			v := amb + float64(0.95*glow*L.shape[p])
			if lamp != 0 {
				v += float64(lamp * L.lampShape[p])
			}
			v = math.Floor(float64(v*libBands)+0.25+float64(0.5*libBayer[(y&3)*4+(x&3)])) / libBands
			vb[p] = v
			lit := libLightUp(rgb{g.col[p*3], g.col[p*3+1], g.col[p*3+2]}, v, gray)
			col[p*3], col[p*3+1], col[p*3+2] = lit[0], lit[1], lit[2]
		}
	}
	add := func(x, y int, r, gg, b float64) {
		if x < 0 || y < 0 || x >= L.W || y >= L.H {
			return
		}
		p := y*L.W + x
		col[p*3] = math.Min(255, col[p*3]+r)
		col[p*3+1] = math.Min(255, col[p*3+1]+gg)
		col[p*3+2] = math.Min(255, col[p*3+2]+b)
	}
	// 켜진 초 둘레의 따뜻한 빛
	if L.furnished && c.mood == "live" {
		for rank := 0; rank < c.lit; rank++ {
			cd := libCandles[libCandleOrder[rank]]
			fx := L.chX + cd[0]
			fy := L.chY + cd[1] - 1
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					k := 1 - math.Sqrt(float64(dx*dx+dy*dy))/2
					add(fx+dx, fy+dy, 70*k, 44*k, 10*k)
				}
			}
		}
	}
	// 쉼: 책상 등의 불
	if lamp != 0 {
		p := (L.floorTop-4)*L.W + L.deskX + 4
		col[p*3], col[p*3+1], col[p*3+2] = libLampWarm[0], libLampWarm[1], libLampWarm[2]
		g.ch[p] = '*'
		g.tone[p] = tYellow
		g.z[p] = zDesk
		for dx := -1; dx <= 2; dx++ {
			add(L.deskX+4+dx, L.floorTop-3, 50, 36, 10)
		}
	}
	L.base = &libBase{mood: c.mood, fq: c.fq, overq: c.overq, lit: c.lit, state: c.state, col: col, vb: vb, z: g.z, ch: g.ch, tone: g.tone}
	return L.base
}

// ── 사람 ─────────────────────────────────────────────

// 이따금 책장을 넘긴다
func libPageTurn(t, phase float64) bool { return mod(t+phase, 7) < 0.35 }

// 걷는 사람의 길. 자리에서 정해지고, 시각으로 지금 어디쯤인지 고른다.
func (L *libLayout) walkRoute() ([]libSeg, float64) {
	if L.route != nil {
		return L.route, L.routeTotal
	}
	var segs []libSeg
	floor := L.floorTop
	sitX := L.deskX + 1
	reachX := int(round(float64(L.deskX+8+L.ladderX) / 2))
	ladX := L.ladderX + 1
	foot := L.stairX - 3
	top := L.stairX + L.run - 1
	// 위층에서 읽는 자리: 샹들리에와 오른쪽 읽는 사람 사이(작은 창에서 샹들리에가 위층을 덮어도 가리지 않게)
	readX := int(round(float64(L.chX+libChW+int(round(float64(L.W)*0.78))-1)/2)) - 1
	climbH := max(2, int(round(float64(L.floorTop-L.walk)*0.45)))
	walk := func(x0, x1, y int, up bool) {
		segs = append(segs, libSeg{kind: "walk", x0: x0, x1: x1, y: y, up: up, dur: math.Abs(float64(x1-x0)) / libWalkV})
	}
	segs = append(segs, libSeg{kind: "sit", x0: sitX, dur: 12})
	segs = append(segs, libSeg{kind: "stand", x0: sitX, dur: 0.6})
	walk(sitX, reachX, floor, false)
	segs = append(segs, libSeg{kind: "reach", x0: reachX, dur: 2.5})
	walk(reachX, ladX, floor, false)
	segs = append(segs, libSeg{kind: "climb", y0: floor, y1: floor - climbH, dur: float64(climbH) / libClimbV})
	segs = append(segs, libSeg{kind: "climb", y0: floor - climbH, y1: floor - climbH, dur: 3})
	segs = append(segs, libSeg{kind: "climb", y0: floor - climbH, y1: floor, dur: float64(climbH) / libClimbV})
	walk(ladX, foot, floor, false)
	segs = append(segs, libSeg{kind: "stairs", x0: foot, x1: top, dur: float64(top-foot) / libStairsV})
	walk(top, readX, L.walk, true)
	segs = append(segs, libSeg{kind: "read", x0: readX, dur: 12})
	walk(readX, top, L.walk, true)
	segs = append(segs, libSeg{kind: "stairs", x0: top, x1: foot, dur: float64(top-foot) / libStairsV})
	walk(foot, sitX, floor, false)
	t := 0.0
	for i := range segs {
		segs[i].start = t
		t += segs[i].dur
	}
	L.route, L.routeTotal = segs, t
	return segs, t
}

type libWho struct {
	sprite   string
	x, y     int
	flip, up bool
}

// 걷는 사람의 지금 모습: 그림, 몸의 왼쪽 위 자리, 뒤집기, 위층인지
func (L *libLayout) walkerAt(t float64) libWho {
	segs, total := L.walkRoute()
	tt := mod(mod(t, total)+total, total)
	s := segs[len(segs)-1]
	for _, seg := range segs {
		if tt < seg.start+seg.dur {
			s = seg
			break
		}
	}
	p := 0.0
	if s.dur != 0 {
		p = (tt - s.start) / s.dur
	}
	floor := L.floorTop
	switch s.kind {
	case "sit":
		return libWho{sprite: "sit", x: s.x0, y: floor - 5}
	case "stand":
		return libWho{sprite: "walkA", x: s.x0, y: floor - 4}
	case "reach":
		return libWho{sprite: "reach", x: s.x0, y: floor - 4}
	case "read":
		name := "read"
		if libPageTurn(t, 0) {
			name = "read2"
		}
		return libWho{sprite: name, x: s.x0, y: L.walk - 4, up: true}
	case "walk", "stairs":
		x := int(round(float64(s.x0) + float64(float64(s.x1-s.x0)*p)))
		feet := s.y
		if s.kind == "stairs" {
			feet = min(L.standY(x), L.standY(x+1))
		}
		name := "walkA"
		if x%2 != 0 {
			name = "walkB"
		}
		return libWho{sprite: name, x: x, y: feet - 4, flip: s.x1 < s.x0, up: s.up}
	}
	// 사다리
	lad := L.ladder
	fy := int(round(float64(s.y0) + float64(float64(s.y1-s.y0)*p)))
	dx := int(round(float64(lad.lean*(lad.bottom-1-fy)) / float64(lad.bottom-lad.top)))
	name := "climbA"
	if fy%2 != 0 {
		name = "climbB"
	}
	return libWho{sprite: name, x: lad.x + dx + 1, y: fy - 4}
}

type libCtx struct {
	noBox
	W, H      int
	t, f      float64
	mood      string
	L         *libLayout
	lit       int
	fq, overq float64
	state     []int32
	glitch    []glitch.Line
	base      *libBase
	stamp     int32
}

func (c *libCtx) Glitch() []glitch.Line { return c.glitch }

// 장마다 얹는 점(촛불, 사람)
func (c *libCtx) overlay(i int, col rgb, z uint8, ch byte, tone uint8) {
	L := c.L
	if L.ovStamp[i] == c.stamp && L.ovZ[i] > z {
		return
	}
	L.ovStamp[i] = c.stamp
	L.ovZ[i] = z
	L.ovCol[i*3], L.ovCol[i*3+1], L.ovCol[i*3+2] = col[0], col[1], col[2]
	L.ovCh[i] = ch
	L.ovTone[i] = tone
}

// 사람 하나를 얹는다. 앞에 있는 구조(겹이 더 큰 것)에 가린다.
// 붐비는 책장 앞에서도 보이게 둘레(발밑은 빼고)를 어둡게 두른다(tidepic 과 같다).
func (c *libCtx) person(name string, x, y int, flip bool, palette libPalette, z uint8) {
	rows := libSprites[name]
	at := func(k, j int) bool {
		return j >= 0 && j < len(rows) && k >= 0 && k < len(rows[j]) && rows[j][k] != '.'
	}
	place := func(k, j int, col rgb, ch byte) {
		px := x + k
		if flip {
			px = x + 1 - k
		}
		py := y + j
		if px < 0 || py < 0 || px >= c.W || py >= c.H {
			return
		}
		i := py*c.W + px
		if c.base.z[i] > z {
			return
		}
		c.overlay(i, libLightUp(col, c.base.vb[i], c.mood == "fault"), z, ch, tBright)
	}
	for j := -1; j < len(rows); j++ {
		for k := -1; k <= 3; k++ {
			if !at(k, j) && (at(k-1, j) || at(k+1, j) || at(k, j-1) || at(k, j+1)) {
				place(k, j, libOutline, 0)
			}
		}
	}
	for j, row := range rows {
		for k := 0; k < len(row); k++ {
			if row[k] == '.' {
				continue
			}
			// 글자 그림에서는 머리 한 점만 o 로 보인다
			ch := byte(0)
			if j == 0 && k == 0 {
				ch = 'o'
			}
			place(k, j, palette[row[k]], ch)
		}
	}
}

func (Library) Frame(in render.Input) render.Ctx {
	v := first(in.Values)
	mood := moodOf(v.State)
	f := level(v)
	L := libLayoutOf(in.W, in.H)
	t, over := 0.0, 0.0
	if mood == "live" {
		t = in.Time
		over = or0(v.Over)
	}
	lit := 0
	if mood == "live" {
		for rank := 0; rank < len(libCandles); rank++ {
			if f > 0.02+float64(float64(rank)*0.098) {
				lit++
			}
		}
	}
	c := &libCtx{
		W: in.W, H: in.H, t: t, mood: mood, f: f, L: L, lit: lit,
		fq:    round(f*50) / 50,
		overq: round(over*20) / 20,
		state: L.shelfState(f, t),
	}
	if mood == "live" {
		c.glitch = glitch.Make(smooth(0, 0.1, or0(v.Under)), in.W, in.H, t, in.GlitchSeed)
	}
	c.base = L.baseOf(c)
	// 표의 번호는 아주 오래 돌면 넘치므로 2^30 에서 표를 비우고 1 부터 다시 센다
	if L.stamps >= 1<<30 {
		clear(L.ovStamp)
		L.stamps = 0
	}
	L.stamps++
	c.stamp = L.stamps
	if mood != "live" || !L.room {
		return c
	}

	// 촛불: 켜진 초마다 불꽃 한 점이 깜빡인다
	if L.furnished {
		for rank := 0; rank < lit; rank++ {
			k := libCandleOrder[rank]
			cd := libCandles[k]
			phase := int(math.Floor(float64(t*libFlicker)+float64(rnd(float64(k), 61)*7))) % 5
			y := L.chY + cd[1] - 1
			if y < 0 {
				continue
			}
			col := libFlame[1]
			if phase == 4 {
				col = libFlame[0]
			} else if phase == 0 {
				col = libFlame[2]
			}
			c.overlay(y*in.W+L.chX+cd[0], col, zChandelier+1, '*', tYellow)
		}
	}
	if f < 0.04 {
		return c
	}
	// 읽는 사람 셋: 위층 왼쪽, 안락의자, 위층 오른쪽
	fW := float64(in.W)
	for k, r := range libReaders {
		if f <= r.at {
			continue
		}
		name := "read"
		if libPageTurn(t, float64(2.3*float64(k+1))) {
			name = "read2"
		}
		switch {
		case r.where == "chair" && L.furnished:
			c.person("armchair", L.chairX-1, L.floorTop-4, true, r.palette, zSeated)
		case r.where == "chair":
			c.person(name, int(round(fW*0.5))-1, L.floorTop-4, true, r.palette, zWalker)
		default:
			share := 0.78
			if r.where == "left" {
				share = 0.2
			}
			c.person(name, int(round(fW*share))-1, L.walk-4, r.where == "right", r.palette, zUpper)
		}
	}
	if L.furnished {
		who := L.walkerAt(t)
		z := zWalker
		if who.up {
			z = zUpper
		}
		c.person(who.sprite, who.x, who.y, who.flip, libWalker, z)
	}
	return c
}

// 점 하나의 색: 얹은 점이 있으면 그것, 없으면 책과 빛을 입힌 판의 색
func (c *libCtx) Pixel(x, y int) (rgb, bool) {
	i := y*c.W + x
	from := c.base.col
	if c.L.ovStamp[i] == c.stamp {
		from = c.L.ovCol
	}
	return rgb{from[i*3], from[i*3+1], from[i*3+2]}, true
}

// 글자 그림: 한 칸의 두 점 가운데 앞세울 글자가 큰 쪽을 쓴다. 어두운 곳은 남청(blue)이다.
func (c *libCtx) dotGlyph(i int) (uint8, uint8, bool) {
	if c.L.ovStamp[i] == c.stamp {
		return c.L.ovCh[i], c.L.ovTone[i], true
	}
	return c.base.ch[i], c.base.tone[i], false
}

func (c *libCtx) Glyph(cx, cy int) (rune, string, bool) {
	ia := cy*2*c.W + cx
	ib := (cy*2+1)*c.W + cx
	cha, tonea, lita := c.dotGlyph(ia)
	var chb, toneb uint8
	var litb bool
	if cy*2+1 < c.H {
		chb, toneb, litb = c.dotGlyph(ib)
	}
	ra, rb := libRank(cha), libRank(chb)
	if ra == 0 && rb == 0 {
		return 0, "", false
	}
	ch, tone, lit, i := cha, tonea, lita, ia
	if ra < rb {
		ch, tone, lit, i = chb, toneb, litb, ib
	}
	switch c.mood {
	case "fault":
		return rune(ch), "grey", true
	case "rest":
		if ch == '*' {
			return rune(ch), "yellow", true
		}
		return rune(ch), "dim", true
	}
	// 얹은 점(촛불, 사람)과 밝은 곳은 제 빛깔, 어두운 곳은 남청
	if lit || c.base.vb[i] >= 0.3 {
		if libToneNames[tone] == "" {
			return rune(ch), "yellow", true
		}
		return rune(ch), libToneNames[tone], true
	}
	return rune(ch), "blue", true
}

// 빛 알갱이: 채움이 0.5 를 넘으면 샹들리에 빛 속에 떠서 천천히 내려온다
func (c *libCtx) Dots(add render.Dot) {
	if c.mood != "live" || !c.L.room {
		return
	}
	amount := c.f + float64(0.5*c.overq)
	if amount < 0.5 {
		return
	}
	L := c.L
	fW, fH := float64(c.W), float64(c.H)
	n := int(math.Floor((amount - 0.5) * 2 * math.Max(4, round(fW*fH/320))))
	span := float64(L.floorTop - L.chY - 4)
	for j := 0; j < n; j++ {
		fj := float64(j)
		sp := 0.3 + float64(rnd(fj, 71)*0.4)
		y := mod(float64(rnd(fj, 72)*span)+float64(c.t*sp), span) + float64(L.chY+4)
		x := L.lightX + float64((rnd(fj, 73)-0.5)*fH*1.1) + float64(math.Sin(float64(c.t*0.4)+fj)*1.2)
		tw := math.Sin(float64(c.t*(1+rnd(fj, 74))) + float64(fj*2.1))
		if tw < -0.2 {
			continue
		}
		b := 0.6 + float64(0.4*tw)
		add(x, y, rgb{libDust[0] * b, libDust[1] * b, libDust[2] * b}, b, true)
	}
}
