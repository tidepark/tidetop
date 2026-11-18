// Package noise 는 순서로 정해지는 난수와 부드러운 잡음이다.
// 같은 입력이면 같은 값이다. 무작위 함수를 쓰지 않는다.
package noise

import (
	"math"

	"github.com/tidepark/tidetop/internal/jsmath"
)

func frac(x float64) float64 { return x - math.Floor(x) }

// Rnd 는 Tidepark 컴패니언과 같은 식이다.
func Rnd(i, k float64) float64 { return frac(jsmath.Sin(i*127.1+k*311.7) * 43758.5453) }

// Clamp 는 0~1 로 자른다.
func Clamp(x float64) float64 { return ClampTo(x, 0, 1) }

// ClampTo 는 lo~hi 로 자른다(JavaScript 의 Math.min(hi, Math.max(lo, x)) 와 같다).
func ClampTo(x, lo, hi float64) float64 { return math.Min(hi, math.Max(lo, x)) }

func Lerp(a, b, t float64) float64 { return a + (b-a)*t }

func Smooth(a, b, x float64) float64 {
	t := Clamp((x - a) / (b - a))
	return t * t * (3 - 2*t)
}

// toInt32 는 JavaScript 의 ToInt32 다(Math.imul 이 인자에 하는 것).
func toInt32(x float64) uint32 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return 0
	}
	return uint32(int64(math.Trunc(math.Mod(x, 4294967296))))
}

// 격자점의 값. 정수 둘을 섞어 0~1 을 낸다.
func hash(ix, iy float64) float64 {
	h := toInt32(ix)*374761393 ^ toInt32(iy)*668265263
	h = (h ^ (h >> 13)) * 1274126177
	return float64(h^(h>>16)) / 4294967296
}

// Noise 는 0~1 의 부드러운 잡음이다.
func Noise(x, y float64) float64 {
	ix, iy := math.Floor(x), math.Floor(y)
	fx, fy := x-ix, y-iy
	sx := fx * fx * (3 - 2*fx)
	sy := fy * fy * (3 - 2*fy)
	a := hash(ix, iy)
	b := hash(ix+1, iy)
	c := hash(ix, iy+1)
	d := hash(ix+1, iy+1)
	return a + (b-a)*sx + (c-a)*sy + (a-b-c+d)*sx*sy
}

// Fbm 은 결이 두 겹인 잡음(0~1)이다.
func Fbm(x, y float64) float64 {
	return Noise(x, y)*0.65 + Noise(x*2.03+17.3, y*2.03+5.1)*0.35
}

// RGB 는 0~255 의 색이다. 더 크거나 작아도 된다(그릴 때 자른다).
type RGB [3]float64

// Stop 은 색 띠의 한 자리다.
type Stop struct {
	At  float64
	Col RGB
}

// Ramp 는 색 띠다. stops 는 자리의 오름차순이다.
func Ramp(stops []Stop, t float64) RGB {
	if t <= stops[0].At {
		return stops[0].Col
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].At {
			p0, c0 := stops[i-1].At, stops[i-1].Col
			p1, c1 := stops[i].At, stops[i].Col
			k := (t - p0) / (p1 - p0)
			return RGB{Lerp(c0[0], c1[0], k), Lerp(c0[1], c1[1], k), Lerp(c0[2], c1[2], k)}
		}
	}
	return stops[len(stops)-1].Col
}

// Mix 는 두 색 사이다.
func Mix(a, b RGB, k float64) RGB {
	return RGB{Lerp(a[0], b[0], k), Lerp(a[1], b[1], k), Lerp(a[2], b[2], k)}
}

// Scale 은 색의 밝기를 곱한다.
func Scale(a RGB, k float64) RGB { return RGB{a[0] * k, a[1] * k, a[2] * k} }

// Mod 는 JavaScript 의 % 다(부호는 나누어지는 수를 따른다).
func Mod(a, b float64) float64 { return math.Mod(a, b) }

// Round 는 JavaScript 의 Math.round 다.
func Round(x float64) float64 { return jsmath.Round(x) }
