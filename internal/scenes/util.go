// Package scenes 는 tidetop 의 장면들이다. 장면은 (값, 상태, 칸 수, 줄 수, 시각) → 화면인 순수한 함수다.
// 같은 입력이면 같은 화면이고, 움직임은 시각의 함수다. 무작위 함수를 쓰지 않고, 빠르기에 값을 곱하지 않는다.
package scenes

import (
	"github.com/tidepark/tidetop/internal/feed"
	"github.com/tidepark/tidetop/internal/glitch"
	"github.com/tidepark/tidetop/internal/noise"
	"github.com/tidepark/tidetop/internal/render"
)

var (
	clamp   = noise.Clamp
	clampTo = noise.ClampTo
	lerp    = noise.Lerp
	smooth  = noise.Smooth
	rnd     = noise.Rnd
	nz      = noise.Noise
	fbm     = noise.Fbm
	ramp    = noise.Ramp
	mix     = noise.Mix
	scale   = noise.Scale
	round   = noise.Round
	mod     = noise.Mod
)

type rgb = noise.RGB
type stop = noise.Stop

// level 은 값이 찬 정도(0~1)다. 범위가 없는 값은 다른 컴패니언처럼 원래 숫자를 0~1 로 자른다.
func level(v feed.Computed) float64 {
	if v.Fill != nil {
		return *v.Fill
	}
	return clamp(v.Raw.Value)
}

func or0(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// first 는 첫째 값이다. 없으면 빈 값이다(상태가 없어 쉼으로 그린다).
func first(values []feed.Computed) feed.Computed {
	if len(values) == 0 {
		return feed.Computed{}
	}
	return values[0]
}

// moodOf: 고장, 정상, 그 밖(쉼·끝남·모름)
func moodOf(state string) string {
	switch state {
	case "fault":
		return "fault"
	case "live":
		return "live"
	}
	return "rest"
}

// 모든 장면이 같이 쓰는 빈 구현
type noDots struct{}

func (noDots) Dots(render.Dot) {}

type noGlitch struct{}

func (noGlitch) Glitch() []glitch.Line { return nil }

type noBox struct{}

func (noBox) Box() *render.Box { return nil }
