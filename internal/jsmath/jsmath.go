// Package jsmath 는 JavaScript(V8)의 Math 함수와 비트까지 같은 값을 내는 함수들이다.
// 장면은 난수와 잡음을 sin 따위로 만들므로, 마지막 비트가 달라도 화면이 달라진다.
package jsmath

import "math"

func Sin(x float64) float64      { return math.Sin(x) }
func Cos(x float64) float64      { return math.Cos(x) }
func Exp(x float64) float64      { return math.Exp(x) }
func Log(x float64) float64      { return math.Log(x) }
func Pow(x, y float64) float64   { return math.Pow(x, y) }
func Atan2(y, x float64) float64 { return math.Atan2(y, x) }

// Hypot 는 V8 의 Math.hypot 과 같은 식이다: 큰 쪽으로 나누고, 카한 합을 내고, 제곱근에 큰 쪽을 곱한다.
func Hypot(x, y float64) float64 {
	ax, ay := math.Abs(x), math.Abs(y)
	if math.IsInf(ax, 0) || math.IsInf(ay, 0) {
		return math.Inf(1)
	}
	if math.IsNaN(ax) || math.IsNaN(ay) {
		return math.NaN()
	}
	top := math.Max(ax, ay)
	if top == 0 {
		return 0
	}
	sum, comp := 0.0, 0.0
	for _, v := range [2]float64{ax, ay} {
		n := v / top
		summand := float64(n*n) - comp
		pre := sum + summand
		comp = (pre - sum) - summand
		sum = pre
	}
	return float64(math.Sqrt(sum) * top)
}

// Round 는 JavaScript 의 Math.round 다. 절반은 +∞ 쪽으로 올린다.
func Round(x float64) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return x
	}
	r := math.Floor(x)
	if x-r >= 0.5 {
		r++
	}
	if r == 0 && math.Signbit(x) {
		return math.Copysign(0, -1)
	}
	return r
}
