// Package glitch 는 값이 범위 아래로 내려갔을 때(under) 깨진 모니터처럼 가는 줄이다.
//
//   - 못 미친 만큼 줄이 는다. 가로줄과 세로줄이 여러 곳에 생기고, 색은 시안, 마젠타, 노랑, 흰빛이다.
//   - 줄마다 깜빡이는 빠르기와 자리를 옮기는 주기가 정해져 있다. 값은 줄의 수만 바꾼다.
//   - 줄은 한 점 두께이고 화면 끝까지 가지 않는 조각이다.
//   - 두 줄이 만나는 점은 흰빛으로 튄다(흰 줄이 끼면 시안). 줄의 끝 2~3점은 점선처럼 흩어진다.
package glitch

import (
	"math"

	"github.com/tidepark/tidetop/internal/noise"
)

// Line 은 줄 하나다. 자리는 점 단위다.
type Line struct {
	Across     bool // 가로줄
	Pos        float64
	Start, End float64
	Width      float64
	Seed       float64
	Color      noise.RGB
	Tone       string
	Short      bool // 두 줄이 만나는 점
}

var (
	white = Line{Color: noise.RGB{255, 255, 255}, Tone: "bright"}
	short = Line{Color: noise.RGB{0, 255, 255}, Tone: "cyan"}
	inks  = []Line{
		{Color: noise.RGB{0, 236, 236}, Tone: "cyan"},
		{Color: noise.RGB{236, 0, 236}, Tone: "magenta"},
		{Color: noise.RGB{236, 236, 0}, Tone: "yellow"},
		{Color: noise.RGB{236, 236, 236}, Tone: "bright"},
	}
)

const fray = 3 // 끝에서 흩어지는 점의 수

var rnd = noise.Rnd

// Make 는 줄들을 만든다. amount 는 0~1 이고 아주 작으면(0.01 이하) 줄이 없다. W, H 는 점 단위다.
// 값은 부드럽게 따라가며 0 에 한없이 가까워지기만 하므로, 잠깐 범위 아래로 내려갔다 올라온 뒤에
// 남는 아주 작은 값으로 줄이 계속 서지 않게 문턱을 둔다.
// seed 는 이어 붙인 창의 조각 번호다. 같은 때 같은 값이어도 조각마다 줄의 배치가 다르다.
func Make(amount float64, W, H int, t float64, seed int) []Line {
	if !(amount > 0.01) {
		return nil
	}
	n := int(math.Max(1, noise.Round(amount*(1+float64(W+H)/24))))
	var lines []Line
	for i := 0; i < n; i++ {
		j := float64(i + seed*97)
		period := 2.2 + 2*rnd(j, 51) // 자리를 옮기는 주기 (초)
		shift := rnd(j, 52) * period
		s := math.Floor((t+shift)/period)*31 + j
		rate := 0.7 + 1.3*rnd(j, 54) // 깜빡이는 빠르기 (번/초)
		if noise.Mod(t*rate+rnd(j, 55), 1) > 0.62 {
			continue // 꺼진 동안
		}
		across := rnd(s, 53) < 0.6
		ink := inks[int(math.Floor(rnd(s, 56)*float64(len(inks))))]
		span := float64(W)
		if !across {
			span = float64(H)
		}
		length := span * (0.3 + 0.7*rnd(s, 59))
		start := math.Floor(rnd(s, 60) * (span - length + 1))
		cross := float64(H)
		if !across {
			cross = float64(W)
		}
		width := 1.0
		if rnd(s, 58) < 0.25 {
			width = 2
		}
		lines = append(lines, Line{
			Across: across, Pos: math.Floor(rnd(s, 57) * cross), Start: start, End: start + length,
			Width: width, Seed: s, Color: ink.Color, Tone: ink.Tone,
		})
	}
	return lines
}

// covers: 줄이 점 (x, y) 를 지나는가. 끝 밖의 몇 점은 띄엄띄엄 지난다.
func covers(l Line, x, y float64) bool {
	p, q := x, y
	if l.Across {
		p, q = y, x
	}
	if p < l.Pos || p >= l.Pos+l.Width {
		return false
	}
	if q >= l.Start && q < l.End {
		return true
	}
	out := q - l.End + 1 // 끝에서 벗어난 점 수
	if q < l.Start {
		out = l.Start - q
	}
	return out <= fray && rnd(l.Seed*7+q, 61) < 0.75-out*0.2
}

// At 은 점 (x, y) 를 지나는 줄이다. 두 줄이 만나는 점이면 흰빛(흰 줄이 끼면 시안)이다.
func At(lines []Line, x, y float64) (Line, bool) {
	var hit Line
	found := false
	for _, l := range lines {
		if !covers(l, x, y) {
			continue
		}
		if !found {
			hit, found = l, true
			continue
		}
		mark := white
		if hit.Tone == "bright" || l.Tone == "bright" {
			mark = short
		}
		return Line{Color: mark.Color, Tone: mark.Tone, Across: hit.Across, Short: true}, true
	}
	return hit, found
}
